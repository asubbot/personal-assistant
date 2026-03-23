package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core/toolfailure"
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"strings"
	"testing"
)

func boolAsPtr(b bool) *bool { return &b }

// ep006TestCatalog returns a minimal catalog with run_echo (same shape as handler_ep006_audit tests).
func ep006TestCatalog() *toolcatalog.Catalog {
	return &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
}

// ep006EscalationConfig returns tools.llm_escalation settings for tests.
func ep006EscalationConfig(enabled bool, max, baseline int) *config.ToolsConfig {
	return &config.ToolsConfig{
		LLMEscalation: &config.LLMEscalationConfig{
			Enabled: enabled, MaxPerUserMessage: max, BaselineIndex: baseline,
		},
	}
}

// Covers EP-006 integration-style wiring: core.Run builds handler; qualifying tool failure advances Complete to second provider (AC-06.006).
func TestRun_toolMayEscalate_advancesToSecondProvider(t *testing.T) {
	catalog := ep006TestCatalog()
	runner := &mockNodeRunner{err: toolfailure.MayEscalate(errors.New("transient node error"))}
	p0, p1, p2 := &mockProvider{}, &mockProvider{}, &mockProvider{}
	var c0, c1, c2 int
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c0++
		return &llm.CompletionResult{
			ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}},
		}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c1++
		return &llm.CompletionResult{Content: "recovered via Run wiring", Usage: llm.Usage{}}, nil
	}
	p2.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c2++
		return &llm.CompletionResult{Content: "wrong", Usage: llm.Usage{}}, nil
	}

	cfg := &config.Config{
		Version: 1,
		LLMProviders: []config.LLMProvider{
			{Type: "a", Model: "m0", SupportsTools: boolAsPtr(true), DefaultTemperature: 0.3, DefaultMaxTokens: 1024, SupportsJSONMode: true, DefaultResponseFormat: "text"},
			{Type: "b", Model: "m1", SupportsTools: boolAsPtr(true), DefaultTemperature: 0.3, DefaultMaxTokens: 1024, SupportsJSONMode: true, DefaultResponseFormat: "text"},
			{Type: "c", Model: "m2", SupportsTools: boolAsPtr(true), DefaultTemperature: 0.3, DefaultMaxTokens: 1024, SupportsJSONMode: true, DefaultResponseFormat: "text"},
		},
		Tools:               ep006EscalationConfig(true, 3, 0),
		ToolCatalog:         catalog,
		LogRedaction:        &config.LogRedaction{},
		PATimezone:          "UTC",
		ToolPreSelection:    &config.ToolPreSelection{ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50},
		ConversationContext: &config.ConversationContextConfig{InjectedContextMaxChars: 4000, VectorSearchTopK: 10},
	}
	logger := slog.Default()
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter,
		[]llm.Provider{p0, p1, p2}, []string{"m0", "m1", "m2"},
		nil, nil, nil, runner, nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if adapter.handler == nil {
		t.Fatal("handler not wired")
	}

	reply, err := adapter.handler.HandleMessage(context.Background(), 1, "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "recovered") {
		t.Errorf("reply = %q", reply)
	}
	if c0 != 1 || c1 != 1 || c2 != 0 {
		t.Errorf("Complete calls p0=%d p1=%d p2=%d, want 1,1,0", c0, c1, c2)
	}
}

// Covers EP-006: after escalation on message 1, message 2 starts from baseline provider again (AC-06.008) through Run-built handler.
func TestRun_twoMessages_resetsBaselineAfterEscalation(t *testing.T) {
	catalog := ep006TestCatalog()
	runner := &mockNodeRunner{err: toolfailure.MayEscalate(errors.New("fail"))}
	p0, p1, p2 := &mockProvider{}, &mockProvider{}, &mockProvider{}
	var c0, c1, c2 int
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c0++
		return &llm.CompletionResult{Content: "should not run p0 at baseline 1", Usage: llm.Usage{}}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c1++
		if c1 == 1 {
			return &llm.CompletionResult{
				ToolCalls: []llm.ToolCall{{ID: "t1", Name: "run_echo", Arguments: `{"msg":"a"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "second message from baseline p1", Usage: llm.Usage{}}, nil
	}
	p2.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c2++
		return &llm.CompletionResult{Content: "first message from p2 after escalate", Usage: llm.Usage{}}, nil
	}

	cfg := &config.Config{
		Version: 1,
		LLMProviders: []config.LLMProvider{
			{Type: "a", Model: "m0", SupportsTools: boolAsPtr(true), DefaultTemperature: 0.3, DefaultMaxTokens: 1024, SupportsJSONMode: true, DefaultResponseFormat: "text"},
			{Type: "b", Model: "m1", SupportsTools: boolAsPtr(true), DefaultTemperature: 0.3, DefaultMaxTokens: 1024, SupportsJSONMode: true, DefaultResponseFormat: "text"},
			{Type: "c", Model: "m2", SupportsTools: boolAsPtr(true), DefaultTemperature: 0.3, DefaultMaxTokens: 1024, SupportsJSONMode: true, DefaultResponseFormat: "text"},
		},
		Tools:               ep006EscalationConfig(true, 3, 1),
		ToolCatalog:         catalog,
		LogRedaction:        &config.LogRedaction{},
		PATimezone:          "UTC",
		ToolPreSelection:    &config.ToolPreSelection{ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50},
		ConversationContext: &config.ConversationContextConfig{InjectedContextMaxChars: 4000, VectorSearchTopK: 10},
	}
	adapter := &capturingAdapter{}
	err := Run(context.Background(), cfg, slog.Default(), adapter,
		[]llm.Provider{p0, p1, p2}, []string{"m0", "m1", "m2"},
		nil, nil, nil, runner, nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	_, err = adapter.handler.HandleMessage(context.Background(), 1, "one")
	if err != nil {
		t.Fatalf("HandleMessage 1: %v", err)
	}
	reply2, err := adapter.handler.HandleMessage(context.Background(), 1, "two")
	if err != nil {
		t.Fatalf("HandleMessage 2: %v", err)
	}
	if !strings.Contains(reply2, "second message from baseline p1") {
		t.Errorf("reply2 = %q", reply2)
	}
	// First user message: p1 (tool) then p2 (reply). Second: p1 only (no tool on second p1 call path — second call is c1==2)
	if c0 != 0 {
		t.Errorf("p0 Complete calls = %d, want 0 (baseline is 1)", c0)
	}
	if c1 != 2 {
		t.Errorf("p1 Complete calls = %d, want 2 (tool turn + second message start)", c1)
	}
	if c2 != 1 {
		t.Errorf("p2 Complete calls = %d, want 1", c2)
	}
}
