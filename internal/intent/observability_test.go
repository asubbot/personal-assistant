package intent

import (
	"bytes"
	"context"
	"log/slog"
	"pa/internal/llm"
	"strings"
	"testing"
	"time"
)

// Covers AC-17.017
func TestModelClassifier_LogsUsageSeparately(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	mp := &mockProvider{
		content: "simple",
		usage:   llm.Usage{PromptTokens: 42, CompletionTokens: 1},
	}
	mc := NewModelClassifier(mp, logger, 5*time.Second)
	_, err := mc.Classify(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logOutput := buf.String()
	if !strings.Contains(logOutput, "intent_classifier_model") {
		t.Errorf("log should contain component=intent_classifier_model, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "prompt_tokens=42") {
		t.Errorf("log should contain prompt_tokens=42, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "completion_tokens=1") {
		t.Errorf("log should contain completion_tokens=1, got: %s", logOutput)
	}
}

// Covers AC-17.016
func TestCascadeClassifier_ResultContainsStageAndLen(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^hello$`}, nil, 40)
	c := NewCascadeClassifier(h, nil, nil)
	r := c.Classify(context.Background(), "hello")
	if r.Tier != TierSimple {
		t.Errorf("tier = %s, want simple", r.Tier)
	}
	if r.Stage != "heuristic" {
		t.Errorf("stage = %q, want heuristic", r.Stage)
	}
	if r.MessageLen != 5 {
		t.Errorf("message_len = %d, want 5", r.MessageLen)
	}
}
