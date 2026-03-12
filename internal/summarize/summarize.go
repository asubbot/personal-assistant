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
	summarizeSystemPrompt      = "You are summarizing a day's conversation with an assistant. Produce a concise summary in a few sentences or bullet points. Write in the same language as the conversation. Output only the summary, no preamble."
	summarizeMonthSystemPrompt = "You are summarizing a month's daily summaries into a single monthly overview. Produce a concise summary in a few sentences or bullet points. Use the same language as the content. Output only the summary, no preamble."
	summarizeYearSystemPrompt  = "You are summarizing a year's monthly summaries into a single yearly overview. Produce a concise summary in a few sentences or bullet points. Use the same language as the content. Output only the summary, no preamble."
	summarizeDayIDPrefix       = "summary:day:"
	summarizeMonthIDPrefix     = "summary:month:"
	summarizeYearIDPrefix      = "summary:year:"
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

// MonthConfig holds dependencies for running month summarization (reads day summaries from memory).
type MonthConfig struct {
	LLMProvider llm.Provider
	MemoryStore *memory.Store
	Embedder    embedding.Embedder
	VectorStore vector.Store
	Logger      *slog.Logger
}

// YearConfig holds dependencies for running year summarization (reads month summaries from memory).
type YearConfig struct {
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

// Month runs summarization for the given month (UTC): reads day summaries from memory,
// calls LLM to produce a monthly overview, writes to memory (YYYY/MM/summary.md) and adds to the vector store.
// If no day summaries exist for the month, no write is performed and nil is returned.
func Month(ctx context.Context, year int, month int, cfg MonthConfig) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	sections, err := gatherDaySummariesForMonth(ctx, cfg.MemoryStore, year, month)
	if err != nil {
		return err
	}
	if len(sections) == 0 {
		cfg.Logger.Info("summarize month: no day summaries, skipping", "year", year, "month", month)
		return nil
	}

	combined := strings.Join(sections, "\n\n")
	summaryText, err := completeRollupSummary(ctx, cfg.LLMProvider, summarizeMonthSystemPrompt, "Summarize the following month's daily summaries:\n\n"+combined)
	if err != nil {
		return fmt.Errorf("summarize month: %w", err)
	}

	if err := cfg.MemoryStore.WriteMonthSummary(ctx, year, month, summaryText); err != nil {
		return fmt.Errorf("summarize month: write memory: %w", err)
	}

	id := summarizeMonthIDPrefix + fmt.Sprintf("%04d-%02d", year, month)
	if err := indexSummary(ctx, cfg.VectorStore, cfg.Embedder, id, summaryText); err != nil {
		return fmt.Errorf("summarize month: %w", err)
	}

	cfg.Logger.Info("summarize month: done", "year", year, "month", month, "day_summaries", len(sections))
	return nil
}

// Year runs summarization for the given year: reads month summaries from memory,
// calls LLM to produce a yearly overview, writes to memory (YYYY/summary.md) and adds to the vector store.
// If no month summaries exist for the year, no write is performed and nil is returned.
func Year(ctx context.Context, year int, cfg YearConfig) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	sections, err := gatherMonthSummariesForYear(ctx, cfg.MemoryStore, year)
	if err != nil {
		return err
	}
	if len(sections) == 0 {
		cfg.Logger.Info("summarize year: no month summaries, skipping", "year", year)
		return nil
	}

	combined := strings.Join(sections, "\n\n")
	summaryText, err := completeRollupSummary(ctx, cfg.LLMProvider, summarizeYearSystemPrompt, "Summarize the following year's monthly summaries:\n\n"+combined)
	if err != nil {
		return fmt.Errorf("summarize year: %w", err)
	}

	if err := cfg.MemoryStore.WriteYearSummary(ctx, year, summaryText); err != nil {
		return fmt.Errorf("summarize year: write memory: %w", err)
	}

	id := summarizeYearIDPrefix + fmt.Sprintf("%04d", year)
	if err := indexSummary(ctx, cfg.VectorStore, cfg.Embedder, id, summaryText); err != nil {
		return fmt.Errorf("summarize year: %w", err)
	}

	cfg.Logger.Info("summarize year: done", "year", year, "month_summaries", len(sections))
	return nil
}

// gatherDaySummariesForMonth reads day summaries for the given month and returns section blocks (## YYYY-MM-DD\ncontent). Missing days are skipped.
func gatherDaySummariesForMonth(ctx context.Context, store *memory.Store, year int, month int) ([]string, error) {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := first.AddDate(0, 1, -1).Day()
	var sections []string
	for d := 1; d <= lastDay; d++ {
		day := time.Date(year, time.Month(month), d, 0, 0, 0, 0, time.UTC)
		content, err := store.ReadDaySummary(ctx, day)
		if err != nil {
			return nil, fmt.Errorf("read day summary %s: %w", day.Format("2006-01-02"), err)
		}
		if content == "" {
			continue
		}
		sections = append(sections, "## "+day.Format("2006-01-02")+"\n"+content)
	}
	return sections, nil
}

// gatherMonthSummariesForYear reads month summaries for the given year and returns section blocks (## YYYY-MM\ncontent). Missing months are skipped.
func gatherMonthSummariesForYear(ctx context.Context, store *memory.Store, year int) ([]string, error) {
	var sections []string
	for m := 1; m <= 12; m++ {
		content, err := store.ReadMonthSummary(ctx, year, m)
		if err != nil {
			return nil, fmt.Errorf("read month summary %04d-%02d: %w", year, m, err)
		}
		if content == "" {
			continue
		}
		sections = append(sections, "## "+fmt.Sprintf("%04d-%02d", year, m)+"\n"+content)
	}
	return sections, nil
}

// completeRollupSummary calls the LLM with the given system prompt and user content, returns trimmed summary or error.
func completeRollupSummary(ctx context.Context, provider llm.Provider, systemPrompt, userContent string) (string, error) {
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}
	result, err := provider.Complete(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}
	summary := strings.TrimSpace(result.Content)
	if summary == "" {
		return "", fmt.Errorf("llm returned empty summary")
	}
	return summary, nil
}

// indexSummary deletes any existing vector for id, embeds text and adds to the store. No-op if store or embedder is nil.
func indexSummary(ctx context.Context, store vector.Store, embedder embedding.Embedder, id, text string) error {
	if store == nil || embedder == nil {
		return nil
	}
	_ = store.Delete(ctx, id)
	emb, err := embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	if err := store.Add(ctx, id, emb, text); err != nil {
		return fmt.Errorf("vector add: %w", err)
	}
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
