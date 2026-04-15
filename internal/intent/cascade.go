package intent

import (
	"context"
	"log/slog"
	"unicode/utf8"
)

// CascadeClassifier chains heuristic → model → default-full (REQ-17.010, REQ-17.011).
type CascadeClassifier struct {
	heuristic *HeuristicClassifier
	model     *ModelClassifier
	logger    *slog.Logger
}

// NewCascadeClassifier creates the two-stage cascade. Either or both stages may be nil.
func NewCascadeClassifier(heuristic *HeuristicClassifier, model *ModelClassifier, logger *slog.Logger) *CascadeClassifier {
	return &CascadeClassifier{heuristic: heuristic, model: model, logger: logger}
}

// Classify runs the cascade: heuristic (if present) → model (if present and heuristic ambiguous) → default full.
func (c *CascadeClassifier) Classify(ctx context.Context, message string) Result {
	msgLen := utf8.RuneCountInString(message)

	if c.heuristic != nil {
		hr := c.heuristic.Classify(message)
		if hr.Confident {
			return Result{Tier: hr.Tier, Stage: "heuristic", MessageLen: msgLen}
		}
	}

	if c.model != nil {
		tier, err := c.model.Classify(ctx, message)
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("intent classifier model stage failed, defaulting to full",
					"error", err,
				)
			}
		} else {
			return Result{Tier: tier, Stage: "model", MessageLen: msgLen}
		}
	}

	return Result{Tier: TierFull, Stage: "default", MessageLen: msgLen}
}
