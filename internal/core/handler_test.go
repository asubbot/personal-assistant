package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/prompt"
	"strings"
	"testing"
	"time"
)

// Covers AC-38.002, AC-38.014
// Supporting AC-01.001, REQ-01.001: handler returns provider content to caller.
func TestHandleMessage_returnsProviderContent(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello back"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	reply, err := h.HandleMessage(context.Background(), 99, "", "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "hello back" {
		t.Errorf("reply = %q, want %q", reply, "hello back")
	}
}

// Supporting AC-01.001, REQ-01.001: handler propagates provider error to caller.
func TestHandleMessage_returnsProviderError(t *testing.T) {
	wantErr := errors.New("provider failed")
	logger := slog.Default()
	provider := &mockProvider{err: wantErr}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if reply != "" {
		t.Errorf("reply = %q, want empty", reply)
	}
}

// Supporting AC-01.001, REQ-01.001: handler passes system and user messages to LLM provider.
// Covers AC-35.017: system message still begins with prompt.TrustPolicy after the prompt-package merge.
func TestHandleMessage_passesSystemAndUserMessages(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	userText := "what is 2+2?"
	_, _ = h.HandleMessage(context.Background(), 42, "", userText)

	if len(provider.lastMessages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(provider.lastMessages))
	}
	sys := provider.lastMessages[0].Content
	if provider.lastMessages[0].Role != "system" || !strings.HasPrefix(sys, prompt.TrustPolicy) {
		t.Errorf("messages[0] = %+v, want system starting with trust policy", provider.lastMessages[0])
	}
	if !strings.Contains(sys, "Calendar date: ") {
		t.Errorf("system message missing calendar date line: %s", sys)
	}
	wantDate := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(sys, wantDate) {
		t.Errorf("system message should contain today's UTC date %q", wantDate)
	}
	if !strings.Contains(sys, "You are a helpful assistant. Reply concisely.") {
		t.Errorf("system message missing personality line: %s", sys)
	}
	if provider.lastMessages[1].Role != "user" || provider.lastMessages[1].Content != userText {
		t.Errorf("messages[1] = %+v, want user + %q", provider.lastMessages[1], userText)
	}
	if provider.lastOpts != nil {
		t.Errorf("opts = %v, want nil", provider.lastOpts)
	}
}

// Covers AC-01.002, REQ-01.001: empty or whitespace message rejected with clear message, no LLM call.
func TestHandleMessage_emptyReturnsRejectionMessage(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "x"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	for _, text := range []string{"", "  ", "\t\n"} {
		reply, err := h.HandleMessage(context.Background(), 1, "", text)
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

// Covers AC-01.002, REQ-01.001: message over max length rejected, no LLM call.
func TestHandleMessage_rejectsWhenOverMaxLength(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 5}.handler()

	// at limit: 5 runes — goes through
	reply, err := h.HandleMessage(context.Background(), 1, "", "12345")
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
	reply, err = h.HandleMessage(context.Background(), 1, "", "1234567")
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

// Supporting AC-01.002, REQ-01.001: when max length is 0, long message is not truncated and goes to provider.
func TestHandleMessage_noLimit_longMessageGoesToProvider(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 0}.handler()

	longText := strings.Repeat("a", 10000)
	reply, err := h.HandleMessage(context.Background(), 1, "", longText)
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

// Covers AC-01.031, REQ-01.021: at INFO level only metadata is logged.
func TestHandleMessage_logsMetadataAtInfo(t *testing.T) {
	cap := &captureHandler{level: slog.LevelInfo}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	_, _ = h.HandleMessage(context.Background(), 1, "", "hi")

	var hasMainLLM bool
	for _, r := range cap.records {
		if r.msg == "main llm completion" && r.level == slog.LevelInfo {
			hasMainLLM = true
			break
		}
	}
	if !hasMainLLM {
		t.Errorf("expected one Info \"main llm completion\" record, got records: %+v", cap.records)
	}
	// No Debug records (request/response) at INFO level
	for _, r := range cap.records {
		if r.level == slog.LevelDebug {
			t.Errorf("at INFO level expected no Debug records, got msg=%q", r.msg)
		}
	}
}

// Covers AC-01.031, REQ-01.021: at DEBUG level full request and response are logged.
func TestHandleMessage_logsFullRequestResponseAtDebug(t *testing.T) {
	cap := &captureHandler{level: slog.LevelDebug}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello", Usage: llm.Usage{}}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	_, _ = h.HandleMessage(context.Background(), 1, "", "hi")

	var hasRequest, hasCall, hasResponse bool
	for _, r := range cap.records {
		switch r.msg {
		case "llm request":
			hasRequest = true
		case "main llm completion":
			hasCall = true
		case "llm response":
			hasResponse = true
		}
	}
	if !hasRequest {
		t.Errorf("at DEBUG expected \"llm request\" record, got %+v", cap.records)
	}
	if !hasCall {
		t.Errorf("at DEBUG expected \"llm call\" record, got %+v", cap.records)
	}
	if !hasResponse {
		t.Errorf("at DEBUG expected \"llm response\" record, got %+v", cap.records)
	}
}

// Covers AC-01.002, REQ-01.001: max length enforced by runes.
func TestHandleMessage_maxLength_unicodeRunes(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	// "привет" = 6 runes
	cyrillic6 := "привет"

	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 6}.handler()
	reply, err := h.HandleMessage(context.Background(), 1, "", cyrillic6)
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
	h5 := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 5}.handler()
	reply, err = h5.HandleMessage(context.Background(), 1, "", cyrillic6)
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
