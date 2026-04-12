package telegram

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-telegram/bot/models"
)

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
