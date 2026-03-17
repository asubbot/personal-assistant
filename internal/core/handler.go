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
	defaultContextMaxLen    = 4000 // used when config conversation_context.injected_context_max_chars is 0 or unset
	defaultVectorSearchTopK = 10   // used when config conversation_context.vector_search_top_k is 0 or unset
	logTruncateMaxLen       = 2000 // max chars per message/response when logging at DEBUG (REQ-01.021)
)

// genRequestID returns a short unique id for LLM log entries (16 hex chars from 8 random bytes).
func genRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// conversationHandler implements MessageHandler: vector search, LLM call, optional index (REQ-01.006, REQ-01.007, REQ-01.018).
// Context is built only from vector store (turns and summaries); no full.md day file.
type conversationHandler struct {
	provider         llm.Provider
	vectorStore      vector.Store // optional; for semantic search and indexing
	embedder         embedding.Embedder
	nodeRunner       NodeRunner // optional; for tools that run allowlisted commands on nodes (REQ-01.004, REQ-01.005, REQ-01.013)
	logger           *slog.Logger
	maxMessageLength int
	contextMaxLen    int                 // max chars for injected context block; 0 = defaultContextMaxLen
	vectorSearchTopK int                 // number of vector search results; 0 = defaultVectorSearchTopK
	llmLog           llmlog.Writer       // optional; when set, each LLM call is logged as JSONL
	model            string              // configured model name for LLM log entries
	logRedactor      func(string) string // optional; redacts content in DEBUG app logs (REQ-01.026)
}

// HandleMessage sends the user message to the LLM and returns the assistant reply.
// Runs semantic search, injects context into the LLM call, then indexes the turn (REQ-01.006, REQ-01.007, REQ-01.018).
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
		systemContent = "You are a personal assistant. Reply concisely." + contextBlock
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
	if err != nil {
		h.logger.Error("llm complete", "error", err)
		return "", err
	}
	h.handleLLMSuccess(ctx, requestID, messages, result, text, time.Since(start))
	return result.Content, nil
}

// handleLLMSuccess logs the LLM call, optionally writes to llmLog, and indexes the turn (REQ-01.018, REQ-01.007).
func (h *conversationHandler) handleLLMSuccess(ctx context.Context, requestID string, messages []llm.Message, result *llm.CompletionResult, userText string, duration time.Duration) {
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
		if err := h.indexTurn(ctx, userText, result.Content); err != nil {
			h.logger.Error("index turn", "error", err)
		}
	}
}

// gatherContext returns a string to inject into the system message: semantic search results from vector store (REQ-01.006, REQ-01.007).
// Only whole chunks are included; when the limit is reached, remaining chunks are dropped (no mid-chunk truncation).
func (h *conversationHandler) gatherContext(ctx context.Context, userText string) string {
	topK := h.vectorSearchTopK
	if topK <= 0 {
		topK = defaultVectorSearchTopK
	}
	if h.vectorStore == nil || h.embedder == nil {
		return ""
	}
	queryEmbedding, err := h.embedder.Embed(ctx, userText)
	if err != nil {
		h.logger.Error("embed query", "error", err)
		return ""
	}
	results, err := h.vectorStore.Search(ctx, queryEmbedding, topK)
	if err != nil {
		h.logger.Error("vector search", "error", err)
		return ""
	}
	if len(results) == 0 {
		return ""
	}

	maxLen := h.contextMaxLen
	if maxLen <= 0 {
		maxLen = defaultContextMaxLen
	}
	const suffixReserve = 4 // for trailing "\n..." when not all chunks fit
	prefix := "\n\n---\n\nRelevant past context:\n"
	buf := prefix
	fitted := 0
	for _, r := range results {
		line := "- " + r.Text + "\n"
		if len(buf)+len(line)+suffixReserve <= maxLen {
			buf += line
			fitted++
		} else {
			break
		}
	}
	if fitted == 0 {
		h.logger.DebugContext(ctx, "context chunks", "fitted", 0, "total", len(results))
		return ""
	}
	if fitted < len(results) {
		buf += "\n..."
	}
	h.logger.DebugContext(ctx, "context chunks", "fitted", fitted, "total", len(results))
	return "\n\nUse the following context if relevant to the user's message.\n\n" + buf
}

// logLLMRequest logs the full request at DEBUG (REQ-01.021). Content may be truncated and redacted (REQ-01.026).
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

// logLLMMetadata logs message count, response length, and usage at INFO (REQ-01.021).
func (h *conversationHandler) logLLMMetadata(ctx context.Context, messageCount int, result *llm.CompletionResult) {
	h.logger.InfoContext(ctx, "llm call", "message_count", messageCount, "response_len", len(result.Content), "prompt_tokens", result.Usage.PromptTokens, "completion_tokens", result.Usage.CompletionTokens, "total_tokens", result.Usage.TotalTokens)
}

// logLLMResponse logs the full response at DEBUG (REQ-01.021). Content may be truncated and redacted (REQ-01.026).
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

// indexTurn adds the user message and assistant reply to the vector store for future semantic search (REQ-01.007).
func (h *conversationHandler) indexTurn(ctx context.Context, userText, reply string) error {
	chunk := "User: " + userText + "\nAssistant: " + reply
	emb, err := h.embedder.Embed(ctx, chunk)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return h.vectorStore.Add(ctx, id, emb, chunk)
}
