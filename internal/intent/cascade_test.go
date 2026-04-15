package intent

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// Covers AC-17.010
func TestCascade_HeuristicConfident(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	c := NewCascadeClassifier(h, nil, nil)
	r := c.Classify(context.Background(), "привет")
	if r.Tier != TierSimple {
		t.Errorf("tier = %s, want simple", r.Tier)
	}
	if r.Stage != "heuristic" {
		t.Errorf("stage = %s, want heuristic", r.Stage)
	}
}

// Covers AC-17.010
func TestCascade_AmbiguousToModel(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	mp := &mockProvider{content: "full"}
	mc := NewModelClassifier(mp, nil, 5*time.Second)
	c := NewCascadeClassifier(h, mc, nil)
	r := c.Classify(context.Background(), "погода")
	if r.Tier != TierFull {
		t.Errorf("tier = %s, want full", r.Tier)
	}
	if r.Stage != "model" {
		t.Errorf("stage = %s, want model", r.Stage)
	}
}

// Covers AC-17.010
func TestCascade_ModelDisabled_DefaultFull(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	c := NewCascadeClassifier(h, nil, nil)
	r := c.Classify(context.Background(), "погода")
	if r.Tier != TierFull {
		t.Errorf("tier = %s, want full", r.Tier)
	}
	if r.Stage != "default" {
		t.Errorf("stage = %s, want default", r.Stage)
	}
}

// Covers AC-17.011
func TestCascade_ModelError_DefaultFull(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	mp := &mockProvider{err: errors.New("fail")}
	mc := NewModelClassifier(mp, nil, 5*time.Second)
	c := NewCascadeClassifier(h, mc, nil)
	r := c.Classify(context.Background(), "погода")
	if r.Tier != TierFull {
		t.Errorf("tier = %s, want full", r.Tier)
	}
	if r.Stage != "default" {
		t.Errorf("stage = %s, want default", r.Stage)
	}
}

// Covers AC-17.011
func TestCascade_ModelError_LogsWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	mp := &mockProvider{err: errors.New("connection refused")}
	mc := NewModelClassifier(mp, nil, 5*time.Second)
	c := NewCascadeClassifier(h, mc, logger)
	r := c.Classify(context.Background(), "погода")
	if r.Tier != TierFull {
		t.Errorf("tier = %s, want full", r.Tier)
	}
	logOutput := buf.String()
	if !strings.Contains(logOutput, "WARN") {
		t.Errorf("expected WARN log entry, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "connection refused") {
		t.Errorf("expected error details in log, got: %s", logOutput)
	}
}

// Covers AC-17.001
func TestCascade_BothNil_DefaultFull(t *testing.T) {
	c := NewCascadeClassifier(nil, nil, nil)
	r := c.Classify(context.Background(), "anything")
	if r.Tier != TierFull {
		t.Errorf("tier = %s, want full", r.Tier)
	}
	if r.Stage != "default" {
		t.Errorf("stage = %s, want default", r.Stage)
	}
}

// Supporting AC-17.016
func TestCascade_MessageLen(t *testing.T) {
	c := NewCascadeClassifier(nil, nil, nil)
	r := c.Classify(context.Background(), "привет")
	if r.MessageLen != 6 {
		t.Errorf("MessageLen = %d, want 6", r.MessageLen)
	}
}

// Covers AC-17.010
func TestCascade_ModelReturnsSimple(t *testing.T) {
	mp := &mockProvider{content: "simple"}
	mc := NewModelClassifier(mp, nil, 5*time.Second)
	c := NewCascadeClassifier(nil, mc, nil)
	r := c.Classify(context.Background(), "test")
	if r.Tier != TierSimple {
		t.Errorf("tier = %s, want simple", r.Tier)
	}
	if r.Stage != "model" {
		t.Errorf("stage = %s, want model", r.Stage)
	}
}
