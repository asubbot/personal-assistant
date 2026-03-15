package llm

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// mockProvider returns fixed result/err and counts calls.
type mockProvider struct {
	result *CompletionResult
	err    error
	calls  int
}

func (m *mockProvider) Complete(ctx context.Context, messages []Message, opts *CompletionOptions) (*CompletionResult, error) {
	m.calls++
	return m.result, m.err
}

func TestFallbackProvider_firstFailsRetryable_secondSucceeds(t *testing.T) {
	first := &mockProvider{err: &APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	second := &mockProvider{result: &CompletionResult{Content: "ok", Usage: Usage{}}}
	fb := NewFallbackProvider([]Provider{first, second}, []string{"a", "b"}, slog.Default())

	result, err := fb.Complete(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("Content = %q, want ok", result.Content)
	}
	if result.Model != "b" {
		t.Errorf("Model = %q, want b", result.Model)
	}
	if first.calls != 1 || second.calls != 1 {
		t.Errorf("calls: first=%d second=%d, want 1,1", first.calls, second.calls)
	}
}

func TestFallbackProvider_allFailRetryable(t *testing.T) {
	first := &mockProvider{err: &APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	second := &mockProvider{err: &APIError{StatusCode: 502, Err: errors.New("bad gateway")}}
	fb := NewFallbackProvider([]Provider{first, second}, nil, slog.Default())

	_, err := fb.Complete(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("Complete: expected error, got nil")
	}
	if first.calls != 1 || second.calls != 1 {
		t.Errorf("calls: first=%d second=%d, want 1,1", first.calls, second.calls)
	}
}

func TestFallbackProvider_nonRetryable_noRetry(t *testing.T) {
	first := &mockProvider{err: &APIError{StatusCode: 401, Err: errors.New("unauthorized")}}
	second := &mockProvider{result: &CompletionResult{Content: "ok", Usage: Usage{}}}
	fb := NewFallbackProvider([]Provider{first, second}, nil, slog.Default())

	_, err := fb.Complete(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("Complete: expected error, got nil")
	}
	if first.calls != 1 || second.calls != 0 {
		t.Errorf("calls: first=%d second=%d, want 1,0", first.calls, second.calls)
	}
}

func TestFallbackProvider_singleProvider(t *testing.T) {
	prov := &mockProvider{result: &CompletionResult{Content: "hi", Usage: Usage{}}}
	fb := NewFallbackProvider([]Provider{prov}, []string{"single"}, slog.Default())

	result, err := fb.Complete(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "hi" {
		t.Errorf("Content = %q, want hi", result.Content)
	}
	if result.Model != "single" {
		t.Errorf("Model = %q, want single", result.Model)
	}
	if prov.calls != 1 {
		t.Errorf("calls = %d, want 1", prov.calls)
	}
}

// slogCapture captures log level, message and attrs for assertion (REQ-031: fallback log).
type slogCapture struct {
	buf strings.Builder
}

func (s *slogCapture) Enabled(_ context.Context, level slog.Level) bool { return true }
func (s *slogCapture) Handle(_ context.Context, r slog.Record) error {
	s.buf.WriteString(r.Level.String())
	s.buf.WriteString(" " + r.Message)
	r.Attrs(func(a slog.Attr) bool {
		s.buf.WriteString(" " + a.Key + "=" + a.Value.String())
		return true
	})
	s.buf.WriteString("\n")
	return nil
}
func (s *slogCapture) WithAttrs([]slog.Attr) slog.Handler { return s }
func (s *slogCapture) WithGroup(string) slog.Handler      { return s }

// Covers AC-043, REQ-031: when fallback tries the next provider, app log records the switch with message and labels (failed_provider, next_provider).
func TestFallbackProvider_logsProviderSwitchWithLabels(t *testing.T) {
	capture := &slogCapture{}
	logger := slog.New(capture)

	first := &mockProvider{err: &APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	second := &mockProvider{result: &CompletionResult{Content: "ok", Usage: Usage{}}}
	fb := NewFallbackProvider([]Provider{first, second}, []string{"openai/gpt-4o", "ollama/llama3"}, logger)

	result, err := fb.Complete(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("Content = %q, want ok", result.Content)
	}

	logOut := capture.buf.String()
	if !strings.Contains(logOut, "llm provider failed, trying next") {
		t.Errorf("log must contain provider switch message; got: %s", logOut)
	}
	if !strings.Contains(logOut, "failed_provider") || !strings.Contains(logOut, "next_provider") {
		t.Errorf("log must contain failed_provider and next_provider; got: %s", logOut)
	}
	if !strings.Contains(logOut, "openai/gpt-4o") || !strings.Contains(logOut, "ollama/llama3") {
		t.Errorf("log must contain provider labels; got: %s", logOut)
	}
}

// Covers AC-043, REQ-031: when fallback tries next provider and labels are nil, app log still records the switch message (without failed_provider/next_provider).
func TestFallbackProvider_logsProviderSwitchWithoutLabels(t *testing.T) {
	capture := &slogCapture{}
	logger := slog.New(capture)

	first := &mockProvider{err: &APIError{StatusCode: 503, Err: errors.New("overloaded")}}
	second := &mockProvider{result: &CompletionResult{Content: "ok", Usage: Usage{}}}
	fb := NewFallbackProvider([]Provider{first, second}, nil, logger)

	result, err := fb.Complete(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("Content = %q, want ok", result.Content)
	}

	logOut := capture.buf.String()
	if !strings.Contains(logOut, "llm provider failed, trying next") {
		t.Errorf("log must contain provider switch message even when labels are nil; got: %s", logOut)
	}
	// When labels are nil, failed_provider/next_provider are not added
	if strings.Contains(logOut, "failed_provider") || strings.Contains(logOut, "next_provider") {
		t.Errorf("log must not contain failed_provider/next_provider when labels are nil; got: %s", logOut)
	}
}
