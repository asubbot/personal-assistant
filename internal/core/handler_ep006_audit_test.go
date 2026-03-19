package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core/toolfailure"
	"pa/internal/llm"
	"pa/internal/llmrouter"
	"pa/internal/toolcatalog"
	"pa/internal/vector"
	"strings"
	"testing"
)

func testRouter(t *testing.T, providers []llm.Provider, labels []string, esc *config.LLMEscalationConfig) *llmrouter.Router {
	t.Helper()
	r, err := llmrouter.New(providers, labels, llmrouter.Config{Escalation: esc}, slog.Default())
	if err != nil {
		t.Fatalf("llmrouter.New: %v", err)
	}
	return r
}

// Covers AC-06.003 (REQ-06.003, REQ-06.004): typed tool failures map to escalation qualification via errors.As (one action per typed outcome).
func TestEP006_classification_QualifiesForEscalation_table(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"MayEscalate", toolfailure.MayEscalate(errors.New("exec failed")), true},
		{"NoEscalate", toolfailure.NoEscalate(errors.New("policy")), false},
		{"plain error", errors.New("noderunner: boom"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toolfailure.QualifiesForEscalation(tt.err); got != tt.want {
				t.Errorf("QualifiesForEscalation() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Covers AC-06.006 (REQ-06.007): with three chain entries, first qualifying tool failure advances Complete to the second provider (strict order, no skip).
func TestHandleMessage_escalation_threeProviders_secondReceivesNextComplete(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: toolfailure.MayEscalate(errors.New("transient node error"))}
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	p2 := &mockProvider{}
	call0, call1, call2 := 0, 0, 0
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		call0++
		return &llm.CompletionResult{
			Content:   "",
			ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}},
		}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		call1++
		return &llm.CompletionResult{Content: "recovered on second model", Usage: llm.Usage{}}, nil
	}
	p2.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		call2++
		return &llm.CompletionResult{Content: "should not run", Usage: llm.Usage{}}, nil
	}
	h := &conversationHandler{
		router:                     testRouter(t, []llm.Provider{p0, p1, p2}, []string{"m0", "m1", "m2"}, &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 3, BaselineIndex: 0}),
		escalation:                 &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 3, BaselineIndex: 0},
		catalog:                    catalog,
		nodeRunner:                 runner,
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "recovered") {
		t.Errorf("reply = %q", reply)
	}
	if call0 != 1 || call1 != 1 || call2 != 0 {
		t.Errorf("Complete calls p0=%d p1=%d p2=%d, want 1,1,0", call0, call1, call2)
	}
}

// Covers AC-06.007 (REQ-06.008): when max_per_user_message is 0, qualifying failure does not advance provider (exhausted budget / stop path).
func TestHandleMessage_escalation_maxZero_noAdvance(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: toolfailure.MayEscalate(errors.New("fail"))}
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	c0, c1 := 0, 0
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c0++
		if c0 == 1 {
			return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
		}
		return &llm.CompletionResult{Content: "after same provider", Usage: llm.Usage{}}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c1++
		return &llm.CompletionResult{Content: "wrong provider", Usage: llm.Usage{}}, nil
	}
	h := &conversationHandler{
		router:                     testRouter(t, []llm.Provider{p0, p1}, []string{"a", "b"}, &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 0, BaselineIndex: 0}),
		escalation:                 &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 0, BaselineIndex: 0},
		catalog:                    catalog,
		nodeRunner:                 runner,
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "x")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if c1 != 0 {
		t.Errorf("p1 Complete calls = %d, want 0 (no advance when max escalations is 0)", c1)
	}
	if c0 < 2 {
		t.Errorf("p0 Complete calls = %d, want >= 2 (follow-up on same index)", c0)
	}
	if !strings.Contains(reply, "same provider") {
		t.Errorf("reply = %q", reply)
	}
}

// Covers AC-06.007 (REQ-06.008): at last chain entry, qualifying failure does not advance; next Complete stays on last provider.
func TestHandleMessage_escalation_atLastProvider_noFurtherAdvance(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: toolfailure.MayEscalate(errors.New("fail"))}
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	p1Calls := 0
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		p1Calls++
		if p1Calls == 1 {
			return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "c2", Name: "run_echo", Arguments: `{"msg":"b"}`}}}, nil
		}
		return &llm.CompletionResult{Content: "p1 final after last-provider stall", Usage: llm.Usage{}}, nil
	}
	h := &conversationHandler{
		router:                     testRouter(t, []llm.Provider{p0, p1}, []string{"m0", "m1"}, &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 5, BaselineIndex: 0}),
		escalation:                 &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 5, BaselineIndex: 0},
		catalog:                    catalog,
		nodeRunner:                 runner,
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "x")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if p1Calls != 2 {
		t.Errorf("p1 Complete calls = %d, want 2 (tool round then final on same last index)", p1Calls)
	}
	if !strings.Contains(reply, "p1 final") {
		t.Errorf("reply = %q, want final from last provider", reply)
	}
}

// Covers AC-06.008 (REQ-06.009): each new HandleMessage resets to configured baseline_index for the first Complete.
func TestHandleMessage_escalation_eachMessageStartsFromBaseline(t *testing.T) {
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	p2 := &mockProvider{}
	var firstIdx []int
	makeFn := func(idx int, p *mockProvider) {
		p.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
			firstIdx = append(firstIdx, idx)
			return &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}, nil
		}
	}
	makeFn(0, p0)
	makeFn(1, p1)
	makeFn(2, p2)
	h := &conversationHandler{
		router:                     testRouter(t, []llm.Provider{p0, p1, p2}, []string{"m0", "m1", "m2"}, &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 2, BaselineIndex: 1}),
		escalation:                 &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 2, BaselineIndex: 1},
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}
	if _, err := h.HandleMessage(context.Background(), 1, "a"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := h.HandleMessage(context.Background(), 1, "b"); err != nil {
		t.Fatalf("second: %v", err)
	}
	// First call of each message should hit provider at baseline index 1 only.
	if len(firstIdx) < 2 {
		t.Fatalf("firstIdx = %v, want at least two first-complete markers", firstIdx)
	}
	if firstIdx[0] != 1 || firstIdx[1] != 1 {
		t.Errorf("first Complete provider indices = %v, want first two entries 1,1 (baseline)", firstIdx)
	}
}

// Covers AC-06.009 (REQ-06.010–06.012): escalation log line includes classification fields (no secret values in this scenario).
func TestHandleMessage_escalation_logContainsPolicyFields(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: toolfailure.MayEscalate(errors.New("fail"))}
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		return &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}, nil
	}
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	h := &conversationHandler{
		router:                     testRouter(t, []llm.Provider{p0, p1}, []string{"m0", "m1"}, &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 2, BaselineIndex: 0}),
		escalation:                 &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 2, BaselineIndex: 0},
		catalog:                    catalog,
		nodeRunner:                 runner,
		logger:                     logger,
		firstProviderSupportsTools: true,
	}
	if _, err := h.HandleMessage(context.Background(), 1, "x"); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var saw bool
	for _, r := range cap.records {
		if r.msg != "llm tool escalation" {
			continue
		}
		saw = true
		if r.attrs["failure_class"] != "tool_execution" {
			t.Errorf("failure_class = %q, want tool_execution", r.attrs["failure_class"])
		}
		if r.attrs["action"] != "escalate_policy" {
			t.Errorf("action = %q, want escalate_policy", r.attrs["action"])
		}
		if r.attrs["from_index"] != "0" || r.attrs["to_index"] != "1" {
			t.Errorf("provider indices before=%q after=%q", r.attrs["from_index"], r.attrs["to_index"])
		}
	}
	if !saw {
		t.Fatal("expected llm tool escalation log record")
	}
}

// Covers AC-06.011 (REQ-06.014): escalation disabled — qualifying tool failure does not move Complete to a later chain entry.
func TestHandleMessage_escalationDisabled_qualifyingToolFailure_staysOnFirstProvider(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: toolfailure.MayEscalate(errors.New("fail"))}
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	c0, c1 := 0, 0
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c0++
		if c0 == 1 {
			return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
		}
		return &llm.CompletionResult{Content: "still on first", Usage: llm.Usage{}}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c1++
		return &llm.CompletionResult{Content: "wrong", Usage: llm.Usage{}}, nil
	}
	h := &conversationHandler{
		router:                     testRouter(t, []llm.Provider{p0, p1}, []string{"m0", "m1"}, &config.LLMEscalationConfig{Enabled: false, MaxPerUserMessage: 2, BaselineIndex: 0}),
		escalation:                 &config.LLMEscalationConfig{Enabled: false, MaxPerUserMessage: 2, BaselineIndex: 0},
		catalog:                    catalog,
		nodeRunner:                 runner,
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "x")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if c1 != 0 {
		t.Errorf("second provider Complete calls = %d, want 0 when escalation disabled", c1)
	}
	if c0 < 2 {
		t.Errorf("first provider calls = %d, want >= 2", c0)
	}
	if !strings.Contains(reply, "still on first") {
		t.Errorf("reply = %q", reply)
	}
}

// Covers AC-06.013 (REQ-06.016): Hermes parse failure on first completion retries Complete on next provider when escalation enabled.
func TestHandleMessage_textBasedHermes_parseFailure_escalatesToNextProvider(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	p0 := &mockProvider{}
	p1 := &mockProvider{}
	c0, c1 := 0, 0
	p0.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c0++
		return &llm.CompletionResult{Content: `<tool_call>broken`, Usage: llm.Usage{}}, nil
	}
	p1.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c1++
		return &llm.CompletionResult{Content: "Plain answer after escalation.", Usage: llm.Usage{}}, nil
	}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:                     testRouter(t, []llm.Provider{p0, p1}, []string{"weak", "strong"}, &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 2, BaselineIndex: 0}),
		escalation:                 &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 2, BaselineIndex: 0},
		catalog:                    catalog,
		nodeRunner:                 &mockNodeRunner{},
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if c0 != 1 || c1 != 1 {
		t.Errorf("Complete calls p0=%d p1=%d, want 1,1", c0, c1)
	}
	if !strings.Contains(reply, "Plain answer") {
		t.Errorf("reply = %q", reply)
	}
}
