package core

import (
	"pa/internal/testutil"
	"testing"
)

// Covers AC-38.004
func TestEP038_maxToolRoundsConstInHandlerGo(t *testing.T) {
	if maxToolRounds != 10 {
		t.Fatalf("maxToolRounds = %d, want 10", maxToolRounds)
	}
}

// Covers AC-38.020
func TestEP038_validateCommandExitZero(t *testing.T) {
	root := moduleRootDir(t)
	testutil.RunValidateEpic(t, root, "EP-038")
}
