package intent

import "testing"

// Covers AC-17.004, AC-36.004
func TestHeuristic_GreetingSimple(t *testing.T) {
	h := NewHeuristicClassifier(
		[]string{`^(привет|hello|hi|hey)$`, `^ты (здесь|тут)\??$`},
		[]string{`(напомни|запусти)`},
		40,
	)
	for _, msg := range []string{"привет", "hello", "Hi", "ты здесь?", "ты тут"} {
		r := h.Classify(msg)
		if r.Tier != TierSimple || !r.Confident {
			t.Errorf("Classify(%q) = {%s, confident=%v}, want {simple, true}", msg, r.Tier, r.Confident)
		}
	}
}

// Covers AC-17.005, AC-36.004
func TestHeuristic_ToolIntentFull(t *testing.T) {
	h := NewHeuristicClassifier(
		[]string{`^(привет|hello)$`},
		[]string{`(напомни|запусти|найди)`},
		40,
	)
	for _, msg := range []string{"напомни что я говорил вчера", "запусти задачу", "найди файл"} {
		r := h.Classify(msg)
		if r.Tier != TierFull || !r.Confident {
			t.Errorf("Classify(%q) = {%s, confident=%v}, want {full, true}", msg, r.Tier, r.Confident)
		}
	}
}

// Covers AC-17.006, AC-36.004
func TestHeuristic_AmbiguousMessage(t *testing.T) {
	h := NewHeuristicClassifier(
		[]string{`^(привет|hello)$`},
		[]string{`(напомни|запусти)`},
		40,
	)
	r := h.Classify("погода")
	if r.Confident {
		t.Errorf("Classify(%q) = confident=%v, want false (ambiguous)", "погода", r.Confident)
	}
	if r.Tier != TierFull {
		t.Errorf("ambiguous default tier = %s, want full", r.Tier)
	}
}

// Covers AC-17.007
func TestHeuristic_NoExternalCalls(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^hi$`}, nil, 100)
	r := h.Classify("hi")
	if r.Tier != TierSimple {
		t.Fatalf("unexpected tier %s", r.Tier)
	}
}

// Supporting AC-17.005
func TestHeuristic_LongMessageFull(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^привет$`}, nil, 10)
	long := "это очень длинное сообщение которое точно длиннее десяти рун"
	r := h.Classify(long)
	if r.Tier != TierFull || !r.Confident {
		t.Errorf("long message: tier=%s confident=%v, want full/true", r.Tier, r.Confident)
	}
}

// Covers AC-36.005
func TestHeuristic_fullPatternsOnly_noFullLiteStep(t *testing.T) {
	h := NewHeuristicClassifier(
		[]string{`^simple$`},
		[]string{`^fullmsg`},
		100,
	)
	if r := h.Classify("fullmsg_extra"); r.Tier != TierFull || !r.Confident {
		t.Fatalf("full pattern match: got %+v", r)
	}
	if r := h.Classify("other"); r.Confident {
		t.Fatalf("unmatched message should be ambiguous, got %+v", r)
	}
}

// Supporting AC-17.004
func TestHeuristic_CaseInsensitive(t *testing.T) {
	h := NewHeuristicClassifier([]string{`^hello$`}, nil, 100)
	for _, msg := range []string{"HELLO", "Hello", "hElLo"} {
		r := h.Classify(msg)
		if r.Tier != TierSimple {
			t.Errorf("Classify(%q) = %s, want simple", msg, r.Tier)
		}
	}
}
