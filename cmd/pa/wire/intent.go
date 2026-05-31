package wire

import (
	"log/slog"
	"pa/internal/config"
	"pa/internal/intent"
)

// BuildIntentClassifier constructs the EP-017/EP-036 heuristic cascade from config. Returns nil when disabled.
func BuildIntentClassifier(cfg *config.Config, logger *slog.Logger) intent.Classifier {
	ic := cfg.IntentClassifier
	if ic == nil || !ic.Enabled {
		return nil
	}
	var heuristic *intent.HeuristicClassifier
	if ic.Heuristic != nil {
		heuristic = intent.NewHeuristicClassifier(
			ic.Heuristic.SimplePatterns,
			ic.Heuristic.FullPatterns,
			ic.Heuristic.MaxSimpleLen,
		)
	}
	logger.Info("intent classifier enabled", "heuristic", heuristic != nil)
	return intent.NewCascadeClassifier(heuristic, logger)
}
