// Package lifecyclelog emits structured slog records for background subsystem lifecycle boundaries (EP-029).
package lifecyclelog

import (
	"log/slog"
	"time"
)

const (
	attrLifecycleEvent = "lifecycle_event"
	attrSubsystem      = "subsystem"
	attrPhase          = "lifecycle_phase"
	attrDurationMs     = "duration_ms"
)

// Info logs a successful lifecycle boundary with optional duration.
func Info(logger *slog.Logger, subsystem, phase string, duration time.Duration, msg string, args ...any) {
	if logger == nil {
		return
	}
	a := append([]any{attrLifecycleEvent, true, attrSubsystem, subsystem, attrPhase, phase}, args...)
	if duration > 0 {
		a = append(a, attrDurationMs, duration.Milliseconds())
	}
	logger.Info(msg, a...)
}

// Error logs a failed lifecycle boundary with optional duration.
func Error(logger *slog.Logger, subsystem, phase string, duration time.Duration, err error, msg string, args ...any) {
	if logger == nil {
		return
	}
	a := append([]any{attrLifecycleEvent, true, attrSubsystem, subsystem, attrPhase, phase, "error", err}, args...)
	if duration > 0 {
		a = append(a, attrDurationMs, duration.Milliseconds())
	}
	logger.Error(msg, a...)
}
