package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/memory"
	"pa/internal/vector"
)

// Run starts the application: wires the adapter to the conversation handler (LLM, memory, vector) and blocks until ctx is cancelled.
// memoryStore, vectorStore, and embedder are optional; when provided, the handler reads memory, runs semantic search, and indexes turns (REQ-006, REQ-007, REQ-018).
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger, adapter Adapter, llmProvider llm.Provider, memoryStore *memory.Store, vectorStore vector.Store, embedder embedding.Embedder) error {
	if adapter == nil {
		return fmt.Errorf("core: adapter is nil")
	}
	if llmProvider == nil {
		return fmt.Errorf("core: llm provider is nil")
	}
	maxLen := 0
	if cfg != nil && cfg.Telegram.MaxMessageLength > 0 {
		maxLen = cfg.Telegram.MaxMessageLength
	}
	handler := &conversationHandler{
		provider:         llmProvider,
		memoryStore:      memoryStore,
		vectorStore:      vectorStore,
		embedder:         embedder,
		logger:           logger,
		maxMessageLength: maxLen,
	}
	return adapter.Run(ctx, handler)
}
