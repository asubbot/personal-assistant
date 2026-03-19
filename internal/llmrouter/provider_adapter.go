package llmrouter

import (
	"context"
	"log/slog"
	"pa/internal/llm"
)

type providerAdapter struct {
	router *Router
}

// NewProviderAdapter returns an llm.Provider backed by Router transport routing.
// It uses router defaults (no tool escalation policy) and creates a fresh state per call.
func NewProviderAdapter(providers []llm.Provider, labels []string, logger *slog.Logger) (llm.Provider, error) {
	r, err := New(providers, labels, Config{}, logger)
	if err != nil {
		return nil, err
	}
	return &providerAdapter{router: r}, nil
}

func (p *providerAdapter) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	st := p.router.NewState()
	result, err := p.router.Complete(ctx, st, messages, opts, nil)
	if err != nil {
		return nil, err
	}
	if result != nil && result.Model == "" {
		result.Model = p.router.providerLabel(st.ActiveIndex)
	}
	return result, nil
}
