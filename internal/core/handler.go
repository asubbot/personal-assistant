package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/memory"
	"pa/internal/vector"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	vectorSearchTopK = 10
	contextMaxLen    = 4000 // max chars injected from memory + vector for LLM context
)

// conversationHandler implements MessageHandler: memory read, vector search, LLM call, optional index (REQ-006, REQ-007, REQ-018).
type conversationHandler struct {
	provider         llm.Provider
	memoryStore      *memory.Store // optional; single store, not per-interlocutor
	vectorStore      vector.Store  // optional; for semantic search and indexing
	embedder         embedding.Embedder
	logger           *slog.Logger
	maxMessageLength int
}

// HandleMessage sends the user message to the LLM and returns the assistant reply.
// Reads relevant memory (today's store), runs semantic search, injects context into the LLM call, then indexes the turn (REQ-006, REQ-007, REQ-018).
func (h *conversationHandler) HandleMessage(ctx context.Context, _ int64, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "Please send a non-empty message.", nil
	}
	if h.maxMessageLength > 0 && utf8.RuneCountInString(text) > h.maxMessageLength {
		return fmt.Sprintf("Message is too long. Maximum length is %d characters.", h.maxMessageLength), nil
	}

	contextBlock := h.gatherContext(ctx, text)

	messages := []llm.Message{
		{Role: "system", Content: "You are a helpful assistant. Reply concisely." + contextBlock},
		{Role: "user", Content: text},
	}
	result, err := h.provider.Complete(ctx, messages, nil)
	if err != nil {
		h.logger.Error("llm complete", "error", err)
		return "", err
	}

	if h.vectorStore != nil && h.embedder != nil {
		if err := h.indexTurn(ctx, text, result.Content); err != nil {
			h.logger.Error("index turn", "error", err)
		}
	}

	return result.Content, nil
}

// gatherContext returns a string to inject into the system message: today's memory + semantic search results (REQ-006, REQ-007).
func (h *conversationHandler) gatherContext(ctx context.Context, userText string) string {
	var parts []string
	now := time.Now().UTC()

	if h.memoryStore != nil {
		dayContent, err := h.memoryStore.ReadDay(ctx, now)
		if err != nil {
			h.logger.Error("read memory day", "error", err)
		} else if strings.TrimSpace(dayContent) != "" {
			parts = append(parts, "Relevant memory (today):\n"+dayContent)
		}
	}

	if h.vectorStore != nil && h.embedder != nil {
		queryEmbedding, err := h.embedder.Embed(ctx, userText)
		if err != nil {
			h.logger.Error("embed query", "error", err)
		} else {
			results, err := h.vectorStore.Search(ctx, queryEmbedding, vectorSearchTopK)
			if err != nil {
				h.logger.Error("vector search", "error", err)
			} else if len(results) > 0 {
				var lines []string
				for _, r := range results {
					lines = append(lines, "- "+r.Text)
				}
				parts = append(parts, "Relevant past context:\n"+strings.Join(lines, "\n"))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	s := "\n\n---\n\n" + strings.Join(parts, "\n\n")
	if len(s) > contextMaxLen {
		s = s[:contextMaxLen] + "..."
	}
	return "\n\nUse the following context if relevant to the user's message.\n\n" + s
}

// indexTurn adds the user message and assistant reply to the vector store for future semantic search (REQ-007).
func (h *conversationHandler) indexTurn(ctx context.Context, userText, reply string) error {
	chunk := "User: " + userText + "\nAssistant: " + reply
	emb, err := h.embedder.Embed(ctx, chunk)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return h.vectorStore.Add(ctx, id, emb, chunk)
}
