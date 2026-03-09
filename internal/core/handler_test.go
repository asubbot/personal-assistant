package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/llm"
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
