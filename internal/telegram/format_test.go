package telegram

import (
	"strings"
	"testing"
)

// Covers AC-12.001 (EP-012): **bold** and `code` become Telegram HTML tags.
func TestMarkdownToTelegramHTML_boldAndInlineCode(t *testing.T) {
	in := "Hello **world** and `code` here"
	out := MarkdownToTelegramHTML(in)
	if !strings.Contains(out, "<b>") || !strings.Contains(out, "world") || !strings.Contains(out, "</b>") {
		t.Fatalf("expected <b>world</b> in %q", out)
	}
	if !strings.Contains(out, "<code>") || !strings.Contains(out, "code") || !strings.Contains(out, "</code>") {
		t.Fatalf("expected <code> in %q", out)
	}
}

// Covers AC-12.001 (EP-012): link to https becomes anchor.
func TestMarkdownToTelegramHTML_link(t *testing.T) {
	in := "See [here](https://example.com/path) now"
	out := MarkdownToTelegramHTML(in)
	if !strings.Contains(out, `<a href="https://example.com/path">`) || !strings.Contains(out, "here") {
		t.Fatalf("expected anchor in %q", out)
	}
}

// Covers AC-12.002 (EP-012): literal script text is escaped.
func TestMarkdownToTelegramHTML_escapesRawAngleBrackets(t *testing.T) {
	in := "Use <script>alert(1)</script> carefully"
	out := MarkdownToTelegramHTML(in)
	if strings.Contains(out, "<script>") {
		t.Fatalf("unescaped script tag in %q", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected escaped script in %q", out)
	}
}

// Covers AC-12.001 (EP-012): fenced block becomes pre.
func TestMarkdownToTelegramHTML_fencedCode(t *testing.T) {
	in := "Before:\n```\na < b\n```\nAfter"
	out := MarkdownToTelegramHTML(in)
	if !strings.Contains(out, "<pre>") || !strings.Contains(out, "a &lt; b") {
		t.Fatalf("expected pre with escaped < in %q", out)
	}
}

// Covers AC-12.001 (EP-012): markdown table lines wrapped in pre.
func TestMarkdownToTelegramHTML_tableAsPre(t *testing.T) {
	in := "| a | b |\n|---|---|\n| 1 | 2 |"
	out := MarkdownToTelegramHTML(in)
	if !strings.Contains(out, "<pre>") || !strings.Contains(out, "| a | b |") {
		t.Fatalf("expected table in pre %q", out)
	}
}
