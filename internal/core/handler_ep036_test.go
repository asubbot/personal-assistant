package core

import (
	"pa/internal/testutil"
	"testing"
)

// Covers AC-38.012, AC-38.019
// Covers AC-36.021
func TestEP036_validateCommandExitZero(t *testing.T) {
	root := moduleRootDir(t)
	testutil.RunValidateEpic(t, root, "EP-036")
}
