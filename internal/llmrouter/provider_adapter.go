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
// Each Complete starts at provider index 0 (routerCfg may set MaxAttemptsPerComplete).
func NewProviderAdapter(providers []llm.Provider, labels []string, routerCfg Config, logger *slog.Logger) (llm.Provider, error) {
	r, err := New(providers, labels, routerCfg, logger)
	if err != nil {
		return nil, err
	}
	return &providerAdapter{router: r}, nil
}

// SummarizeRouterConfig returns llmrouter.Config for summarization (memory job, -summarize). Always starts at provider index 0 (EP-034).
func SummarizeRouterConfig() Config {
	return Config{}
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
