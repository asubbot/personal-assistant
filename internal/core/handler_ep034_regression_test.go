package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/llmrouter"
	"pa/internal/toolcatalog"
	"strings"
	"testing"
)

func mustRouterMulti(t *testing.T, providers []llm.Provider, labels []string) *llmrouter.Router {
	t.Helper()
	r, err := llmrouter.New(providers, labels, llmrouter.Config{}, slog.Default())
	if err != nil {
		t.Fatalf("llmrouter.New: %v", err)
	}
	return r
}

// Covers AC-38.006, AC-38.019
// Covers AC-34.001, AC-34.013, AC-34.014 (REQ-34.001, REQ-34.013, REQ-34.014): qualifying tool failure does not advance provider; replaces EP-006 escalation tests.
func TestHandleMessage_toolFailure_doesNotAdvanceProvider(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: errors.New("transient node error")}
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	call0, call1 := 0, 0
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		call0++
		if call0 == 1 {
			return &llm.CompletionResult{
				Content:   "",
				ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "recovered on same provider", Usage: llm.Usage{}}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		call1++
		return &llm.CompletionResult{Content: "wrong provider", Usage: llm.Usage{}}, nil
	}
	h := &conversationHandler{
		router:                     mustRouterMulti(t, []llm.Provider{p0, p1}, []string{"m0", "m1"}),
		catalog:                    catalog,
		nodeRunner:                 runner,
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "same provider") {
		t.Errorf("reply = %q, want recovery on first provider", reply)
	}
	if call0 < 2 {
		t.Errorf("p0 Complete calls = %d, want >= 2 (initial + follow-up on same index)", call0)
	}
	if call1 != 0 {
		t.Errorf("p1 Complete calls = %d, want 0 (no tool-path provider advance)", call1)
	}
}
