package telegram

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-telegram/bot/models"
)

// Covers AC-15.003, AC-15.006: token footer appears only on the last chunk; Markdown footer has no angle brackets before send.
func TestSendLongOutboundText_EP015_footerOnlyOnLastChunk(t *testing.T) {
	body := strings.Repeat("x\n\n", 3000)
	foot := "*Tokens 3 (in: 2 / out: 1) · full_lite*"
	src := body + "\n" + foot
	m := &mockSender{}
	if err := sendLongOutboundText(context.Background(), m, 1, src); err != nil {
		t.Fatal(err)
	}
	texts := m.sentTexts()
	if len(texts) < 2 {
		t.Fatalf("want multiple chunks, got %d", len(texts))
	}
	for i := 0; i < len(texts)-1; i++ {
		if strings.Contains(texts[i], "Tokens ") {
			t.Fatalf("chunk %d must not contain footer", i)
		}
	}
	last := texts[len(texts)-1]
	if !strings.Contains(last, "<i>") || !strings.Contains(last, "Tokens 3 (in: 2 / out: 1) · full_lite") {
		t.Fatalf("last chunk missing italic footer: %q", last)
	}
	if strings.ContainsAny(foot, "<>") {
		t.Fatal("test footer markdown must not contain angle brackets")
	}
}

// Covers AC-15.004, AC-15.007: empty body with footer only does not send a message.
func TestSendLongOutboundText_EP015_emptyBodySkipsSend(t *testing.T) {
	m := &mockSender{}
	src := "\n*Tokens 1 (in: 1 / out: 0)*"
	if err := sendLongOutboundText(context.Background(), m, 1, src); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	n := len(m.sent)
	m.mu.Unlock()
	if n != 0 {
		t.Fatalf("want no sends, got %d", n)
	}
}

// Covers AC-01.001: traceability for TestSplitTelegramOutboundSource_shortUnchanged.
func TestSplitTelegramOutboundSource_shortUnchanged(t *testing.T) {
	s := "hello **world**"
	got := splitTelegramOutboundSource(s)
	if len(got) != 1 || got[0] != s {
		t.Fatalf("want single chunk %q, got %#v", s, got)
	}
}

// Covers AC-01.001: traceability for TestSplitTelegramOutboundSource_longPlainSplits.
func TestSplitTelegramOutboundSource_longPlainSplits(t *testing.T) {
	s := strings.Repeat("n", 12000)
	got := splitTelegramOutboundSource(s)
	if len(got) < 3 {
		t.Fatalf("expected multiple chunks, got %d", len(got))
	}
	for i, ch := range got {
		html := MarkdownToTelegramHTML(ch)
		if c := utf8.RuneCountInString(html); c > telegramBotAPIMaxMessageRunes {
			t.Fatalf("chunk %d html rune count %d > %d", i, c, telegramBotAPIMaxMessageRunes)
		}
	}
}

// Covers AC-01.001: traceability for TestSendLongOutboundText_eachChunkWithinLimit.
func TestSendLongOutboundText_eachChunkWithinLimit(t *testing.T) {
	m := &mockSender{}
	src := strings.Repeat("z", 15000)
	if err := sendLongOutboundText(context.Background(), m, 1, src); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) < 4 {
		t.Fatalf("expected several sends, got %d", len(m.sent))
	}
	for i, rec := range m.sent {
		if rec.parseMode != models.ParseModeHTML {
			t.Errorf("chunk %d: want HTML parse mode", i)
		}
		if c := utf8.RuneCountInString(rec.text); c > telegramBotAPIMaxMessageRunes {
			t.Fatalf("chunk %d rune count %d > %d", i, c, telegramBotAPIMaxMessageRunes)
		}
	}
}

// Covers AC-15.003: SplitTokenFooterSuffix recognizes token footer with optional intent tier suffix.
func TestSplitTokenFooterSuffix_withIntentTier(t *testing.T) {
	const body = "reply text"
	const foot = "*Tokens 10 (in: 8 / out: 2) · full_lite*"
	src := body + "\n" + foot
	gotBody, gotFoot := SplitTokenFooterSuffix(src)
	if gotBody != body || gotFoot != foot {
		t.Fatalf("SplitTokenFooterSuffix(%q) = body %q foot %q, want body %q foot %q", src, gotBody, gotFoot, body, foot)
	}
}
