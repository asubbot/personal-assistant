package llmrouter

import (
	"context"
	"errors"
	"pa/internal/llm"
	"testing"
)

func TestProviderAdapter_retryableFallbackAndModelLabel(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter, err := NewProviderAdapter([]llm.Provider{p0, p1}, []string{"a/m0", "b/m1"}, nil)
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
