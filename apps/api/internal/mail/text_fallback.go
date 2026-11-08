package mail

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// htmlToText converts an HTML email body to a plain-text fallback.
//
// It is not a general-purpose HTML-to-text converter. It handles what the
// framework's email templates produce: headings, paragraphs, links, and a
// verification-code block. The goal is deliverability (a multipart/alternative
// message passes spam filters that reject HTML-only mail), not pixel-perfect
// rendering in a text client.
func htmlToText(htmlBody string) string {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		// Fall back to stripping tags by regex when the parser fails.
		return collapseSpace(reTag.ReplaceAllString(htmlBody, " "))
	}
	var buf bytes.Buffer
	walkNode(&buf, doc)
	return strings.TrimSpace(collapseSpace(buf.String()))
}

var reTag = regexp.MustCompile(`<[^>]+>`)

// skipTags are elements whose text content is never useful in plain-text.
var skipTags = map[string]bool{
	"head": true, "style": true, "script": true,
}

// blockTags produce a blank line before and after them.
var blockTags = map[string]bool{
	"p": true, "div": true, "tr": true, "br": true,
	"h1": true, "h2": true, "h3": true, "h4": true,
	"table": true, "td": true, "li": true,
}

func walkNode(buf *bytes.Buffer, n *html.Node) {
	if n.Type == html.ElementNode {
		tag := strings.ToLower(n.Data)
		if skipTags[tag] {
			return
		}
		if blockTags[tag] {
			buf.WriteString("\n")
		}
		// For anchor tags, emit the text and the href when useful.
		if tag == "a" {
			var linkBuf bytes.Buffer
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkNode(&linkBuf, c)
			}
			text := strings.TrimSpace(linkBuf.String())
			var href string
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
				}
			}
			if href != "" && !strings.HasPrefix(href, "#") && href != text {
				buf.WriteString(text + " ( " + href + " )")
			} else {
				buf.WriteString(text)
			}
			return
		}
	}
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNode(buf, c)
	}
	if n.Type == html.ElementNode && blockTags[strings.ToLower(n.Data)] {
		buf.WriteString("\n")
	}
}

// collapseSpace replaces runs of whitespace (including newlines) with a single
// space, then restores paragraph breaks (two or more newlines).
func collapseSpace(s string) string {
	// Normalise line endings.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Collapse runs of spaces and tabs on a single line.
	reSpaces := regexp.MustCompile(`[ \t]+`)
	s = reSpaces.ReplaceAllString(s, " ")
	// Collapse more than two consecutive newlines into two.
	reNewlines := regexp.MustCompile(`\n{3,}`)
	s = reNewlines.ReplaceAllString(s, "\n\n")
	// Trim trailing space from each line.
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.Join(lines, "\n")
}
