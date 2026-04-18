//go:build !e2e

package e2e

import "testing"

// TestPackageReservedForE2EBuildTag keeps ./tests/e2e a valid package when the e2e tag is off.
// End-to-end scenarios run via: make test-e2e (see Makefile).
//
// Covers AC-25.002
func TestPackageReservedForE2EBuildTag(t *testing.T) {}
