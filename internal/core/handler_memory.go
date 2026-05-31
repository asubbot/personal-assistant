package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/prompt"
	"pa/internal/summarize"
	"pa/internal/vector"
	"strings"
	"time"
)

// handleLLMSuccess logs the LLM call, optionally writes to llmLog, and indexes the turn (REQ-01.018, REQ-01.007).
func (h *conversationHandler) handleLLMSuccess(ctx context.Context, requestID string, messages []llm.Message, result *llm.CompletionResult, userText string, duration time.Duration) {
	if h.llm.llmLog != nil {
		model := h.llm.model
		if result.Model != "" {
			model = result.Model
		}
		h.llm.llmLog.Log(&llmlog.Entry{
			RequestID:       requestID,
			Messages:        messages,
			Model:           model,
			ResponseContent: result.Content,
			Usage:           result.Usage,
			DurationMs:      duration.Milliseconds(),
		})
	}
	if h.llm.logger.Enabled(ctx, slog.LevelDebug) {
		h.logLLMResponse(ctx, result)
	}
	if h.memory.memVec != nil && h.memory.memVec.Turns != nil && h.memory.embedder != nil {
		if err := h.indexTurn(ctx, userText, result.Content); err != nil {
			h.llm.logger.Error("index turn", "error", err)
		}
	}
}

// retrievalChunkWithLabel prepends a type line for the LLM when stored vector text does not already carry it
// (summaries and indexed turns embed `[summary:*]` / `[turn]` after the Date line — avoid duplicating the label).
func retrievalChunkWithLabel(label, body string) string {
	marker := "\n[" + label + "]\n"
	if strings.Contains(body, marker) || strings.HasPrefix(strings.TrimSpace(body), "["+label+"]\n") {
		return body
	}
	return "[" + label + "]\n" + body
}

// gatherRetrievedChunkTexts returns non-empty vector memory chunk texts in search order (REQ-01.006, REQ-01.007).
// The dynamic system tail fitter may drop trailing chunks to satisfy max_dynamic_system_runes.
func (h *conversationHandler) gatherRetrievedChunkTexts(ctx context.Context, userText string) []string {
	mv := h.memory.memVec
	cfgTop := h.memory.memoryVectorTopK
	if mv == nil || !mv.anyNonNil() || h.memory.embedder == nil {
		return nil
	}
	if cfgTop.NotesTopK == 0 && cfgTop.SummariesTopK == 0 && cfgTop.TurnsTopK == 0 {
		return nil
	}
	queryEmbedding, err := h.memory.embedder.Embed(ctx, userText)
	if err != nil {
		h.llm.logger.Error("embed query", "error", err)
		return nil
	}
	return h.gatherSplitTableChunks(ctx, queryEmbedding, cfgTop)
}

func (h *conversationHandler) labeledChunksFromResults(results []vector.SearchResult) []string {
	var chunks []string
	for _, r := range results {
		if strings.TrimSpace(r.Text) == "" {
			continue
		}
		label := summarize.VectorChunkLabel(r.ID)
		chunks = append(chunks, retrievalChunkWithLabel(label, r.Text))
	}
	return chunks
}

func (h *conversationHandler) gatherSplitTableChunks(ctx context.Context, queryEmbedding []float32, topK config.MemoryVectorConfig) []string {
	mv := h.memory.memVec
	var chunks []string
	if mv.Notes != nil && topK.NotesTopK > 0 {
		r, err := mv.Notes.Search(ctx, queryEmbedding, topK.NotesTopK)
		if err != nil {
			h.llm.logger.Error("vector search notes", "error", err)
		} else {
			chunks = append(chunks, h.labeledChunksFromResults(r)...)
		}
	}
	if topK.SummariesTopK > 0 {
		sr, err := mergeSummarySearch(ctx, mv.Summaries, queryEmbedding, topK.SummariesTopK)
		if err != nil {
			h.llm.logger.Error("vector search summaries", "error", err)
			return nil
		}
		chunks = append(chunks, h.labeledChunksFromResults(sr)...)
	}
	if mv.Turns != nil && topK.TurnsTopK > 0 {
		r, err := mv.Turns.Search(ctx, queryEmbedding, topK.TurnsTopK)
		if err != nil {
			h.llm.logger.Error("vector search turns", "error", err)
			return nil
		}
		chunks = append(chunks, h.labeledChunksFromResults(r)...)
	}
	h.llm.logger.DebugContext(ctx, "context chunks", "non_empty", len(chunks))
	return chunks
}

// indexTurn adds the user message and assistant reply to the turn vector store (REQ-01.007, EP-016 dedup).
func (h *conversationHandler) indexTurn(ctx context.Context, userText, reply string) error {
	if h.memory.memVec == nil || h.memory.memVec.Turns == nil || h.memory.embedder == nil {
		return nil
	}
	loc := time.UTC
	if h.memory.paLoc != nil {
		loc = h.memory.paLoc
	}
	dateStr := eventAlignedTurnDate(ctx, loc)
	chunk := "Date: " + dateStr + "\n[turn]\nUser: " + userText + "\nAssistant: " + reply
	if prompt.TextContainsForbiddenMarkerLine(chunk) {
		return fmt.Errorf("indexTurn: chunk contains forbidden PA marker line")
	}
	emb, err := h.memory.embedder.Embed(ctx, chunk)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(canonicalizeTurnPair(userText, reply)))
	// First 12 bytes of the digest (24 hex chars) keep ids short; collision risk for same-day dedup is negligible.
	id := fmt.Sprintf("turn:%s:%x", dateStr, sum[:12])
	_ = h.memory.memVec.Turns.Delete(ctx, id)
	return h.memory.memVec.Turns.Add(ctx, id, emb, chunk)
}

func eventAlignedTurnDate(ctx context.Context, paLoc *time.Location) string {
	if u := TelegramMessageDateUnix(ctx); u > 0 {
		return time.Unix(u, 0).In(paLoc).Format("2006-01-02")
	}
	return time.Now().In(paLoc).Format("2006-01-02")
}

func canonicalizeTurnPair(userText, reply string) string {
	u := canonicalizeTurnText(userText)
	a := canonicalizeTurnText(reply)
	return u + "\n---\n" + a
}

func canonicalizeTurnText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Join(strings.Fields(s), " ")
}
