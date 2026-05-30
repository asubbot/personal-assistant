package intent

import (
	"context"
	"testing"
)

// Covers AC-17.016, AC-36.007
func TestCascadeClassifier_ResultContainsStageAndLen(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^hello$`}, nil, 40)
	c := NewCascadeClassifier(h, nil)
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
