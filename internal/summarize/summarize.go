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
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Scope is the summarization scope: day (YYYY-MM-DD), month (YYYY-MM), or year (YYYY).
type Scope struct {
	Kind  string    // "day", "month", "year"
	Day   time.Time // set when Kind == "day"
	Year  int       // set for month and year
	Month int       // 1-12 when Kind == "month", 0 for year
}

var (
	reYear  = regexp.MustCompile(`^\d{4}$`)
	reMonth = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	reDay   = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
)

// ParseSummarizeScope parses -summarize value: YYYY (year), YYYY-MM (month), or YYYY-MM-DD (day). No default; value is required.
func ParseSummarizeScope(value string) (Scope, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return Scope{}, fmt.Errorf("summarize: value required (YYYY, YYYY-MM, or YYYY-MM-DD)")
	}
	if reYear.MatchString(s) {
		y, _ := strconv.Atoi(s)
		return Scope{Kind: "year", Year: y}, nil
	}
	if m := reMonth.FindStringSubmatch(s); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		if mo < 1 || mo > 12 {
			return Scope{}, fmt.Errorf("summarize: invalid month in %q", s)
		}
		return Scope{Kind: "month", Year: y, Month: mo}, nil
	}
	if m := reDay.FindStringSubmatch(s); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
		if t.Year() != y || t.Month() != time.Month(mo) || t.Day() != d {
			return Scope{}, fmt.Errorf("summarize: invalid date %q", s)
		}
		return Scope{Kind: "day", Day: t}, nil
	}
	return Scope{}, fmt.Errorf("summarize: invalid format %q (use YYYY, YYYY-MM, or YYYY-MM-DD)", s)
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
