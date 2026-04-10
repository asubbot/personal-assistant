package llmrouter

import (
	"context"
	"errors"
	"pa/internal/config"
	"pa/internal/llm"
	"testing"
)

func TestProviderAdapter_retryableFallbackAndModelLabel(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter, err := NewProviderAdapter([]llm.Provider{p0, p1}, []string{"a/m0", "b/m1"}, Config{}, nil)
	if err != nil {
		t.Fatalf("NewProviderAdapter: %v", err)
	}
	result, err := adapter.Complete(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("content = %q, want ok", result.Content)
	}
	if result.Model != "b/m1" {
		t.Errorf("model = %q, want b/m1", result.Model)
	}
}

func TestProviderAdapter_doesNotOverrideExistingModel(t *testing.T) {
	p := &testProvider{result: &llm.CompletionResult{Content: "ok", Model: "provider/native"}}
	adapter, err := NewProviderAdapter([]llm.Provider{p}, []string{"a/m0"}, Config{}, nil)
	if err != nil {
		t.Fatalf("NewProviderAdapter: %v", err)
	}
	result, err := adapter.Complete(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Model != "provider/native" {
		t.Errorf("model = %q, want provider/native", result.Model)
	}
}

func TestProviderAdapter_startsAtBaselineIndex(t *testing.T) {
	p0 := &testProvider{result: &llm.CompletionResult{Content: "wrong"}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "from-baseline"}}
	adapter, err := NewProviderAdapter(
		[]llm.Provider{p0, p1},
		[]string{"a/m0", "b/m1"},
		Config{Escalation: &config.LLMEscalationConfig{Enabled: true, BaselineIndex: 1, MaxPerUserMessage: 2}},
		nil,
	)
	if err != nil {
		t.Fatalf("NewProviderAdapter: %v", err)
	}
	result, err := adapter.Complete(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if p0.calls != 0 {
		t.Errorf("provider index 0 calls = %d, want 0 (baseline is 1)", p0.calls)
	}
	if p1.calls != 1 {
		t.Errorf("provider index 1 calls = %d, want 1", p1.calls)
	}
	if result.Content != "from-baseline" {
		t.Errorf("content = %q", result.Content)
	}
}

func TestSummarizeRouterConfig_nilConfig_returnsEmpty(t *testing.T) {
	if c := SummarizeRouterConfig(nil); c.Escalation != nil {
		t.Fatalf("SummarizeRouterConfig(nil): want empty escalation, got %+v", c)
	}
}
