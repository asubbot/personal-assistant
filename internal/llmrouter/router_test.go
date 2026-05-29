package llmrouter

import (
	"context"
	"errors"
	"net"
	"pa/internal/llm"
	"strings"
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

// Covers AC-34.004, AC-34.014
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

// Covers AC-34.004
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

// Covers AC-34.004
func TestComplete_nilState_returnsError(t *testing.T) {
	p := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New([]llm.Provider{p}, []string{"a/m0"}, Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = r.Complete(context.Background(), nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Covers AC-34.004
func TestComplete_outOfRangeStateIndex_returnsError(t *testing.T) {
	p := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New([]llm.Provider{p}, []string{"a/m0"}, Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = r.Complete(context.Background(), &State{ActiveIndex: 7}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Covers AC-34.004
func TestComplete_maxAttemptsExceeded_stopsDeterministically(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("down")}}
	p1 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("down")}}
	r, err := New(
		[]llm.Provider{p0, p1},
		[]string{"a/m0", "b/m1"},
		Config{MaxAttemptsPerComplete: 1},
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var events []Event
	_, err = r.Complete(context.Background(), r.NewState(), nil, nil, func(e Event) {
		events = append(events, e)
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	last := events[len(events)-1]
	if last.Action != ActionStop {
		t.Errorf("last action = %s, want %s", last.Action, ActionStop)
	}
}

// Covers AC-34.004
func TestComplete_emitsSwitchEventPayload(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New([]llm.Provider{p0, p1}, []string{"a/m0", "b/m1"}, Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var events []Event
	_, err = r.Complete(context.Background(), r.NewState(), nil, nil, func(e Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events, got none")
	}
	e := events[0]
	if e.Phase != PhaseCompleteError || e.Action != ActionSwitchNextTransport {
		t.Errorf("event phase/action = %s/%s", e.Phase, e.Action)
	}
	if e.FromIndex != 0 || e.ToIndex != 1 || e.Attempt != 1 {
		t.Errorf("event indexes/attempt = from:%d to:%d attempt:%d", e.FromIndex, e.ToIndex, e.Attempt)
	}
}

type fakeNetErr struct{}

func (fakeNetErr) Error() string   { return "net" }
func (fakeNetErr) Timeout() bool   { return true }
func (fakeNetErr) Temporary() bool { return true }

var _ net.Error = fakeNetErr{}

// Covers AC-34.004
func TestClassifyCompleteError_networkAndTimeout(t *testing.T) {
	if got := ClassifyCompleteError(fakeNetErr{}); got != FailureClassTransportNetwork {
		t.Errorf("network class = %s", got)
	}
	if got := ClassifyCompleteError(context.DeadlineExceeded); got != FailureClassTransportTimeout {
		t.Errorf("timeout class = %s", got)
	}
}

// Covers AC-34.006
func TestNewState_alwaysStartsAtZero(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New([]llm.Provider{p0, p1}, []string{"a/m0", "b/m1"}, Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := r.NewState()
	_, err = r.Complete(context.Background(), st, nil, nil, nil)
	if err != nil {
		t.Fatalf("first Complete: %v", err)
	}
	if st.ActiveIndex != 1 {
		t.Fatalf("after transport fallback ActiveIndex = %d, want 1", st.ActiveIndex)
	}
	st2 := r.NewState()
	if st2.ActiveIndex != 0 {
		t.Errorf("new turn ActiveIndex = %d, want 0", st2.ActiveIndex)
	}
}

// Covers AC-34.004
func TestComplete_wrapsExceededAttemptsErrorText(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("down")}}
	p1 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("down")}}
	r, err := New(
		[]llm.Provider{p0, p1},
		[]string{"a/m0", "b/m1"},
		Config{MaxAttemptsPerComplete: 1},
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = r.Complete(context.Background(), r.NewState(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "exceeded max attempts") {
		t.Errorf("unexpected error text: %q", got)
	}
}
