// Package promptmarkers defines canonical PA system-block marker lines (EP-013).
package promptmarkers

import (
	"bufio"
	"strings"
)

// Block marker line constants (exact bytes; EP-013 / analytics §8.5.1).
const (
	BeginContext = "<<<PA_BEGIN_CONTEXT>>>"
	EndContext   = "<<<PA_END_CONTEXT>>>"
	BeginTools   = "<<<PA_BEGIN_TOOLS>>>"
	EndTools     = "<<<PA_END_TOOLS>>>"
	BeginSkills  = "<<<PA_BEGIN_SKILLS>>>"
	EndSkills    = "<<<PA_END_SKILLS>>>"
)

// ForbiddenMarkerLines returns all canonical marker lines that must not appear
// as a full line (after trim) in SKILL.md or in text indexed into vector stores.
func ForbiddenMarkerLines() []string {
	return []string{
		BeginContext,
		EndContext,
		BeginTools,
		EndTools,
		BeginSkills,
		EndSkills,
	}
}

// TextContainsForbiddenMarkerLine reports whether any line in text, after
// strings.TrimSpace per line, equals one of the canonical marker lines.
func TextContainsForbiddenMarkerLine(text string) bool {
	set := make(map[string]struct{}, len(ForbiddenMarkerLines())*2)
	for _, line := range ForbiddenMarkerLines() {
		set[line] = struct{}{}
	}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		if _, ok := set[strings.TrimSpace(sc.Text())]; ok {
			return true
		}
	}
	return false
}
