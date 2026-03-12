package summarize

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/memory"
	"pa/internal/vector"
	"strings"
	"time"
)

// ParseDayDate returns the day to summarize (midnight UTC). If dateFlag is non-empty, parses YYYY-MM-DD; otherwise returns yesterday in paTimezone (or UTC if empty).
func ParseDayDate(dateFlag, paTimezone string) (time.Time, error) {
	if strings.TrimSpace(dateFlag) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(dateFlag))
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	loc := time.UTC
	if strings.TrimSpace(paTimezone) != "" {
		var err error
		loc, err = time.LoadLocation(paTimezone)
		if err != nil {
			return time.Time{}, err
		}
	}
	now := time.Now().In(loc)
	yesterday := now.AddDate(0, 0, -1)
	return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC), nil
}

const (
	summarizeSystemPrompt = "You are summarizing a day's conversation with an assistant. Produce a concise summary in a few sentences or bullet points. Write in the same language as the conversation. Output only the summary, no preamble."
	summarizeDayIDPrefix  = "summary:day:"
)

// DayConfig holds dependencies for running day summarization.
type DayConfig struct {
	LLMLogDir   string
	LLMProvider llm.Provider
	MemoryStore *memory.Store
	Embedder    embedding.Embedder
	VectorStore vector.Store
	Logger      *slog.Logger
}

// Day runs summarization for the given calendar day (UTC): reads LLM logs, builds transcript,
// calls LLM to summarize, writes summary to memory (YYYY/MM/DD/summary.md) and adds it to the vector store.
// If there are no log entries for the day, no write is performed and nil is returned (skip empty day).
func Day(ctx context.Context, day time.Time, cfg DayConfig) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	entries, err := llmlog.ReadEntriesForDay(cfg.LLMLogDir, day)
	if err != nil {
		return fmt.Errorf("summarize day: read logs: %w", err)
	}
	if len(entries) == 0 {
		cfg.Logger.Info("summarize day: no entries, skipping", "day", day.UTC().Format("2006-01-02"))
		return nil
	}

	transcript := buildDayTranscript(entries)
	messages := []llm.Message{
		{Role: "system", Content: summarizeSystemPrompt},
		{Role: "user", Content: "Summarize the following day's conversation:\n\n" + transcript},
	}

	result, err := cfg.LLMProvider.Complete(ctx, messages, nil)
	if err != nil {
		return fmt.Errorf("summarize day: llm: %w", err)
	}
	summaryText := strings.TrimSpace(result.Content)
	if summaryText == "" {
		return fmt.Errorf("summarize day: llm returned empty summary")
	}

	if err := cfg.MemoryStore.WriteDaySummary(ctx, day, summaryText); err != nil {
		return fmt.Errorf("summarize day: write memory: %w", err)
	}

	id := summarizeDayIDPrefix + day.UTC().Format("2006-01-02")
	if cfg.VectorStore != nil && cfg.Embedder != nil {
		_ = cfg.VectorStore.Delete(ctx, id)
		emb, err := cfg.Embedder.Embed(ctx, summaryText)
		if err != nil {
			return fmt.Errorf("summarize day: embed: %w", err)
		}
		if err := cfg.VectorStore.Add(ctx, id, emb, summaryText); err != nil {
			return fmt.Errorf("summarize day: vector add: %w", err)
		}
	}

	cfg.Logger.Info("summarize day: done", "day", day.UTC().Format("2006-01-02"), "entries", len(entries))
	return nil
}

// buildDayTranscript concatenates log entries into a single transcript (user/assistant turns).
func buildDayTranscript(entries []llmlog.Entry) string {
	var b strings.Builder
	for _, e := range entries {
		for _, m := range e.Messages {
			b.WriteString(m.Role)
			b.WriteString(": ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
		if e.ResponseContent != "" {
			b.WriteString("Assistant: ")
			b.WriteString(e.ResponseContent)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
