package llmrouter

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
)

type providerAdapter struct {
	router *Router
}

// NewProviderAdapter returns an llm.Provider backed by Router transport routing.
// routerCfg supplies optional Escalation (e.g. tools.llm_escalation): when Enabled, each Complete starts at BaselineIndex like conversation turns; when absent or disabled, starts at index 0. Tool-escalation policy in the same struct is unused here (ProviderAdapter only calls Complete). Pass Config{} for legacy index-0 behavior.
func NewProviderAdapter(providers []llm.Provider, labels []string, routerCfg Config, logger *slog.Logger) (llm.Provider, error) {
	r, err := New(providers, labels, routerCfg, logger)
	if err != nil {
		return nil, err
	}
	return &providerAdapter{router: r}, nil
}

// SummarizeRouterConfig returns llmrouter.Config for summarization (memory job, -summarize): start at tools.llm_escalation.baseline_index when escalation is enabled, else index 0.
func SummarizeRouterConfig(cfg *config.Config) Config {
	if cfg == nil {
		return Config{}
	}
	esc := cfg.ToolsLLMEscalation()
	if esc == nil {
		return Config{}
	}
	return Config{Escalation: esc}
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
