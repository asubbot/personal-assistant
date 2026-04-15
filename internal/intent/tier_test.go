package intent

import "testing"

// Covers AC-17.001
func TestTierConstants(t *testing.T) {
	if TierSimple != "simple" {
		t.Errorf("TierSimple = %q, want %q", TierSimple, "simple")
	}
	if TierFull != "full" {
		t.Errorf("TierFull = %q, want %q", TierFull, "full")
	}
}
