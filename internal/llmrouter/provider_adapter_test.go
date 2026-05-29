package llmrouter

import (
	"context"
	"errors"
	"pa/internal/llm"
	"testing"
)

// Covers AC-34.004
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

// Covers AC-34.004
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

// Covers AC-34.006
func TestProviderAdapter_eachCompleteStartsAtIndexZero(t *testing.T) {
	p0 := &testProvider{result: &llm.CompletionResult{Content: "first"}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "second"}}
	adapter, err := NewProviderAdapter([]llm.Provider{p0, p1}, []string{"a/m0", "b/m1"}, Config{}, nil)
	if err != nil {
		t.Fatalf("NewProviderAdapter: %v", err)
	}
	if _, err := adapter.Complete(context.Background(), nil, nil); err != nil {
		t.Fatalf("first Complete: %v", err)
	}
	if p0.calls != 1 || p1.calls != 0 {
		t.Fatalf("first call: p0=%d p1=%d, want 1,0", p0.calls, p1.calls)
	}
	if _, err := adapter.Complete(context.Background(), nil, nil); err != nil {
		t.Fatalf("second Complete: %v", err)
	}
	if p0.calls != 2 || p1.calls != 0 {
		t.Errorf("second call: p0=%d p1=%d, want 2,0 (new turn starts at index 0)", p0.calls, p1.calls)
	}
}

// Covers AC-34.005
func TestSummarizeRouterConfig_returnsEmptyConfig(t *testing.T) {
	if c := SummarizeRouterConfig(); c.MaxAttemptsPerComplete != 0 {
		t.Fatalf("SummarizeRouterConfig(): want empty config, got %+v", c)
	}
}
