package llmrouter

import (
	"context"
	"log/slog"
	"pa/internal/config"
)

// Action is a routing decision outcome.
type Action string

const (
	ActionRetrySame           Action = "retry_same"
	ActionSwitchNextTransport Action = "switch_next_transport"
	ActionEscalatePolicy      Action = "escalate_policy"
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
	PhaseToolFailure   Phase = "tool_failure"
	PhaseHermesParse   Phase = "hermes_parse"
)

// State is mutable routing state for one user message.
type State struct {
	ActiveIndex int
	EscUsed     int
}

// Event is emitted for each routing transition.
type Event struct {
	Phase           Phase
	FailureClass    string
	Action          Action
	FromIndex       int
	ToIndex         int
	Attempt         int
	EscalationsUsed int
	ProviderLabel   string
}

// Config controls unified router behavior.
type Config struct {
	Escalation *config.LLMEscalationConfig
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
		"escalations_used", e.EscalationsUsed,
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
