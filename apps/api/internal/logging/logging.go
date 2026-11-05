// Package logging gives every line the API prints one shape.
//
// Before this, a boot wrote straight through the standard logger: a wall of
// "[GIN-debug] GET /api/v1/..." route lines, then a mix of messages stamped
// "2026/11/04 17:08:16" — .Setup replaces the standard logger's destination
// with one that formats every line the same way, so nothing had to be changed
// at the four hundred call sites that already log:
//
//	17:08:16  Server listening on http://localhost:8080
//	17:08:16  [realtime] backplane connected on "grit:realtime"
//	17:09:57  401  GET   /api/v1/auth/me              1.6ms  ::1
//
// A dim timestamp, then the message. A leading [module] tag is tinted so a
// subsystem is scannable, and "Warning:" / "Error:" lines are tinted by the
// words the code already uses. Colour is used only when the destination is a
// terminal and NO_COLOR is unset, so a log piped to a file, air, Docker or a
// collector stays plain and greppable.
package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// ANSI SGR sequences. Every tinted span is closed with reset so one truncated
// line cannot bleed colour into the next.
const (
	reset  = "\x1b[0m"
	dim    = "\x1b[2m"
	bold   = "\x1b[1m"
	red    = "\x1b[31m"
	green  = "\x1b[32m"
	yellow = "\x1b[33m"
	cyan   = "\x1b[36m"
)

// colour is decided once in Setup and read by every write.
var colour bool

// Setup installs the process-wide log format. Call it before anything else
// logs, so no early line escapes in the old shape.
func Setup() {
	colour = wantColour()
	log.SetFlags(0)
	log.SetOutput(writer{out: os.Stderr})
}

// wantColour decides between tinted and plain output.
//
// A terminal gets colour; a pipe does not, so air, Docker, CI and
// "go run ... > file" all get clean text free of escape codes. NO_COLOR turns
// it off, LOG_COLOR=always turns it on for a colourising pager, and anything
// else falls back to asking whether stderr is a character device.
func wantColour() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	switch strings.ToLower(os.Getenv("LOG_COLOR")) {
	case "always", "1", "true", "yes":
		return true
	case "never", "0", "false", "no":
		return false
	}
	info, err := os.Stderr.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// writer renders one log entry.
type writer struct{ out *os.File }

func (w writer) Write(p []byte) (int, error) {
	// The log package makes one Write call per entry and appends a newline. A
	// message carrying its own newlines is stamped on the first line only and
	// left indented after that, so a boot banner reads as one block rather
	// than a column of identical timestamps.
	body := strings.TrimRight(string(p), "\n")
	stamp := paint(dim, time.Now().Format("15:04:05"))
	for i, line := range strings.Split(body, "\n") {
		if i == 0 {
			fmt.Fprintf(w.out, "%s  %s\n", stamp, highlight(line))
			continue
		}
		fmt.Fprintf(w.out, "%s%s\n", contIndent, line)
	}
	return len(p), nil
}

// contIndent lines a continuation up under the first line's message, past the
// "HH:MM:SS" and its two trailing spaces.
const contIndent = "          "

// highlight tints the parts of a message that carry meaning and leaves the
// rest plain, so a message still reads as a sentence.
//
//	[realtime] ...   a leading [module] tag, so subsystems are scannable
//	Warning: ...     the one severity word this codebase already uses
//	Error: ...       a failure
func highlight(s string) string {
	if !colour {
		return s
	}
	switch {
	case strings.HasPrefix(s, "Warning:"):
		return paint(yellow, s)
	case strings.HasPrefix(s, "Error:"):
		return paint(red, s)
	}
	if tag, rest, ok := splitTag(s); ok {
		return paint(cyan, tag) + rest
	}
	return s
}

// splitTag pulls a leading "[realtime]" off a message. The bounds keep a
// message that merely starts with a bracket from being read as a module tag.
func splitTag(s string) (tag, rest string, ok bool) {
	if !strings.HasPrefix(s, "[") {
		return "", s, false
	}
	end := strings.IndexByte(s, ']')
	if end < 2 || end > 24 {
		return "", s, false
	}
	return s[:end+1], s[end+1:], true
}

// paint wraps s in a colour when colour is on, and returns it untouched when
// it is off so piped output has no escape codes at all.
func paint(code, s string) string {
	if !colour {
		return s
	}
	return code + s + reset
}

// Info prints a normal line.
func Info(format string, a ...any) { log.Printf(format, a...) }

// Warn prints a non-fatal problem under the "Warning:" prefix the codebase
// already uses, so it is tinted yellow wherever it appears.
func Warn(format string, a ...any) { log.Printf("Warning: "+format, a...) }

// Error prints a failure, tinted red.
func Error(format string, a ...any) { log.Printf("Error: "+format, a...) }

// Request prints one access line:
//
//	17:09:57  401  GET   /api/v1/auth/me              1.6ms  ::1
//
// The status is coloured by class, method and path are padded so a column of
// requests lines up, and the request id is appended only when the response is
// an error — that is the case a support report needs to find, and it is a long
// value that would otherwise sit on every success line.
func Request(status int, method, path string, d time.Duration, ip, requestID string) {
	line := fmt.Sprintf("%s  %-6s %-46s %8s  %s",
		paint(statusColour(status), fmt.Sprintf("%d", status)),
		method,
		clip(path, 46),
		shortDuration(d),
		ip,
	)
	if status >= 400 && requestID != "" {
		line += "  " + paint(dim, "id="+requestID)
	}
	log.Print(line)
}

// Banner prints the boot header: a title and aligned key/value rows, as a
// single entry so the timestamps do not repeat down the block.
//
//	logging.Banner("CampVote API", [][2]string{
//		{"env", cfg.AppEnv},
//		{"listening", "http://localhost:" + cfg.Port},
//	})
func Banner(title string, rows [][2]string) {
	if len(rows) == 0 {
		log.Print(paint(bold, title))
		return
	}
	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	var b strings.Builder
	b.WriteString(paint(bold, title))
	for _, row := range rows {
		fmt.Fprintf(&b, "\n  %s  %s", paint(cyan, fmt.Sprintf("%-*s", width, row[0])), row[1])
	}
	log.Print(b.String())
}

// statusColour picks the tint for a response class: green for served, cyan for
// redirected, yellow for a client mistake, red for a server one.
func statusColour(status int) string {
	switch {
	case status >= 500:
		return red
	case status >= 400:
		return yellow
	case status >= 300:
		return cyan
	default:
		return green
	}
}

// shortDuration rounds to one decimal and picks the unit, so the column stays
// narrow: 0.9ms, 12.4ms, 1.2s.
func shortDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.0fus", float64(d)/float64(time.Microsecond))
	default:
		return "0ms"
	}
}

// clip trims a long path to width, keeping the head and marking the cut, so a
// runaway query string cannot push the rest of the columns off the screen.
func clip(s string, width int) string {
	if len(s) <= width {
		return s
	}
	return s[:width-1] + "…"
}
