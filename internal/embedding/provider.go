package embedding

import (
	"fmt"
	"pa/internal/config"
	"strings"
)

// NewEmbedder returns an Embedder for the given embedding provider config.
// Supports "openai", "openai-compatible", and "ollama" (same embeddings API).
func NewEmbedder(cfg *config.EmbeddingProvider) (Embedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("embedding: config is nil")
	}
	typ := strings.TrimSpace(strings.ToLower(cfg.Type))
	switch typ {
	case "openai", "openai-compatible", "ollama":
		return NewOpenAICompatible(cfg)
	default:
		return nil, fmt.Errorf("embedding: unsupported type %q (supported: openai, openai-compatible, ollama)", cfg.Type)
	}
}
