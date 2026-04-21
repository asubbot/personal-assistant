package systemprompt

import (
	"pa/internal/promptmarkers"
	"strings"
	"testing"
)

// Covers AC-13.004: traceability for TestWrapRetrievedContext_nonEmpty.
func TestWrapRetrievedContext_nonEmpty(t *testing.T) {
	// Covers AC-13.004
	out := WrapRetrievedContext("- item one\n")
	if !strings.Contains(out, promptmarkers.BeginContext) {
		t.Fatal("missing begin marker")
	}
	if !strings.Contains(out, promptmarkers.EndContext) {
		t.Fatal("missing end marker")
	}
	if !strings.Contains(out, "item one") {
		t.Fatal("missing body")
	}
}

func TestWrapToolInstructions(t *testing.T) {
	// Covers AC-13.005
	out := WrapToolInstructions("do the thing")
	if !strings.Contains(out, promptmarkers.BeginTools) || !strings.Contains(out, promptmarkers.EndTools) {
		t.Fatal(out)
	}
}

func TestTrustPolicyPrefix(t *testing.T) {
	if TrustPolicy == "" {
		t.Fatal("empty trust policy")
	}
}
