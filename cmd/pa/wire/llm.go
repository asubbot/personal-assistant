package wire

import (
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/llmrouter"
	"strings"
)

// BuildLLMProviders builds one Provider per cfg.LLMProviders entry and parallel labels (Type/Model).
func BuildLLMProviders(cfg *config.Config) ([]llm.Provider, []string, error) {
	if len(cfg.LLMProviders) == 0 {
		return nil, nil, fmt.Errorf("no llm providers configured")
	}
	var providers []llm.Provider
	var labels []string
	for i := range cfg.LLMProviders {
		p, err := llm.NewProvider(&cfg.LLMProviders[i])
		if err != nil {
			return nil, nil, err
		}
		providers = append(providers, p)
		typ := strings.TrimSpace(strings.ToLower(cfg.LLMProviders[i].Type))
		model := strings.TrimSpace(cfg.LLMProviders[i].Model)
		if model == "" {
			model = "default"
		}
		labels = append(labels, typ+"/"+model)
	}
	return providers, labels, nil
}

// BuildAppLLM constructs conversation providers and a summarize-only adapter from cfg.
func BuildAppLLM(cfg *config.Config, logger *slog.Logger) ([]llm.Provider, []string, llm.Provider, error) {
	providers, labels, err := BuildLLMProviders(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	summarizeLLM, err := llmrouter.NewProviderAdapter(providers, labels, llmrouter.SummarizeRouterConfig(), logger)
	if err != nil {
		return nil, nil, nil, err
	}
	return providers, labels, summarizeLLM, nil
}
