// Package promptmarkers defines canonical PA system-block marker lines (EP-013).
package promptmarkers

import (
	"bufio"
	"strings"
)

// Block marker line constants (exact bytes; EP-013 / analytics §8.5.1).
const (
	BeginRetrievedContext = "<<<PA_BEGIN_RETRIEVED_CONTEXT>>>"
	EndRetrievedContext   = "<<<PA_END_RETRIEVED_CONTEXT>>>"
	BeginToolInstructions = "<<<PA_BEGIN_TOOL_INSTRUCTIONS>>>"
	EndToolInstructions   = "<<<PA_END_TOOL_INSTRUCTIONS>>>"
	BeginRuntimeSkills    = "<<<PA_BEGIN_RUNTIME_SKILLS>>>"
	EndRuntimeSkills      = "<<<PA_END_RUNTIME_SKILLS>>>"
)

// ForbiddenMarkerLines returns all canonical marker lines that must not appear
// as a full line (after trim) in SKILL.md or in text indexed into vector stores.
func ForbiddenMarkerLines() []string {
	return []string{
		BeginRetrievedContext,
		EndRetrievedContext,
		BeginToolInstructions,
		EndToolInstructions,
		BeginRuntimeSkills,
		EndRuntimeSkills,
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
