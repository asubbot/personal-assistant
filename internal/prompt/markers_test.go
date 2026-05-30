package prompt

import "testing"

// Independent golden literals (pre-EP-035 marker constants); S-001 non-tautological byte checks.
const (
	expectedBeginContext = "<<<PA_BEGIN_CONTEXT>>>"
	expectedEndContext   = "<<<PA_END_CONTEXT>>>"
	expectedBeginTools   = "<<<PA_BEGIN_TOOLS>>>"
	expectedEndTools     = "<<<PA_END_TOOLS>>>"
	expectedBeginSkills  = "<<<PA_BEGIN_SKILLS>>>"
	expectedEndSkills    = "<<<PA_END_SKILLS>>>"
)

// Covers AC-35.009: marker constants remain byte-identical after package merge.
// Supporting AC-35.020: EP-013 marker constant test retained under internal/prompt.
func TestMarkerConstants_byteIdentical(t *testing.T) {
	if BeginContext != expectedBeginContext {
		t.Fatalf("BeginContext drift")
	}
	if EndContext != expectedEndContext {
		t.Fatalf("EndContext drift")
	}
	if BeginTools != expectedBeginTools {
		t.Fatalf("BeginTools drift")
	}
	if EndTools != expectedEndTools {
		t.Fatalf("EndTools drift")
	}
	if BeginSkills != expectedBeginSkills {
		t.Fatalf("BeginSkills drift")
	}
	if EndSkills != expectedEndSkills {
		t.Fatalf("EndSkills drift")
	}
}

// Covers AC-13.001, AC-35.010: traceability for forbidden-marker line detection.
func TestTextContainsForbiddenMarkerLine_detectsExactLine(t *testing.T) {
	if !TextContainsForbiddenMarkerLine("hello\n" + BeginContext + "\nend") {
		t.Fatal("expected marker line detected")
	}
}

// Covers AC-35.007: merged internal/prompt exposes ForbiddenMarkerLines with all six canonical markers.
// Covers AC-35.010: ForbiddenMarkerLines content matches pre-EP-035 behaviour.
func TestForbiddenMarkerLines_allSixCanonical(t *testing.T) {
	got := ForbiddenMarkerLines()
	want := []string{BeginContext, EndContext, BeginTools, EndTools, BeginSkills, EndSkills}
	if len(got) != len(want) {
		t.Fatalf("ForbiddenMarkerLines len = %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("ForbiddenMarkerLines[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// Covers AC-35.010: forbidden-marker validation (trimmed line match).
func TestTextContainsForbiddenMarkerLine_trimmed(t *testing.T) {
	if !TextContainsForbiddenMarkerLine("  " + EndTools + "  ") {
		t.Fatal("expected trimmed match")
	}
}

// Covers AC-35.010: forbidden-marker validation (negative cases).
func TestTextContainsForbiddenMarkerLine_negative(t *testing.T) {
	if TextContainsForbiddenMarkerLine("<<<PA_BEGIN_CONTEXT>>> not alone") {
		t.Fatal("partial line on same line should not match policy (line-based)")
	}
	if TextContainsForbiddenMarkerLine("safe text") {
		t.Fatal("unexpected match")
	}
}
