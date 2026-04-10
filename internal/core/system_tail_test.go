package core

import (
	"context"
	"log/slog"
	"pa/internal/runtimeskills"
	"strings"
	"testing"
	"unicode/utf8"
)

// Covers AC-13.014, REQ-13.012: when the dynamic tail exceeds budget, whole lowest-ranked selected skills are removed first.
func TestFitDynamicTail_trimsSkillsFromEnd(t *testing.T) {
	a := &runtimeskills.Package{ID: "a", Name: "A", Description: "d", Body: "aaaa"}
	b := &runtimeskills.Package{ID: "b", Name: "B", Description: "d", Body: "bbbbbbbb"}
	h := &conversationHandler{}
	stBoth := &tailFitState{skills: []*runtimeskills.Package{a, b}}
	stOne := &tailFitState{skills: []*runtimeskills.Package{a}}
	bothRunes := utf8.RuneCountInString(h.buildDynamicTailString(stBoth))
	oneRunes := utf8.RuneCountInString(h.buildDynamicTailString(stOne))
	if oneRunes >= bothRunes {
		t.Fatalf("test setup: expected one skill smaller than two (one=%d both=%d)", oneRunes, bothRunes)
	}
	st := &tailFitState{skills: []*runtimeskills.Package{a, b}}
	h.fitDynamicTailToBudget(context.Background(), st, oneRunes)
	if len(st.skills) != 1 || st.skills[0].ID != "a" {
		t.Fatalf("got skills %+v", st.skills)
	}
}

func TestFitDynamicTail_logsWhenTrimmed(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	h := &conversationHandler{logger: logger, catalog: nil}
	p := &runtimeskills.Package{ID: "x", Name: "X", Description: "d", Body: strings.Repeat("Z", 800)}
	st := &tailFitState{skills: []*runtimeskills.Package{p}}
	h.fitDynamicTailToBudget(context.Background(), st, 80)
	var found bool
	for _, r := range cap.records {
		if r.msg == "system dynamic tail trimmed to rune budget" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected Info log for tail trim")
	}
	if len(st.skills) != 0 {
		t.Fatalf("skills = %d, want 0", len(st.skills))
	}
}
