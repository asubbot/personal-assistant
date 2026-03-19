package llmrouter

import (
	"context"
	"errors"
	"pa/internal/config"
	"pa/internal/llm"
	"testing"
)

type testProvider struct {
	result *llm.CompletionResult
	err    error
	calls  int
}

func (p *testProvider) Complete(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
	p.calls++
	return p.result, p.err
}

func TestComplete_retryableFirst_switchesToNext(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New([]llm.Provider{p0, p1}, []string{"a/m0", "b/m1"}, Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := r.NewState()
	result, err := r.Complete(context.Background(), st, nil, nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("content = %q, want ok", result.Content)
	}
	if st.ActiveIndex != 1 {
		t.Errorf("active index = %d, want 1", st.ActiveIndex)
	}
	if p0.calls != 1 || p1.calls != 1 {
		t.Errorf("calls p0=%d p1=%d, want 1,1", p0.calls, p1.calls)
	}
}

func TestComplete_nonRetryable_stopsImmediately(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 401, Err: errors.New("unauthorized")}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New([]llm.Provider{p0, p1}, []string{"a/m0", "b/m1"}, Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := r.NewState()
	_, err = r.Complete(context.Background(), st, nil, nil, nil)
	if err == nil {
		t.Fatal("Complete: expected error, got nil")
	}
	if p0.calls != 1 || p1.calls != 0 {
		t.Errorf("calls p0=%d p1=%d, want 1,0", p0.calls, p1.calls)
	}
}

func TestOnQualifyingFailure_respectsEscalationCap(t *testing.T) {
	p0 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	p2 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New(
		[]llm.Provider{p0, p1, p2},
		[]string{"a/m0", "b/m1", "c/m2"},
		Config{Escalation: &config.LLMEscalationConfig{Enabled: true, BaselineIndex: 0, MaxPerUserMessage: 1}},
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := r.NewState()
	if !r.OnQualifyingFailure(st, PhaseToolFailure, "tool_execution", nil) {
		t.Fatal("expected first qualifying failure to escalate")
	}
	if st.ActiveIndex != 1 || st.EscUsed != 1 {
		t.Errorf("state after first escalate = %+v, want index=1 esc=1", *st)
	}
	if r.OnQualifyingFailure(st, PhaseToolFailure, "tool_execution", nil) {
		t.Fatal("expected second qualifying failure to stop at cap")
	}
	if st.ActiveIndex != 1 || st.EscUsed != 1 {
		t.Errorf("state after second attempt = %+v, want unchanged", *st)
	}
}
