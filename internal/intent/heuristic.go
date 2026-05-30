package intent

import (
	"regexp"
	"unicode/utf8"
)

// HeuristicResult is the outcome of the heuristic classification stage.
type HeuristicResult struct {
	Tier      Tier
	Confident bool
}

// HeuristicClassifier evaluates a message against compiled regex patterns (REQ-17.004–REQ-17.006, EP-036).
type HeuristicClassifier struct {
	simplePatterns []*regexp.Regexp
	fullPatterns   []*regexp.Regexp
	maxSimpleLen   int
}

// NewHeuristicClassifier compiles pattern strings into a heuristic classifier.
// Callers must validate patterns at config load time (fail fast).
func NewHeuristicClassifier(simplePatterns, fullPatterns []string, maxSimpleLen int) *HeuristicClassifier {
	h := &HeuristicClassifier{maxSimpleLen: maxSimpleLen}
	for _, p := range simplePatterns {
		h.simplePatterns = append(h.simplePatterns, regexp.MustCompile("(?i)"+p))
	}
	for _, p := range fullPatterns {
		h.fullPatterns = append(h.fullPatterns, regexp.MustCompile("(?i)"+p))
	}
	return h
}

// Classify assigns a tier based on patterns and message length.
// Order: length → simple → full → ambiguous (EP-036).
func (h *HeuristicClassifier) Classify(message string) HeuristicResult {
	if h.maxSimpleLen > 0 && utf8.RuneCountInString(message) > h.maxSimpleLen {
		return HeuristicResult{Tier: TierFull, Confident: true}
	}
	for _, re := range h.simplePatterns {
		if re.MatchString(message) {
			return HeuristicResult{Tier: TierSimple, Confident: true}
		}
	}
	for _, re := range h.fullPatterns {
		if re.MatchString(message) {
			return HeuristicResult{Tier: TierFull, Confident: true}
		}
	}
	return HeuristicResult{Tier: TierFull, Confident: false}
}
