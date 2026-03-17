package llm

import (
	"context"
	"testing"
)

// mockProviderWithTools implements Provider; when opts has Tools, returns result with ToolCalls (Covers AC-04.009).
type mockProviderWithTools struct{}

func (m *mockProviderWithTools) Complete(_ context.Context, _ []Message, opts *CompletionOptions) (*CompletionResult, error) {
	result := &CompletionResult{Content: "ok", Usage: Usage{}}
	if opts != nil && len(opts.Tools) > 0 {
		first := opts.Tools[0]
		result.ToolCalls = []ToolCall{
			{ID: "call-1", Name: first.Name, Arguments: "{}"},
		}
	}
	return result, nil
}

// Covers AC-04.009: provider interface accepts optional tools payload and returns tool_calls; mock can be used in tests.
func TestProvider_optionalToolsAndToolCalls_mockReturnsToolCalls(t *testing.T) {
	prov := &mockProviderWithTools{}
	opts := &CompletionOptions{
		Tools: []ToolDef{
			{Name: "run_on_node", Description: "Run command on node", Parameters: `{"type":"object"}`},
		},
	}
	result, err := prov.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result == nil {
		t.Fatal("Complete: nil result")
	}
	if result.ToolCalls == nil {
		t.Fatal("Complete: expected ToolCalls when opts.Tools set, got nil")
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("Complete: len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}
	tc := result.ToolCalls[0]
	if tc.ID != "call-1" || tc.Name != "run_on_node" || tc.Arguments != "{}" {
		t.Errorf("Complete: ToolCalls[0] = %+v", tc)
	}
}

// Covers AC-04.009: when opts is nil or has no tools, provider returns result with ToolCalls nil.
func TestProvider_optionalToolsAndToolCalls_mockReturnsNilWhenNoTools(t *testing.T) {
	prov := &mockProviderWithTools{}
	result, err := prov.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.ToolCalls != nil {
		t.Errorf("Complete(opts=nil): ToolCalls = %v, want nil", result.ToolCalls)
	}
	result2, err := prov.Complete(context.Background(), nil, &CompletionOptions{})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result2.ToolCalls != nil {
		t.Errorf("Complete(opts.Tools empty): ToolCalls = %v, want nil", result2.ToolCalls)
	}
}
