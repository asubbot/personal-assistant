package toolindex

import "log/slog"

// LogBuildOutcome logs tool index build result per REQ-04.025 / AC-04.021.
// On failure logs ERROR with reason; on success logs INFO with tool count.
func LogBuildOutcome(logger *slog.Logger, toolCount int, err error) {
	if logger == nil {
		return
	}
	if err != nil {
		logger.Error("tool index build failed", "error", err)
		return
	}
	logger.Info("tool index built", "tools", toolCount)
}
