package promptmarkers

import "testing"

// Covers AC-13.001: traceability for TestTextContainsForbiddenMarkerLine_detectsExactLine.
func TestTextContainsForbiddenMarkerLine_detectsExactLine(t *testing.T) {
	// Covers AC-13.001
	if !TextContainsForbiddenMarkerLine("hello\n" + BeginRetrievedContext + "\nend") {
		t.Fatal("expected marker line detected")
	}
}

func TestTextContainsForbiddenMarkerLine_trimmed(t *testing.T) {
	// Covers AC-13.001
	if !TextContainsForbiddenMarkerLine("  " + EndToolInstructions + "  ") {
		t.Fatal("expected trimmed match")
	}
}

func TestTextContainsForbiddenMarkerLine_negative(t *testing.T) {
	if TextContainsForbiddenMarkerLine("<<<PA_BEGIN_RETRIEVED_CONTEXT>>> not alone") {
		t.Fatal("partial line on same line should not match policy (line-based)")
	}
	if TextContainsForbiddenMarkerLine("safe text") {
		t.Fatal("unexpected match")
	}
}
