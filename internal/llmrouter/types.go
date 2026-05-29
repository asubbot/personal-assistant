package llmrouter

import (
	"context"
	"log/slog"
)

// Action is a routing decision outcome.
type Action string

const (
	ActionRetrySame           Action = "retry_same"
	ActionSwitchNextTransport Action = "switch_next_transport"
	ActionStop                Action = "stop"
)

// FailureClass is a stable classification for routing decisions.
type FailureClass string

const (
	FailureClassTransportTimeout FailureClass = "transport_timeout"
	FailureClassTransportNetwork FailureClass = "transport_network"
	FailureClassTransport5xx     FailureClass = "transport_5xx"
	FailureClassCanceled         FailureClass = "context_canceled"
	FailureClassOther            FailureClass = "other"
)

// Phase identifies where the decision is made.
type Phase string

const (
	PhaseCompleteError Phase = "complete_error"
)

// State is mutable routing state for one user message (active provider index only).
type State struct {
	ActiveIndex int
}

// Event is emitted for each routing transition.
type Event struct {
	Phase             Phase
	FailureClass      string
	Action            Action
	FromIndex         int
	ToIndex           int
	Attempt           int
	FromProviderLabel string // label at FromIndex (provider before transition)
	ProviderLabel     string // label at ToIndex after transition (destination; kept name for backward compatibility)
}

// Config controls unified router behavior.
type Config struct {
	// MaxAttemptsPerComplete bounds retry/switch loops for one Complete call.
	// 0 means use router default based on provider count.
	MaxAttemptsPerComplete int
}

// LogAttrs returns structured log attrs for this event.
func (e Event) LogAttrs() []any {
	return []any{
		"phase", string(e.Phase),
		"failure_class", e.FailureClass,
		"action", string(e.Action),
		"from_index", e.FromIndex,
		"to_index", e.ToIndex,
		"attempt", e.Attempt,
		"from_provider", e.FromProviderLabel,
		"provider_label", e.ProviderLabel,
	}
}

// LogEvent writes a normalized routing event.
func LogEvent(logger *slog.Logger, e Event) {
	if logger == nil {
		return
	}
	level := slog.LevelInfo
	if e.Action == ActionStop {
		level = slog.LevelWarn
	}
	logger.Log(context.TODO(), level, "llm routing", e.LogAttrs()...)
}
