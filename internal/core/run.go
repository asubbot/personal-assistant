package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
)

// Run starts the application: wires the adapter to the conversation handler (LLM) and blocks until ctx is cancelled.
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger, adapter Adapter, llmProvider llm.Provider) error {
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
	handler := &conversationHandler{provider: llmProvider, logger: logger, maxMessageLength: maxLen}
	return adapter.Run(ctx, handler)
}
