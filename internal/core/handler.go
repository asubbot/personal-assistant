package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/llm"
	"strings"
	"unicode/utf8"
)

// conversationHandler implements MessageHandler by calling the LLM provider (no memory/vector/tools yet).
type conversationHandler struct {
	provider         llm.Provider
	logger           *slog.Logger
	maxMessageLength int // 0 = no limit; messages longer than this (in runes) are rejected
}

// HandleMessage sends the user message to the LLM and returns the assistant reply.
// Empty or whitespace-only text is rejected with a clear message.
// If maxMessageLength > 0 and text exceeds it (in runes), the message is rejected (no LLM call).
func (h *conversationHandler) HandleMessage(ctx context.Context, _ int64, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "Please send a non-empty message.", nil
	}
	if h.maxMessageLength > 0 && utf8.RuneCountInString(text) > h.maxMessageLength {
		return fmt.Sprintf("Message is too long. Maximum length is %d characters.", h.maxMessageLength), nil
	}
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
