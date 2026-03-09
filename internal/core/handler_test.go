package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/llm"
	"strings"
	"testing"
)

type mockProvider struct {
	result *llm.CompletionResult
	err    error
	// lastCall records the last Complete call (messages and opts) for assertion
	lastMessages []llm.Message
	lastOpts     *llm.CompletionOptions
}

func (m *mockProvider) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.lastMessages = messages
	m.lastOpts = opts
	return m.result, m.err
}

func TestHandleMessage_returnsProviderContent(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello back"}}
	h := &conversationHandler{provider: provider, logger: logger}

	reply, err := h.HandleMessage(context.Background(), 99, "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "hello back" {
		t.Errorf("reply = %q, want %q", reply, "hello back")
	}
}

func TestHandleMessage_returnsProviderError(t *testing.T) {
	wantErr := errors.New("provider failed")
	logger := slog.Default()
	provider := &mockProvider{err: wantErr}
	h := &conversationHandler{provider: provider, logger: logger}

	reply, err := h.HandleMessage(context.Background(), 1, "hi")
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if reply != "" {
		t.Errorf("reply = %q, want empty", reply)
	}
}

func TestHandleMessage_passesSystemAndUserMessages(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{provider: provider, logger: logger}

	userText := "what is 2+2?"
	_, _ = h.HandleMessage(context.Background(), 42, userText)

	if len(provider.lastMessages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(provider.lastMessages))
	}
	if provider.lastMessages[0].Role != "system" || provider.lastMessages[0].Content != "You are a helpful assistant. Reply concisely." {
		t.Errorf("messages[0] = %+v, want system + assistant prompt", provider.lastMessages[0])
	}
	if provider.lastMessages[1].Role != "user" || provider.lastMessages[1].Content != userText {
		t.Errorf("messages[1] = %+v, want user + %q", provider.lastMessages[1], userText)
	}
	if provider.lastOpts != nil {
		t.Errorf("opts = %v, want nil", provider.lastOpts)
	}
}

func TestHandleMessage_emptyReturnsRejectionMessage(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "x"}}
	h := &conversationHandler{provider: provider, logger: logger}

	for _, text := range []string{"", "  ", "\t\n"} {
		reply, err := h.HandleMessage(context.Background(), 1, text)
		if err != nil {
			t.Errorf("text %q: err = %v", text, err)
		}
		if reply != "Please send a non-empty message." {
			t.Errorf("text %q: reply = %q, want rejection message", text, reply)
		}
	}
	if len(provider.lastMessages) != 0 {
		t.Error("provider.Complete should not be called for empty text")
	}
}

func TestHandleMessage_rejectsWhenOverMaxLength(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 5}

	// at limit: 5 runes — goes through
	reply, err := h.HandleMessage(context.Background(), 1, "12345")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q", reply)
	}
	if provider.lastMessages[1].Content != "12345" {
		t.Errorf("content = %q, want 12345", provider.lastMessages[1].Content)
	}

	// over limit: 7 runes — rejected, no LLM call
	provider.lastMessages = nil
	reply, err = h.HandleMessage(context.Background(), 1, "1234567")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "Message is too long. Maximum length is 5 characters." {
		t.Errorf("reply = %q", reply)
	}
	if len(provider.lastMessages) != 0 {
		t.Error("provider should not be called for over-length message")
	}
}

func TestHandleMessage_noLimit_longMessageGoesToProvider(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 0}

	longText := strings.Repeat("a", 10000)
	reply, err := h.HandleMessage(context.Background(), 1, longText)
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q", reply)
	}
	if len(provider.lastMessages) != 2 || provider.lastMessages[1].Content != longText {
		t.Errorf("provider should receive full long message; got content len %d", len(provider.lastMessages[1].Content))
	}
}

func TestHandleMessage_maxLength_unicodeRunes(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	// "привет" = 6 runes
	cyrillic6 := "привет"

	h := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 6}
	reply, err := h.HandleMessage(context.Background(), 1, cyrillic6)
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want ok (at limit)", reply)
	}
	if provider.lastMessages[1].Content != cyrillic6 {
		t.Errorf("content = %q, want %q", provider.lastMessages[1].Content, cyrillic6)
	}

	// limit 5: 6 runes → rejected
	provider.lastMessages = nil
	h5 := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 5}
	reply, err = h5.HandleMessage(context.Background(), 1, cyrillic6)
	if err != nil {
		t.Fatal(err)
	}
	if reply != "Message is too long. Maximum length is 5 characters." {
		t.Errorf("reply = %q", reply)
	}
	if len(provider.lastMessages) != 0 {
		t.Error("provider should not be called when over limit (runes)")
	}
}
