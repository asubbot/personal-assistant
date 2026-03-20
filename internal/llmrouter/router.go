package llmrouter

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/llm"
)

// Router unifies transport fallback and escalation-state transitions.
type Router struct {
	providers []llm.Provider
	labels    []string
	cfg       Config
	logger    *slog.Logger
}

// New creates a Router for the configured providers.
func New(providers []llm.Provider, labels []string, cfg Config, logger *slog.Logger) (*Router, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("llmrouter: providers are required")
	}
	if len(labels) != len(providers) {
		return nil, fmt.Errorf("llmrouter: labels length %d != providers length %d", len(labels), len(providers))
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{providers: providers, labels: labels, cfg: cfg, logger: logger}, nil
}

// NewState returns initial routing state for a new user turn.
func (r *Router) NewState() *State {
	st := &State{ActiveIndex: 0, EscUsed: 0}
	if r.cfg.Escalation != nil && r.cfg.Escalation.Enabled {
		st.ActiveIndex = r.cfg.Escalation.BaselineIndex
	}
	return st
}

func (r *Router) providerLabel(idx int) string {
	if idx >= 0 && idx < len(r.labels) {
		return r.labels[idx]
	}
	return ""
}

func (r *Router) maxAttempts() int {
	if r.cfg.MaxAttemptsPerComplete > 0 {
		return r.cfg.MaxAttemptsPerComplete
	}
	return len(r.providers) + 2
}

// Complete runs one LLM completion with policy-based provider switching on retryable transport failures.
func (r *Router) Complete(ctx context.Context, st *State, messages []llm.Message, opts *llm.CompletionOptions, onEvent func(Event)) (*llm.CompletionResult, error) {
	if st == nil {
		return nil, fmt.Errorf("llmrouter: state is nil")
	}
	attempt := 0
	for {
		attempt++
		if attempt > r.maxAttempts() {
			idx := st.ActiveIndex
			e := Event{
				Phase:             PhaseCompleteError,
				FailureClass:      string(FailureClassOther),
				Action:            ActionStop,
				FromIndex:         idx,
				ToIndex:           idx,
				Attempt:           attempt,
				EscalationsUsed:   st.EscUsed,
				FromProviderLabel: r.providerLabel(idx),
				ProviderLabel:     r.providerLabel(idx),
			}
			if onEvent != nil {
				onEvent(e)
			}
			return nil, fmt.Errorf("llmrouter: exceeded max attempts (%d)", r.maxAttempts())
		}
		if st.ActiveIndex < 0 || st.ActiveIndex >= len(r.providers) {
			return nil, fmt.Errorf("llmrouter: provider index %d out of range [0,%d)", st.ActiveIndex, len(r.providers))
		}
		result, err := r.providers[st.ActiveIndex].Complete(ctx, messages, opts)
		if err == nil {
			return result, nil
		}
		class := ClassifyCompleteError(err)
		from := st.ActiveIndex
		action := DecideCompleteError(class, st.ActiveIndex+1 < len(r.providers))
		to := from
		if action == ActionSwitchNextTransport {
			to = from + 1
			st.ActiveIndex = to
		}
		e := Event{
			Phase:             PhaseCompleteError,
			FailureClass:      string(class),
			Action:            action,
			FromIndex:         from,
			ToIndex:           to,
			Attempt:           attempt,
			EscalationsUsed:   st.EscUsed,
			FromProviderLabel: r.providerLabel(from),
			ProviderLabel:     r.providerLabel(to),
		}
		if onEvent != nil {
			onEvent(e)
		}
		if action == ActionStop {
			return nil, err
		}
	}
}

// OnQualifyingFailure applies policy escalation for tool/hermes qualifying failures.
func (r *Router) OnQualifyingFailure(st *State, phase Phase, failureClass string, onEvent func(Event)) bool {
	if st == nil {
		return false
	}
	enabled := r.cfg.Escalation != nil && r.cfg.Escalation.Enabled
	maxEsc := 0
	if r.cfg.Escalation != nil {
		maxEsc = r.cfg.Escalation.MaxPerUserMessage
	}
	from := st.ActiveIndex
	action := DecideToolFailure(st, enabled, maxEsc, st.ActiveIndex+1 < len(r.providers))
	to := from
	if action == ActionEscalatePolicy {
		st.ActiveIndex++
		st.EscUsed++
		to = st.ActiveIndex
	}
	e := Event{
		Phase:             phase,
		FailureClass:      failureClass,
		Action:            action,
		FromIndex:         from,
		ToIndex:           to,
		EscalationsUsed:   st.EscUsed,
		FromProviderLabel: r.providerLabel(from),
		ProviderLabel:     r.providerLabel(to),
	}
	if onEvent != nil {
		onEvent(e)
	}
	return action == ActionEscalatePolicy
}
