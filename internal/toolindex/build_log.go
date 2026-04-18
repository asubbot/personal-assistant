package toolindex

import (
	"log/slog"
	"time"
)

// LogBuildOutcome logs tool index build result per REQ-04.025 / AC-04.021 and EP-029 lifecycle fields.
// On failure logs ERROR with reason; on success logs INFO with tool count and duration_ms.
// Lifecycle attribute names match internal/lifecyclelog (subsystem, lifecycle_phase, duration_ms).
func LogBuildOutcome(logger *slog.Logger, toolCount int, duration time.Duration, err error) {
	if logger == nil {
		return
	}
	base := []any{
		"lifecycle_event", true,
		"subsystem", "tool_index",
		"lifecycle_phase", "build",
	}
	if duration > 0 {
		base = append(base, "duration_ms", duration.Milliseconds())
	}
	if err != nil {
		logger.Error("lifecycle", append(base, "error", err, "tool_count", toolCount)...)
		return
	}
	logger.Info("lifecycle", append(base, "tool_count", toolCount, "outcome", "ok")...)
}
