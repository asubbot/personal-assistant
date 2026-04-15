// Package intent provides a two-stage intent classifier for prompt optimization (EP-017).
package intent

import "context"

// Tier determines which prompt components are included in the main LLM call.
type Tier string

const (
	TierSimple Tier = "simple"
	TierFull   Tier = "full"
)

// Result holds the classification outcome for one user turn.
type Result struct {
	Tier       Tier
	Stage      string // "heuristic", "model", or "default"
	MessageLen int    // original message length in runes
}

// Classifier assigns an incoming user message to a complexity tier.
type Classifier interface {
	Classify(ctx context.Context, message string) Result
}
