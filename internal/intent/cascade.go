package intent

import (
	"context"
	"log/slog"
	"unicode/utf8"
)

// CascadeClassifier chains heuristic → default full (REQ-17.010, EP-036).
type CascadeClassifier struct {
	heuristic *HeuristicClassifier
	logger    *slog.Logger
}

// NewCascadeClassifier creates the heuristic-only cascade. Heuristic may be nil.
func NewCascadeClassifier(heuristic *HeuristicClassifier, logger *slog.Logger) *CascadeClassifier {
	return &CascadeClassifier{heuristic: heuristic, logger: logger}
}

// Classify runs the cascade: heuristic (if present) → default full when ambiguous.
func (c *CascadeClassifier) Classify(ctx context.Context, message string) Result {
	msgLen := utf8.RuneCountInString(message)

	if c.heuristic != nil {
		hr := c.heuristic.Classify(message)
		if hr.Confident {
			return Result{Tier: hr.Tier, Stage: "heuristic", MessageLen: msgLen}
		}
	}

	return Result{Tier: TierFull, Stage: "default", MessageLen: msgLen}
}
