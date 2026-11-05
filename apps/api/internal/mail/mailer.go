package mail

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Mailer sends email through a Transport. FromConfig picks the transport once
// from configuration, so the code that sends mail does not change when the
// provider does.
type Mailer struct {
	transport Transport
	from      string
}

// New returns a Mailer that sends through Resend. It keeps the signature it
// has always had, so code written against it compiles unchanged. FromConfig is
// the constructor that honours MAIL_MAILER.
func New(apiKey, from string) *Mailer {
	return NewWithTransport(NewResend(apiKey), from)
}

// NewWithTransport returns a Mailer over any transport, including the fake in
// internal/mail/mailtest.
func NewWithTransport(t Transport, from string) *Mailer {
	return &Mailer{transport: t, from: from}
}

// Driver names the transport: "smtp", "resend", "failover(smtp,log)" and so on.
func (m *Mailer) Driver() string {
	return m.transport.Name()
}

// From is the address a message without its own From is sent from.
func (m *Mailer) From() string {
	return m.from
}

// SendOptions configures an email to send.
type SendOptions struct {
	To       string
	Subject  string
	Template string
	Data     map[string]interface{}
}

// Send renders a template from EmailTemplates and sends it.
func (m *Mailer) Send(ctx context.Context, opts SendOptions) error {
	// The subject doubles as the <title> and, unless the caller says otherwise,
	// as the inbox preview line.
	if opts.Data == nil {
		opts.Data = map[string]interface{}{}
	}
	if opts.Data["Subject"] == nil {
		opts.Data["Subject"] = opts.Subject
	}
	if opts.Data["Preheader"] == nil {
		opts.Data["Preheader"] = opts.Subject
	}
	// Inject recipient email and support contact for the footer. These are
	// available here from SendOptions but not inside the caller's data map.
	if opts.Data["RecipientEmail"] == nil {
		opts.Data["RecipientEmail"] = recipientEmail(opts.To)
	}
	if opts.Data["SupportEmail"] == nil {
		opts.Data["SupportEmail"] = ""
	}
	htmlBody, err := m.renderTemplate(opts.Template, opts.Data)
	if err != nil {
		return fmt.Errorf("rendering template %q: %w", opts.Template, err)
	}
	// Always include a text/plain part alongside the HTML. Google Workspace and
	// most spam filters expect multipart/alternative. For verification emails,
	// generate a human-readable plain text with the code prominent, rather
	// than stripping the HTML (which produces a wall of style tokens).
	textBody := buildTextBody(opts)
	if textBody == "" {
		textBody = htmlToText(htmlBody)
	}
	return m.SendMessage(ctx, &Message{To: []string{opts.To}, Subject: opts.Subject, HTML: htmlBody, Text: textBody})
}

// buildTextBody produces a clean plain-text version for templates that carry
// a Code. A stripped version of the full HTML template is hard to read; a
// hand-written plain text is more likely to pass spam filters and render well
// in text-only clients.
func buildTextBody(opts SendOptions) string {
	if opts.Data == nil {
		return ""
	}
	code, hasCode := opts.Data["Code"]
	if !hasCode {
		return ""
	}
	appName := "CampVote"
	if v, ok := opts.Data["AppName"]; ok {
		appName = fmt.Sprintf("%v", v)
	}
	msg := "Enter this code to verify your address."
	if v, ok := opts.Data["Message"]; ok {
		msg = fmt.Sprintf("%v", v)
	}
	minutes := "10"
	if v, ok := opts.Data["Minutes"]; ok {
		minutes = fmt.Sprintf("%v", v)
	}
	to := ""
	if v, ok := opts.Data["RecipientEmail"]; ok {
		to = fmt.Sprintf("%v", v)
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\nYour verification code:\n\n  %v\n\nThis code expires in %s minutes and works once.\n\n--\n%s will never ask for this code by email or phone.\nSent to %s.",
		opts.Subject, msg, code, minutes, appName, to,
	)
}

// recipientEmail pulls a bare address from a To value that may be either
// "Name <addr>" or plain "addr", for use in the footer.
func recipientEmail(to string) string {
	to = strings.TrimSpace(to)
	if i := strings.LastIndex(to, "<"); i >= 0 {
		to = strings.TrimSuffix(strings.TrimPrefix(to[i:], "<"), ">")
	}
	return to
}

// SendRaw sends an email with raw HTML content (no template rendering).
func (m *Mailer) SendRaw(ctx context.Context, to, subject, htmlBody string) error {
	return m.SendMessage(ctx, &Message{To: []string{to}, Subject: subject, HTML: htmlBody})
}

// SendMessage sends a message using any of Message's fields: several
// recipients, cc, bcc, reply-to, a text part, attachments and headers.
func (m *Mailer) SendMessage(ctx context.Context, msg *Message) error {
	if msg == nil {
		return errors.New("mail: nil message")
	}
	out := *msg
	if out.From == "" {
		out.From = m.from
	}
	if err := out.Validate(); err != nil {
		return err
	}
	return m.transport.Send(ctx, &out)
}

// renderTemplate renders a fragment and wraps it in Layout.
//
// Two passes on purpose. The fragment is rendered first so that whatever it
// produces becomes the Content of the shell, which means a template can put a
// button or a code block inside the card without the shell knowing anything
// about it.
func (m *Mailer) renderTemplate(name string, data map[string]interface{}) (string, error) {
	tmplStr, ok := EmailTemplates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}

	full := map[string]interface{}{}
	for k, v := range Style(Theme()) {
		full[k] = v
	}
	for k, v := range data {
		full[k] = v
	}
	if full["Year"] == nil {
		full["Year"] = time.Now().Year()
	}
	// RecipientEmail, Digits, and DigitInk are injected by Send() from the
	// SendOptions rather than from the caller's data map: they are structural
	// properties of every email, not template-specific values.
	// Callers that use renderTemplate directly (RenderLayout, preview) may
	// supply them themselves; if they do not, safe fallbacks are used.
	if full["RecipientEmail"] == nil {
		full["RecipientEmail"] = ""
	}
	if full["SupportEmail"] == nil {
		full["SupportEmail"] = ""
	}
	// Split the Code into individual runes for the digit-cell table. If no Code
	// is present (non-verification templates) these remain nil and the template
	// range produces nothing.
	if code, ok := full["Code"]; ok && full["Digits"] == nil {
		codeStr := fmt.Sprintf("%v", code)
		// CodeRaw is the plain string for the copy bar.
		if full["CodeRaw"] == nil {
			full["CodeRaw"] = codeStr
		}
		digits := []string{}
		inks := []template.CSS{}
		ink := template.CSS(full["Ink"].(string))
		for _, ch := range codeStr {
			digits = append(digits, string(ch))
			inks = append(inks, ink)
		}
		full["Digits"] = digits
		full["DigitInk"] = inks
	}

	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template %q: %w", name, err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, full); err != nil {
		return "", fmt.Errorf("executing template %q: %w", name, err)
	}

	// The fragment is our own markup, already escaped by its own pass, so it
	// goes into the shell as HTML rather than being escaped a second time into
	// visible angle brackets. Escaping twice is what turns a paragraph into
	// visible &lt;p&gt; in somebody's inbox.
	//
	// Safe because body is the output of html/template, not an input to it:
	// whatever the caller put in the data was escaped when the fragment was
	// executed, one statement above.
	//nolint:gosec // G203: already escaped by the fragment's own render
	//#nosec G203
	full["Content"] = template.HTML(body.String())

	shell, err := template.New("layout").Parse(Layout)
	if err != nil {
		return "", fmt.Errorf("parsing the mail layout: %w", err)
	}
	var out bytes.Buffer
	if err := shell.Execute(&out, full); err != nil {
		return "", fmt.Errorf("executing the mail layout: %w", err)
	}
	return out.String(), nil
}
