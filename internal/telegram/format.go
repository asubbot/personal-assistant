package telegram

import (
	"html"
	"regexp"
	"strings"
)

var reTelegramLink = regexp.MustCompile(`\[([^\]]*)\]\((https?://[^)\s]+|tg://[^)\s]+)\)`)

var (
	reHeader3 = regexp.MustCompile(`^###\s+(.*)$`)
	reHeader2 = regexp.MustCompile(`^##\s+(.*)$`)
)

// MarkdownToTelegramHTML converts common assistant Markdown-style text to Telegram Bot API HTML.
// It escapes raw HTML and maps a subset of Markdown to supported tags.
func MarkdownToTelegramHTML(s string) string {
	if s == "" {
		return ""
	}
	var out strings.Builder
	rest := s
	for {
		idx := strings.Index(rest, "```")
		if idx < 0 {
			out.WriteString(convertTextChunk(rest))
			break
		}
		out.WriteString(convertTextChunk(rest[:idx]))
		rest = rest[idx+3:]
		rest = strings.TrimPrefix(rest, "\r\n")
		rest = strings.TrimPrefix(rest, "\n")
		end := strings.Index(rest, "```")
		if end < 0 {
			out.WriteString(html.EscapeString("```"))
			out.WriteString(convertTextChunk(rest))
			break
		}
		code := rest[:end]
		out.WriteString("<pre>")
		out.WriteString(html.EscapeString(strings.TrimSuffix(strings.TrimPrefix(code, "\n"), "\r")))
		out.WriteString("</pre>")
		rest = rest[end+3:]
	}
	return out.String()
}

func convertTextChunk(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	var out strings.Builder
	var tableBuf []string
	flushTable := func() {
		if len(tableBuf) == 0 {
			return
		}
		out.WriteString("<pre>")
		out.WriteString(html.EscapeString(strings.Join(tableBuf, "\n")))
		out.WriteString("</pre>\n")
		tableBuf = nil
	}
	for _, line := range lines {
		if isTableLine(line) {
			tableBuf = append(tableBuf, line)
			continue
		}
		flushTable()
		out.WriteString(convertLine(line))
		out.WriteString("\n")
	}
	flushTable()
	res := out.String()
	return strings.TrimSuffix(res, "\n")
}

func isTableLine(line string) bool {
	return strings.Count(line, "|") >= 2
}

func convertLine(line string) string {
	t := strings.TrimRight(line, "\r")
	if m := reHeader3.FindStringSubmatch(t); m != nil {
		return "<b>" + html.EscapeString(m[1]) + "</b>"
	}
	if m := reHeader2.FindStringSubmatch(t); m != nil {
		return "<b>" + html.EscapeString(m[1]) + "</b>"
	}
	return convertLineWithBackticks(t)
}

func convertLineWithBackticks(line string) string {
	parts := strings.Split(line, "`")
	var b strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			b.WriteString("<code>")
			b.WriteString(html.EscapeString(p))
			b.WriteString("</code>")
			continue
		}
		b.WriteString(convertRun(p))
	}
	return b.String()
}

// splitBold splits s into alternating outside / inside ** ... ** segments.
func splitBold(s string) []string {
	var out []string
	rest := s
	for {
		i := strings.Index(rest, "**")
		if i < 0 {
			out = append(out, rest)
			return out
		}
		out = append(out, rest[:i])
		rest = rest[i+2:]
		j := strings.Index(rest, "**")
		if j < 0 {
			out = append(out, rest)
			return out
		}
		out = append(out, rest[:j])
		rest = rest[j+2:]
	}
}

// convertRun applies **bold**, links, and *italic* (outside code and pre).
func convertRun(s string) string {
	parts := splitBold(s)
	var b strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			b.WriteString("<b>")
			b.WriteString(convertRunPlain(p))
			b.WriteString("</b>")
		} else {
			b.WriteString(convertRunPlain(p))
		}
	}
	return b.String()
}

func convertRunPlain(s string) string {
	var b strings.Builder
	rest := s
	for {
		loc := reTelegramLink.FindStringIndex(rest)
		if loc == nil {
			b.WriteString(convertItalic(html.EscapeString(rest)))
			break
		}
		if loc[0] > 0 {
			b.WriteString(convertItalic(html.EscapeString(rest[:loc[0]])))
		}
		sub := reTelegramLink.FindStringSubmatch(rest[loc[0]:loc[1]])
		if len(sub) != 3 {
			break
		}
		href := html.EscapeString(sub[2])
		label := html.EscapeString(sub[1])
		b.WriteString(`<a href="`)
		b.WriteString(href)
		b.WriteString(`">`)
		b.WriteString(label)
		b.WriteString(`</a>`)
		rest = rest[loc[1]:]
	}
	return b.String()
}

// convertItalic wraps first-level *...* pairs in <i> after the rest is HTML-escaped.
func convertItalic(escaped string) string {
	var b strings.Builder
	rest := escaped
	for {
		i := strings.Index(rest, "*")
		if i < 0 {
			b.WriteString(rest)
			return b.String()
		}
		b.WriteString(rest[:i])
		rest = rest[i+1:]
		j := strings.Index(rest, "*")
		if j < 0 {
			b.WriteString("*")
			b.WriteString(rest)
			return b.String()
		}
		inner := rest[:j]
		rest = rest[j+1:]
		b.WriteString("<i>")
		b.WriteString(inner)
		b.WriteString("</i>")
	}
}
