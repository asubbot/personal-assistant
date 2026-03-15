package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/vector"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	vectorSearchTopK  = 10
	contextMaxLen     = 4000 // max chars injected from memory + vector for LLM context
	logTruncateMaxLen = 2000 // max chars per message/response when logging at DEBUG (REQ-021)
)

// genRequestID returns a short unique id for LLM log entries (16 hex chars from 8 random bytes).
func genRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// conversationHandler implements MessageHandler: vector search, LLM call, optional index (REQ-006, REQ-007, REQ-018).
// Context is built only from vector store (turns and summaries); no full.md day file.
type conversationHandler struct {
	provider         llm.Provider
	vectorStore      vector.Store // optional; for semantic search and indexing
	embedder         embedding.Embedder
	nodeRunner       NodeRunner // optional; for tools that run allowlisted commands on nodes (REQ-004, REQ-005, REQ-013)
	logger           *slog.Logger
	maxMessageLength int
	llmLog           llmlog.Writer       // optional; when set, each LLM call is logged as JSONL
	model            string              // configured model name for LLM log entries
	logRedactor      func(string) string // optional; redacts content in DEBUG app logs (REQ-026)
}

// HandleMessage sends the user message to the LLM and returns the assistant reply.
// Runs semantic search, injects context into the LLM call, then indexes the turn (REQ-006, REQ-007, REQ-018).
func (h *conversationHandler) HandleMessage(ctx context.Context, _ int64, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "Please send a non-empty message.", nil
	}
	if h.maxMessageLength > 0 && utf8.RuneCountInString(text) > h.maxMessageLength {
		return fmt.Sprintf("Message is too long. Maximum length is %d characters.", h.maxMessageLength), nil
	}

	contextBlock := h.gatherContext(ctx, text)
	systemContent := "You are a helpful assistant. Reply concisely."
	if contextBlock != "" {
		systemContent = "You are a helpful assistant. Reply concisely. The following is your memory of past conversations; use it to personalize replies and to remember what the user has told you. Do not say you cannot remember when the information is provided below." + contextBlock
	}
	messages := []llm.Message{
		{Role: "system", Content: systemContent},
		{Role: "user", Content: text},
	}
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		h.logLLMRequest(ctx, messages)
	}
	requestID := genRequestID()
	start := time.Now()
	result, err := h.provider.Complete(ctx, messages, nil)
	duration := time.Since(start)
	if err != nil {
		h.logger.Error("llm complete", "error", err)
		return "", err
	}
	if h.llmLog != nil {
		model := h.model
		if result.Model != "" {
			model = result.Model
		}
		h.llmLog.Log(&llmlog.Entry{
			RequestID:       requestID,
			Messages:        messages,
			Model:           model,
			ResponseContent: result.Content,
			Usage:           result.Usage,
			DurationMs:      duration.Milliseconds(),
		})
	}
	h.logLLMMetadata(ctx, len(messages), result)
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		h.logLLMResponse(ctx, result)
	}

	if h.vectorStore != nil && h.embedder != nil {
		if err := h.indexTurn(ctx, text, result.Content); err != nil {
			h.logger.Error("index turn", "error", err)
		}
	}

	return result.Content, nil
}

// gatherContext returns a string to inject into the system message: semantic search results from vector store (REQ-006, REQ-007).
func (h *conversationHandler) gatherContext(ctx context.Context, userText string) string {
	var parts []string

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

// logLLMRequest logs the full request at DEBUG (REQ-021). Content may be truncated and redacted (REQ-026).
func (h *conversationHandler) logLLMRequest(ctx context.Context, messages []llm.Message) {
	for i, m := range messages {
		content := m.Content
		if len(content) > logTruncateMaxLen {
			content = content[:logTruncateMaxLen] + "...[truncated]"
		}
		if h.logRedactor != nil {
			content = h.logRedactor(content)
		}
		h.logger.DebugContext(ctx, "llm request", "index", i, "role", m.Role, "content_len", len(m.Content), "content", content)
	}
}

// logLLMMetadata logs message count, response length, and usage at INFO (REQ-021).
func (h *conversationHandler) logLLMMetadata(ctx context.Context, messageCount int, result *llm.CompletionResult) {
	h.logger.InfoContext(ctx, "llm call", "message_count", messageCount, "response_len", len(result.Content), "prompt_tokens", result.Usage.PromptTokens, "completion_tokens", result.Usage.CompletionTokens, "total_tokens", result.Usage.TotalTokens)
}

// logLLMResponse logs the full response at DEBUG (REQ-021). Content may be truncated and redacted (REQ-026).
func (h *conversationHandler) logLLMResponse(ctx context.Context, result *llm.CompletionResult) {
	content := result.Content
	if len(content) > logTruncateMaxLen {
		content = content[:logTruncateMaxLen] + "...[truncated]"
	}
	if h.logRedactor != nil {
		content = h.logRedactor(content)
	}
	h.logger.DebugContext(ctx, "llm response", "content", content, "content_len", len(result.Content), "usage", result.Usage)
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
