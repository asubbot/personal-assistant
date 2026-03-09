package core

import (
	"context"
	"log/slog"
	"pa/internal/llm"
)

// conversationHandler implements MessageHandler by calling the LLM provider (no memory/vector/tools yet).
type conversationHandler struct {
	provider llm.Provider
	logger   *slog.Logger
}

// HandleMessage sends the user message to the LLM and returns the assistant reply.
func (h *conversationHandler) HandleMessage(ctx context.Context, _ int64, text string) (string, error) {
	messages := []llm.Message{
		{Role: "system", Content: "You are a helpful assistant. Reply concisely."},
		{Role: "user", Content: text},
	}
	result, err := h.provider.Complete(ctx, messages, nil)
	if err != nil {
		h.logger.Error("llm complete", "error", err)
		return "", err
	}
	return result.Content, nil
}
