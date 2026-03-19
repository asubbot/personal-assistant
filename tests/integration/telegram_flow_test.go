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

// mockLLMWithCalled records whether Complete was invoked (for AC-01.002 integration: assert no LLM call on rejection).
type mockLLMWithCalled struct {
	content string
	Called  bool
}

func (m *mockLLMWithCalled) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.Called = true
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
// Mocks: fake adapter (no real Telegram), mock LLM (no real API). Asserts a reply is returned before test timeout (AC-01.001).
func TestTelegramFlow_OneMessage_ReplyWithinTimeout(t *testing.T) {
	t.Parallel()
	wantReply := "hello from mock"
	adapter := &fakeAdapter{userID: 1, text: "hi", done: make(chan result, 1)}
	provider := &mockLLM{content: wantReply}
	cfg := &config.Config{}
	logger := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil)
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

// TestTelegramFlow_EmptyMessage_RejectionNoLLMCall covers AC-01.002 (integration): empty message → rejection, no LLM call.
func TestTelegramFlow_EmptyMessage_RejectionNoLLMCall(t *testing.T) {
	t.Parallel()
	provider := &mockLLMWithCalled{content: "should not be used"}
	adapter := &fakeAdapter{userID: 1, text: "   ", done: make(chan result, 1)}
	cfg := &config.Config{}
	logger := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil)
		close(done)
	}()

	select {
	case res := <-adapter.done:
		if res.err != nil {
			t.Fatalf("handler error: %v", res.err)
		}
		if res.reply != "Please send a non-empty message." {
			t.Errorf("reply = %q, want rejection message", res.reply)
		}
		if provider.Called {
			t.Error("LLM Complete must not be called for empty/whitespace message")
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("no reply within %v", integrationTimeout)
	}

	cancel()
	<-done
}

// TestTelegramFlow_OverMaxLength_RejectionNoLLMCall covers AC-01.002 (integration): message over max length → rejection, no LLM call.
func TestTelegramFlow_OverMaxLength_RejectionNoLLMCall(t *testing.T) {
	t.Parallel()
	provider := &mockLLMWithCalled{content: "should not be used"}
	adapter := &fakeAdapter{userID: 1, text: "1234567", done: make(chan result, 1)}
	cfg := &config.Config{}
	cfg.Telegram.MaxMessageLength = 5
	logger := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil)
		close(done)
	}()

	select {
	case res := <-adapter.done:
		if res.err != nil {
			t.Fatalf("handler error: %v", res.err)
		}
		if res.reply != "Message is too long. Maximum length is 5 characters." {
			t.Errorf("reply = %q, want max length rejection", res.reply)
		}
		if provider.Called {
			t.Error("LLM Complete must not be called when message exceeds max length")
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("no reply within %v", integrationTimeout)
	}

	cancel()
	<-done
}

// TestTelegramFlow_DifferentProviderUsedPerRun covers AC-01.016 (integration): run with provider A then with provider B → each run uses the provider passed to Run.
func TestTelegramFlow_DifferentProviderUsedPerRun(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	cfg := &config.Config{}

	// First run: provider A
	replyA := "reply from provider A"
	adapterA := &fakeAdapter{userID: 1, text: "hi", done: make(chan result, 1)}
	providerA := &mockLLM{content: replyA}

	ctx1, cancel1 := context.WithCancel(context.Background())
	done1 := make(chan struct{})
	go func() {
		_ = core.Run(ctx1, cfg, logger, adapterA, []llm.Provider{providerA}, []string{"test/default"}, nil, nil, nil, nil, nil)
		close(done1)
	}()

	select {
	case res := <-adapterA.done:
		if res.err != nil {
			t.Fatalf("first run handler error: %v", res.err)
		}
		if res.reply != replyA {
			t.Errorf("first run reply = %q, want %q", res.reply, replyA)
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("first run: no reply within %v", integrationTimeout)
	}
	cancel1()
	<-done1

	// Second run: provider B (new provider used)
	replyB := "reply from provider B"
	adapterB := &fakeAdapter{userID: 1, text: "hi", done: make(chan result, 1)}
	providerB := &mockLLM{content: replyB}

	ctx2, cancel2 := context.WithCancel(context.Background())
	done2 := make(chan struct{})
	go func() {
		_ = core.Run(ctx2, cfg, logger, adapterB, []llm.Provider{providerB}, []string{"test/default"}, nil, nil, nil, nil, nil)
		close(done2)
	}()

	select {
	case res := <-adapterB.done:
		if res.err != nil {
			t.Fatalf("second run handler error: %v", res.err)
		}
		if res.reply != replyB {
			t.Errorf("second run reply = %q, want %q", res.reply, replyB)
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("second run: no reply within %v", integrationTimeout)
	}
	cancel2()
	<-done2
}
