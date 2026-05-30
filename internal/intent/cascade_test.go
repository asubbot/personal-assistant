package intent

import (
	"context"
	"testing"
)

// Covers AC-36.001, AC-36.002
func TestTierConstants_twoTiersOnly(t *testing.T) {
	if TierSimple != "simple" || TierFull != "full" {
		t.Fatalf("tiers = %q, %q; want simple and full only", TierSimple, TierFull)
	}
}

// Covers AC-17.010, AC-36.007
func TestCascade_HeuristicConfident(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	c := NewCascadeClassifier(h, nil)
	r := c.Classify(context.Background(), "привет")
	if r.Tier != TierSimple {
		t.Errorf("tier = %s, want simple", r.Tier)
	}
	if r.Stage != "heuristic" {
		t.Errorf("stage = %s, want heuristic", r.Stage)
	}
}

// Covers AC-36.006
func TestCascade_AmbiguousDefaultsToFull(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	c := NewCascadeClassifier(h, nil)
	r := c.Classify(context.Background(), "погода")
	if r.Tier != TierFull {
		t.Errorf("tier = %s, want full", r.Tier)
	}
	if r.Stage != "default" {
		t.Errorf("stage = %s, want default", r.Stage)
	}
}

// Covers AC-17.010, AC-36.006
func TestCascade_ModelDisabled_DefaultFull(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 40)
	c := NewCascadeClassifier(h, nil)
	r := c.Classify(context.Background(), "погода")
	if r.Tier != TierFull {
		t.Errorf("tier = %s, want full", r.Tier)
	}
	if r.Stage != "default" {
		t.Errorf("stage = %s, want default", r.Stage)
	}
}

// Covers AC-17.001, AC-36.006
func TestCascade_BothNil_DefaultFull(t *testing.T) {
	c := NewCascadeClassifier(nil, nil)
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
	c := NewCascadeClassifier(nil, nil)
	r := c.Classify(context.Background(), "привет")
	if r.MessageLen != 6 {
		t.Errorf("MessageLen = %d, want 6", r.MessageLen)
	}
}
