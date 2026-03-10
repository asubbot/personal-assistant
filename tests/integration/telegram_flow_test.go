//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/llm"
	"testing"
	"time"
)

// mockLLM implements llm.Provider for integration tests.
type mockLLM struct {
	content string
}

func (m *mockLLM) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	return &llm.CompletionResult{Content: m.content}, nil
}

// fakeAdapter simulates one incoming message: calls the handler once and sends the result to a channel.
type fakeAdapter struct {
	userID int64
	text   string
	done   chan result
}

type result struct {
	reply string
	err   error
}

func (a *fakeAdapter) Run(ctx context.Context, handler core.MessageHandler) error {
	reply, err := handler.HandleMessage(ctx, a.userID, a.text)
	a.done <- result{reply: reply, err: err}
	return nil
}

const integrationTimeout = 5 * time.Second

// TestTelegramFlow_OneMessage_ReplyWithinTimeout exercises the path: adapter → core handler → LLM → reply.
// Mocks: fake adapter (no real Telegram), mock LLM (no real API). Asserts a reply is returned before test timeout (AC-001).
func TestTelegramFlow_OneMessage_ReplyWithinTimeout(t *testing.T) {
	wantReply := "hello from mock"
	adapter := &fakeAdapter{userID: 1, text: "hi", done: make(chan result, 1)}
	provider := &mockLLM{content: wantReply}
	cfg := &config.Config{}
	logger := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, provider)
		close(done)
	}()

	select {
	case res := <-adapter.done:
		if res.err != nil {
			t.Fatalf("handler error: %v", res.err)
		}
		if res.reply != wantReply {
			t.Errorf("reply = %q, want %q", res.reply, wantReply)
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("no reply within %v (test timeout)", integrationTimeout)
	}

	cancel()
	<-done
}
