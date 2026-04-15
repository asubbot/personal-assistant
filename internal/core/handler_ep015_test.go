package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"strings"
	"testing"
)

// Covers AC-15.001: multi-completion turn aggregates usage into the trailing footer (also covers REQ-15.006 layout).
// Also asserts one INFO "main llm completion" per successful main-model Complete (usage observability).
func TestHandleMessage_EP015_tokenFooter_sumsAcrossToolRound(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "hello from node"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		return &llm.CompletionResult{
			Content: "Done.",
			Usage:   llm.Usage{PromptTokens: 20, CompletionTokens: 7, TotalTokens: 27},
		}, nil
	}
	cap := &captureHandler{level: slog.LevelInfo}
	logger := slog.New(cap)
	h := &conversationHandler{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     logger,
	}

	reply, err := h.HandleMessage(context.Background(), 1, "", "run echo hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	wantSuffix := "\n*Tokens 42 (in: 30 / out: 12) · full*"
	if !strings.HasSuffix(reply, wantSuffix) {
		t.Fatalf("reply = %q, want suffix %q", reply, wantSuffix)
	}
	var mainLLMLogs int
	for _, r := range cap.records {
		if r.msg == "main llm completion" && r.level == slog.LevelInfo {
			mainLLMLogs++
		}
	}
	if mainLLMLogs != 2 {
		t.Fatalf("expected 2 Info \"main llm completion\" records, got %d: %+v", mainLLMLogs, cap.records)
	}
}

// Covers AC-15.002: all zero usage → no token footer in reply.
func TestHandleMessage_EP015_noFooterWhenZeroUsage(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{
		Content: "hello",
		Usage:   llm.Usage{},
	}}
	h := &conversationHandler{
		router:  mustRouterSingle(t, provider),
		catalog: nil,
		logger:  slog.Default(),
	}
	reply, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(reply, "\n*Tokens ") || strings.Contains(reply, "\nTokens ") {
		t.Fatalf("unexpected token footer in %q", reply)
	}
}

// Covers AC-15.005: session memory stores assistant body without the token footer.
func TestHandleMessage_EP015_sessionMemoryExcludesFooter(t *testing.T) {
	store := newSessionWindowStore()
	provider := &mockProvider{result: &llm.CompletionResult{
		Content: "hello",
		Usage:   llm.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
	}}
	h := &conversationHandler{
		router:       mustRouterSingle(t, provider),
		catalog:      nil,
		logger:       slog.Default(),
		sessionCfg:   &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore: store,
	}
	const sk = "chat-ep015"
	reply, err := h.HandleMessage(context.Background(), 1, sk, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply, "Tokens ") {
		t.Fatalf("expected footer in reply, got %q", reply)
	}
	snap := store.snapshot(sk)
	if len(snap) != 1 {
		t.Fatalf("snapshot len = %d, want 1", len(snap))
	}
	if strings.Contains(snap[0].assistant, "Tokens ") {
		t.Fatalf("session assistant must not contain footer, got %q", snap[0].assistant)
	}
	if snap[0].assistant != "hello" {
		t.Fatalf("session assistant = %q, want hello", snap[0].assistant)
	}
}
