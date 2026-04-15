package intent

import (
	"context"
	"errors"
	"pa/internal/llm"
	"testing"
	"time"
)

type mockProvider struct {
	content string
	usage   llm.Usage
	err     error
	delay   time.Duration
}

func (m *mockProvider) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.err != nil {
		return nil, m.err
	}
	return &llm.CompletionResult{Content: m.content, Usage: m.usage}, nil
}

// Covers AC-17.008
func TestModel_ClassifySimple(t *testing.T) {
	mc := NewModelClassifier(&mockProvider{content: "simple", usage: llm.Usage{PromptTokens: 50, CompletionTokens: 1}}, nil, 5*time.Second)
	tier, err := mc.Classify(context.Background(), "привет")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tier != TierSimple {
		t.Errorf("tier = %s, want simple", tier)
	}
}

// Covers AC-17.008
func TestModel_ClassifyFull(t *testing.T) {
	mc := NewModelClassifier(&mockProvider{content: "full"}, nil, 5*time.Second)
	tier, err := mc.Classify(context.Background(), "напомни")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tier != TierFull {
		t.Errorf("tier = %s, want full", tier)
	}
}

// Covers AC-17.011
func TestModel_UnparseableResponse(t *testing.T) {
	mc := NewModelClassifier(&mockProvider{content: "maybe"}, nil, 5*time.Second)
	_, err := mc.Classify(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for unparseable response")
	}
}

// Covers AC-17.011
// Supporting AC-17.008
func TestModel_ThinkBlockStripped(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    Tier
	}{
		{"think then simple", "<think>reasoning here</think>simple", TierSimple},
		{"think then full", "<think>\nlong reasoning\n</think>\nfull", TierFull},
		{"think only empty", "<think>tokens</think>", ""},
		{"multiline think simple", "<think>\nstep1\nstep2\n</think> simple", TierSimple},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mc := NewModelClassifier(&mockProvider{content: tc.content}, nil, 5*time.Second)
			tier, err := mc.Classify(context.Background(), "test")
			if tc.want == "" {
				if err == nil {
					t.Fatal("expected error for empty after stripping think block")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tier != tc.want {
				t.Errorf("tier = %s, want %s", tier, tc.want)
			}
		})
	}
}

// Covers AC-17.011
func TestModel_ProviderError(t *testing.T) {
	mc := NewModelClassifier(&mockProvider{err: errors.New("connection refused")}, nil, 5*time.Second)
	_, err := mc.Classify(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

// Covers AC-17.011
func TestModel_Timeout(t *testing.T) {
	mc := NewModelClassifier(&mockProvider{content: "simple", delay: 2 * time.Second}, nil, 50*time.Millisecond)
	_, err := mc.Classify(context.Background(), "test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// Covers AC-17.009
func TestModel_PromptContainsOnlyMessageAndTiers(t *testing.T) {
	var captured []llm.Message
	mp := &capturingProvider{inner: &mockProvider{content: "simple"}}
	mc := NewModelClassifier(mp, nil, 5*time.Second)
	_, _ = mc.Classify(context.Background(), "test message")
	captured = mp.messages
	if len(captured) != 1 {
		t.Fatalf("expected 1 message, got %d", len(captured))
	}
	if captured[0].Role != "user" {
		t.Errorf("role = %s, want user", captured[0].Role)
	}
}

type capturingProvider struct {
	inner    *mockProvider
	messages []llm.Message
}

func (c *capturingProvider) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	c.messages = append(c.messages, messages...)
	return c.inner.Complete(ctx, messages, opts)
}
