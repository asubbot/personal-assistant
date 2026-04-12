package llmrouter

import (
	"context"
	"errors"
	"net"
	"pa/internal/config"
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

// EP-006: transport fallback in Complete must not consume policy escalation budget (EscUsed).
// Covers AC-01.001: traceability for TestComplete_transportRetry_doesNotIncrementEscUsed_withEscalationConfigured.
func TestComplete_transportRetry_doesNotIncrementEscUsed_withEscalationConfigured(t *testing.T) {
	p0 := &testProvider{err: &llm.APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New(
		[]llm.Provider{p0, p1},
		[]string{"a/m0", "b/m1"},
		Config{Escalation: &config.LLMEscalationConfig{Enabled: true, BaselineIndex: 0, MaxPerUserMessage: 3}},
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := r.NewState()
	if st.EscUsed != 0 {
		t.Fatalf("initial EscUsed = %d, want 0", st.EscUsed)
	}
	result, err := r.Complete(context.Background(), st, nil, nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("content = %q", result.Content)
	}
	if st.ActiveIndex != 1 {
		t.Errorf("ActiveIndex = %d, want 1 (switched for transport)", st.ActiveIndex)
	}
	if st.EscUsed != 0 {
		t.Errorf("EscUsed = %d after transport switch, want 0 (policy budget untouched)", st.EscUsed)
	}
}

// Covers AC-01.001: traceability for TestComplete_retryableFirst_switchesToNext.
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

// Covers AC-01.001: traceability for TestComplete_nonRetryable_stopsImmediately.
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

// Covers AC-01.001: traceability for TestOnQualifyingFailure_respectsEscalationCap.
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

// Covers AC-01.001: traceability for TestComplete_nilState_returnsError.
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

// Covers AC-01.001: traceability for TestComplete_outOfRangeStateIndex_returnsError.
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

// Covers AC-01.001: traceability for TestComplete_maxAttemptsExceeded_stopsDeterministically.
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

// Covers AC-01.001: traceability for TestComplete_emitsSwitchEventPayload.
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

// Covers AC-01.001: traceability for TestOnQualifyingFailure_nilState_returnsFalse.
func TestOnQualifyingFailure_nilState_returnsFalse(t *testing.T) {
	p := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New([]llm.Provider{p}, []string{"a/m0"}, Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r.OnQualifyingFailure(nil, PhaseToolFailure, "x", nil) {
		t.Fatal("expected false for nil state")
	}
}

// Covers AC-01.001: traceability for TestOnQualifyingFailure_noNextProvider_returnsFalse.
func TestOnQualifyingFailure_noNextProvider_returnsFalse(t *testing.T) {
	p := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New(
		[]llm.Provider{p},
		[]string{"a/m0"},
		Config{Escalation: &config.LLMEscalationConfig{Enabled: true, BaselineIndex: 0, MaxPerUserMessage: 2}},
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := r.NewState()
	if r.OnQualifyingFailure(st, PhaseToolFailure, "tool_execution", nil) {
		t.Fatal("expected false when no next provider")
	}
}

type fakeNetErr struct{}

func (fakeNetErr) Error() string   { return "net" }
func (fakeNetErr) Timeout() bool   { return true }
func (fakeNetErr) Temporary() bool { return true }

var _ net.Error = fakeNetErr{}

// Covers AC-01.001: traceability for TestClassifyCompleteError_networkAndTimeout.
func TestClassifyCompleteError_networkAndTimeout(t *testing.T) {
	if got := ClassifyCompleteError(fakeNetErr{}); got != FailureClassTransportNetwork {
		t.Errorf("network class = %s", got)
	}
	if got := ClassifyCompleteError(context.DeadlineExceeded); got != FailureClassTransportTimeout {
		t.Errorf("timeout class = %s", got)
	}
}

// Covers AC-01.001: traceability for TestDecideToolFailure_matrix.
func TestDecideToolFailure_matrix(t *testing.T) {
	st := &State{ActiveIndex: 0, EscUsed: 0}
	if got := DecideToolFailure(st, false, 2, true); got != ActionStop {
		t.Errorf("disabled = %s", got)
	}
	if got := DecideToolFailure(st, true, 0, true); got != ActionStop {
		t.Errorf("max0 = %s", got)
	}
	if got := DecideToolFailure(st, true, 2, false); got != ActionStop {
		t.Errorf("no next = %s", got)
	}
	if got := DecideToolFailure(st, true, 2, true); got != ActionEscalatePolicy {
		t.Errorf("eligible = %s", got)
	}
}

// Covers AC-01.001: traceability for TestNewState_usesConfiguredBaseline.
func TestNewState_usesConfiguredBaseline(t *testing.T) {
	p0 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	p1 := &testProvider{result: &llm.CompletionResult{Content: "ok"}}
	r, err := New(
		[]llm.Provider{p0, p1},
		[]string{"a/m0", "b/m1"},
		Config{Escalation: &config.LLMEscalationConfig{Enabled: true, BaselineIndex: 1, MaxPerUserMessage: 2}},
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	st := r.NewState()
	if st.ActiveIndex != 1 {
		t.Errorf("baseline state index = %d, want 1", st.ActiveIndex)
	}
}

// Covers AC-01.001: traceability for TestComplete_wrapsExceededAttemptsErrorText.
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
