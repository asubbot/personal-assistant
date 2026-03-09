package llm

import (
	"fmt"
	"pa/internal/config"
	"strings"
)

// NewProvider returns a Provider for the given config entry.
// Supports "openai", "openai-compatible" (HTTP chat completions; API key from api_key_path when set),
// and "ollama" (OpenAI-compatible endpoint, no API key).
func NewProvider(cfg *config.LLMProvider) (Provider, error) {
	typ := strings.TrimSpace(strings.ToLower(cfg.Type))
	switch typ {
	case "openai", "openai-compatible", "ollama":
		return NewOpenAICompatible(cfg)
	default:
		return nil, fmt.Errorf("unsupported llm provider type %q (supported: openai, openai-compatible, ollama)", cfg.Type)
	}
}
