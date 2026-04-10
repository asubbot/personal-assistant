package systemprompt

import (
	"pa/internal/promptmarkers"
	"strings"
	"testing"
)

func TestWrapRetrievedContext_nonEmpty(t *testing.T) {
	// Covers AC-13.004
	out := WrapRetrievedContext("- item one\n")
	if !strings.Contains(out, promptmarkers.BeginRetrievedContext) {
		t.Fatal("missing begin marker")
	}
	if !strings.Contains(out, promptmarkers.EndRetrievedContext) {
		t.Fatal("missing end marker")
	}
	if !strings.Contains(out, "item one") {
		t.Fatal("missing body")
	}
}

func TestWrapToolInstructions(t *testing.T) {
	// Covers AC-13.005
	out := WrapToolInstructions("do the thing")
	if !strings.Contains(out, promptmarkers.BeginToolInstructions) || !strings.Contains(out, promptmarkers.EndToolInstructions) {
		t.Fatal(out)
	}
}

func TestTrustPolicyPrefix(t *testing.T) {
	if TrustPolicy == "" {
		t.Fatal("empty trust policy")
	}
}
