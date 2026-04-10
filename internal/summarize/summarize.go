package summarize

import (
	"context"
	"encoding/json"
	"errors"
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

// llmMessagesDebugText formats messages for debug logs (same text bodies passed to the LLM as in Complete).
func llmMessagesDebugText(messages []llm.Message) string {
	var b strings.Builder
	for i, m := range messages {
		if i > 0 {
			b.WriteString("\n\n--- message ---\n\n")
		}
		b.WriteString("[")
		b.WriteString(m.Role)
		b.WriteString("]\n")
		b.WriteString(m.Content)
	}
	return b.String()
}

// llmMessagesJSONByteLen returns the UTF-8 byte length of JSON-encoding messages (the chat messages array).
// The provider HTTP body also includes model, max_tokens, temperature, etc.; this value is the messages payload only.
func llmMessagesJSONByteLen(messages []llm.Message) int {
	b, err := json.Marshal(messages)
	if err != nil {
		return 0
	}
	return len(b)
}

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
	// summarizeDayTranscriptLegend explains how buildDayTranscript output is shaped so the model summarizes substance, not log mechanics.
	summarizeDayTranscriptLegend = "Transcript: chronological lines with prefixes user:, assistant:, or Assistant: (final reply when stored separately). Omitted: system, tool payloads, and empty assistant turns—URLs appear only if present in the shown user/assistant text. Merge repeated user questions into themes; summarize what happened, not the log format."

	summarizeMonthInputLegend = "Input: sections headed ## YYYY-MM-DD per day; days without a saved summary are omitted."

	summarizeYearInputLegend = "Input: sections headed ## YYYY-MM per month; months without a saved summary are omitted."

	// rollupOutputBlock is shared by month and year system prompts (dominant language, structure, links, no tables).
	rollupOutputBlock = "Use the dominant language of the excerpts below (do not default to English when they are mostly another language).\n\nMarkdown: 1–2 overview paragraphs; a line ## plus a title in that language for main themes/facts; then 3–8 lines starting \"- \". Keep important [label](url) links only when the URL appears verbatim in the excerpts; never invent URLs. No tables. No preamble.\n\nYour entire reply must be ONLY that markdown. Do not output planning, numbered steps, headings like \"Analyze\", \"Draft\", \"Review\", scratchpads, self-checklists, or any chain-of-thought. Do not repeat these instructions. Start with the first overview paragraph.\n\n"

	// summarizeCompletionMaxTokens caps LLM completion length for day summaries and month/year rollups (provider default_max_tokens is often too low and truncates mid-sentence).
	summarizeCompletionMaxTokens = 8192

	summarizeSystemPrompt = "Summarize this day's assistant conversation. Write in the dominant language of user: lines in the log (if mixed, pick the dominant one; do not default to English).\n\nMarkdown: 1–2 overview paragraphs; a line ## plus a title in that language meaning main/key facts; then 3–8 lines starting \"- \" with durable facts (decisions, tool/integration status, preferences). Use [label](url) only for URLs copied verbatim from the transcript; never invent links. No tables. Omit the ## section only if there are no facts beyond the overview. No preamble.\n\nYour entire reply must be ONLY that markdown. Do not output planning, numbered steps, headings like \"Analyze the request\", \"Draft\", \"Review\", scratchpads, self-checklists, or chain-of-thought in any language. Do not repeat these instructions. Start with the first overview paragraph.\n\n" + summarizeDayTranscriptLegend

	summarizeMonthSystemPrompt = "Roll up the month's daily summaries into one overview.\n\n" + rollupOutputBlock + summarizeMonthInputLegend

	summarizeYearSystemPrompt = "Roll up the year's monthly summaries into one overview.\n\n" + rollupOutputBlock + summarizeYearInputLegend
)

// DayConfig holds dependencies for running day summarization.
type DayConfig struct {
	LLMLogDir   string
	LLMProvider llm.Provider
	MemoryStore *memory.Store
	Embedder    embedding.Embedder
	VectorStore vector.Store
	Logger      *slog.Logger
	// Loc is pa_timezone for log file selection and calendar ids; nil means UTC.
	Loc *time.Location
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

// logLLMPayloadDebug logs the full messages text when debug logging is enabled (PA_LOG_LEVEL=debug).
func errEmptyLLMResult(result *llm.CompletionResult, label string) error {
	hint := fmt.Sprintf("completion_tokens=%d prompt_tokens=%d", result.Usage.CompletionTokens, result.Usage.PromptTokens)
	if len(result.ToolCalls) > 0 {
		hint += fmt.Sprintf("; model returned %d tool_calls with no assistant text", len(result.ToolCalls))
	}
	return fmt.Errorf("%s: llm returned empty summary (%s)", label, hint)
}

func logLLMPayloadDebug(ctx context.Context, log *slog.Logger, msg string, messages []llm.Message, attrs ...any) {
	if log == nil || !log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	args := make([]any, 0, len(attrs)+2)
	args = append(args, attrs...)
	args = append(args, "llm_messages_text", llmMessagesDebugText(messages))
	log.Debug(msg, args...)
}

func completeDaySummaryLLM(ctx context.Context, log *slog.Logger, dateStr string, p llm.Provider, transcript string) (string, error) {
	messages := []llm.Message{
		{Role: "system", Content: summarizeSystemPrompt},
		{Role: "user", Content: "Summarize the following day's conversation:\n\n" + transcript},
	}
	log.Info("summarize day: calling llm", "day", dateStr,
		"llm_request_messages_json_bytes", llmMessagesJSONByteLen(messages))
	logLLMPayloadDebug(ctx, log, "summarize day: llm messages payload", messages, "day", dateStr)
	result, err := p.Complete(ctx, messages, &llm.CompletionOptions{MaxTokens: summarizeCompletionMaxTokens})
	if err != nil {
		return "", fmt.Errorf("summarize day: llm: %w", err)
	}
	summaryText := strings.TrimSpace(result.Content)
	if summaryText == "" {
		return "", errEmptyLLMResult(result, "summarize day")
	}
	log.Info("summarize day: llm returned summary", "day", dateStr, "summary_bytes", len(summaryText))
	return summaryText, nil
}

// Day runs summarization for the given calendar day in cfg.Loc (pa_timezone): reads LLM logs, builds transcript,
// calls LLM to summarize, writes summary to memory (YYYY/MM/DD/summary.md) and adds it to the vector store.
// If there are no log entries for the day, no write is performed and nil is returned (skip empty day).
func Day(ctx context.Context, day time.Time, cfg DayConfig) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	loc := cfg.Loc
	if loc == nil {
		loc = time.UTC
	}
	dateStr := day.In(loc).Format("2006-01-02")
	cfg.Logger.Info("summarize day: reading llm log", "day", dateStr, "dir", cfg.LLMLogDir)

	entries, err := llmlog.ReadEntriesForDay(cfg.LLMLogDir, day, loc)
	if err != nil {
		return fmt.Errorf("summarize day: read logs: %w", err)
	}
	if len(entries) == 0 {
		cfg.Logger.Info("summarize day: no entries, skipping", "day", dateStr)
		return nil
	}
	cfg.Logger.Info("summarize day: loaded log entries", "day", dateStr, "entries", len(entries))

	transcript := buildDayTranscript(entries)
	cfg.Logger.Info("summarize day: built transcript", "day", dateStr, "transcript_bytes", len(transcript))
	summaryText, err := completeDaySummaryLLM(ctx, cfg.Logger, dateStr, cfg.LLMProvider, transcript)
	if err != nil {
		return err
	}

	cfg.Logger.Info("summarize day: writing memory", "day", dateStr)
	if err := cfg.MemoryStore.WriteDaySummary(ctx, day, summaryText); err != nil {
		return fmt.Errorf("summarize day: write memory: %w", err)
	}

	id := VectorIDPrefixDay + dateStr
	if cfg.VectorStore != nil && cfg.Embedder != nil {
		vecText := FormatDayVectorText(dateStr, summaryText)
		cfg.Logger.Info("summarize day: embedding and indexing vector", "day", dateStr, "id", id)
		_ = cfg.VectorStore.Delete(ctx, id)
		emb, err := cfg.Embedder.Embed(ctx, vecText)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrVectorIndexAfterFileWrite, err)
		}
		if err := cfg.VectorStore.Add(ctx, id, emb, vecText); err != nil {
			return fmt.Errorf("%w: %w", ErrVectorIndexAfterFileWrite, err)
		}
		cfg.Logger.Info("summarize day: vector index updated", "day", dateStr, "id", id)
	}

	cfg.Logger.Info("summarize day: done", "day", dateStr, "entries", len(entries))
	return nil
}

// Month runs summarization for the given month (UTC): reads day summaries from memory,
// calls LLM to produce a monthly overview, writes to memory (YYYY/MM/summary.md) and adds to the vector store.
// If no day summaries exist for the month, no write is performed and nil is returned.
func Month(ctx context.Context, year int, month int, cfg MonthConfig) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	cfg.Logger.Info("summarize month: gathering day summaries", "year", year, "month", month)
	sections, err := GatherDaySummariesForMonth(ctx, cfg.MemoryStore, year, month)
	if err != nil {
		return err
	}
	if len(sections) == 0 {
		cfg.Logger.Info("summarize month: no day summaries, skipping", "year", year, "month", month)
		return nil
	}
	cfg.Logger.Info("summarize month: loaded day summary sections", "year", year, "month", month, "sections", len(sections))

	combined := strings.Join(sections, "\n\n")
	monthRollupMessages := []llm.Message{
		{Role: "system", Content: summarizeMonthSystemPrompt},
		{Role: "user", Content: "Summarize the following month's daily summaries:\n\n" + combined},
	}
	cfg.Logger.Info("summarize month: calling llm", "year", year, "month", month,
		"rollup_bytes", len(combined),
		"llm_request_messages_json_bytes", llmMessagesJSONByteLen(monthRollupMessages))
	logLLMPayloadDebug(ctx, cfg.Logger, "summarize month: llm messages payload", monthRollupMessages, "year", year, "month", month)
	summaryText, err := completeFromLLMMessages(ctx, cfg.LLMProvider, monthRollupMessages)
	if err != nil {
		return fmt.Errorf("summarize month: %w", err)
	}
	cfg.Logger.Info("summarize month: llm returned summary", "year", year, "month", month, "summary_bytes", len(summaryText))

	cfg.Logger.Info("summarize month: writing memory", "year", year, "month", month)
	if err := cfg.MemoryStore.WriteMonthSummary(ctx, year, month, summaryText); err != nil {
		return fmt.Errorf("summarize month: write memory: %w", err)
	}

	if cfg.VectorStore != nil && cfg.Embedder != nil {
		id := VectorIDPrefixMonth + fmt.Sprintf("%04d-%02d", year, month)
		vecText := FormatMonthVectorText(year, month, summaryText)
		cfg.Logger.Info("summarize month: embedding and indexing vector", "year", year, "month", month, "id", id)
		if err := indexSummary(ctx, cfg.VectorStore, cfg.Embedder, id, vecText); err != nil {
			return fmt.Errorf("summarize month: %w: %w", ErrVectorIndexAfterFileWrite, err)
		}
		cfg.Logger.Info("summarize month: vector index updated", "year", year, "month", month, "id", id)
	} else {
		cfg.Logger.Info("summarize month: vector indexing skipped", "year", year, "month", month, "reason", "no vector store or embedder")
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

	cfg.Logger.Info("summarize year: gathering month summaries", "year", year)
	sections, err := gatherMonthSummariesForYear(ctx, cfg.MemoryStore, year)
	if err != nil {
		return err
	}
	if len(sections) == 0 {
		cfg.Logger.Info("summarize year: no month summaries, skipping", "year", year)
		return nil
	}
	cfg.Logger.Info("summarize year: loaded month summary sections", "year", year, "sections", len(sections))

	combined := strings.Join(sections, "\n\n")
	yearRollupMessages := []llm.Message{
		{Role: "system", Content: summarizeYearSystemPrompt},
		{Role: "user", Content: "Summarize the following year's monthly summaries:\n\n" + combined},
	}
	cfg.Logger.Info("summarize year: calling llm", "year", year,
		"rollup_bytes", len(combined),
		"llm_request_messages_json_bytes", llmMessagesJSONByteLen(yearRollupMessages))
	logLLMPayloadDebug(ctx, cfg.Logger, "summarize year: llm messages payload", yearRollupMessages, "year", year)
	summaryText, err := completeFromLLMMessages(ctx, cfg.LLMProvider, yearRollupMessages)
	if err != nil {
		return fmt.Errorf("summarize year: %w", err)
	}
	cfg.Logger.Info("summarize year: llm returned summary", "year", year, "summary_bytes", len(summaryText))

	cfg.Logger.Info("summarize year: writing memory", "year", year)
	if err := cfg.MemoryStore.WriteYearSummary(ctx, year, summaryText); err != nil {
		return fmt.Errorf("summarize year: write memory: %w", err)
	}

	if cfg.VectorStore != nil && cfg.Embedder != nil {
		id := VectorIDPrefixYear + fmt.Sprintf("%04d", year)
		vecText := FormatYearVectorText(year, summaryText)
		cfg.Logger.Info("summarize year: embedding and indexing vector", "year", year, "id", id)
		if err := indexSummary(ctx, cfg.VectorStore, cfg.Embedder, id, vecText); err != nil {
			return fmt.Errorf("summarize year: %w: %w", ErrVectorIndexAfterFileWrite, err)
		}
		cfg.Logger.Info("summarize year: vector index updated", "year", year, "id", id)
	} else {
		cfg.Logger.Info("summarize year: vector indexing skipped", "year", year, "reason", "no vector store or embedder")
	}

	cfg.Logger.Info("summarize year: done", "year", year, "month_summaries", len(sections))
	return nil
}

// GatherDaySummariesForMonth reads day summaries for the given month and returns section blocks (## YYYY-MM-DD\ncontent). Missing days are skipped (EP-002).
func GatherDaySummariesForMonth(ctx context.Context, store *memory.Store, year int, month int) ([]string, error) {
	loc := store.Location()
	first := time.Date(year, time.Month(month), 1, 12, 0, 0, 0, loc)
	lastDay := first.AddDate(0, 1, -1).Day()
	var sections []string
	for d := 1; d <= lastDay; d++ {
		day := time.Date(year, time.Month(month), d, 12, 0, 0, 0, loc)
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

// completeFromLLMMessages calls the LLM with the given messages and returns trimmed assistant text or error.
func completeFromLLMMessages(ctx context.Context, provider llm.Provider, messages []llm.Message) (string, error) {
	result, err := provider.Complete(ctx, messages, &llm.CompletionOptions{MaxTokens: summarizeCompletionMaxTokens})
	if err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}
	summary := strings.TrimSpace(result.Content)
	if summary == "" {
		return "", errEmptyLLMResult(result, "summarize")
	}
	return summary, nil
}

// indexSummary deletes any existing vector for id, embeds vectorText and adds to the store. No-op if store or embedder is nil.
func indexSummary(ctx context.Context, store vector.Store, embedder embedding.Embedder, id, vectorText string) error {
	if store == nil || embedder == nil {
		return nil
	}
	_ = store.Delete(ctx, id)
	emb, err := embedder.Embed(ctx, vectorText)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	if err := store.Add(ctx, id, emb, vectorText); err != nil {
		return fmt.Errorf("vector add: %w", err)
	}
	return nil
}

// IsVectorIndexAfterFileWrite reports whether err is ErrVectorIndexAfterFileWrite (or wraps it).
func IsVectorIndexAfterFileWrite(err error) bool {
	return err != nil && errors.Is(err, ErrVectorIndexAfterFileWrite)
}

// transcriptMessageRole reports whether a message role is included in the day summarization transcript.
// Only operator (user) and in-thread assistant text are kept; system prompts, tool results, and other roles are omitted.
func transcriptMessageRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user", "assistant":
		return true
	default:
		return false
	}
}

// buildDayTranscript builds a day transcript for summarization: user and assistant messages only (no system,
// no tool), no assistant lines with empty body, no duplicate final assistant line when response_content matches the last assistant message in Messages.
func buildDayTranscript(entries []llmlog.Entry) string {
	var b strings.Builder
	for _, e := range entries {
		var lastRole, lastContent string
		for _, m := range e.Messages {
			if !transcriptMessageRole(m.Role) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(m.Role), "assistant") && strings.TrimSpace(m.Content) == "" {
				continue
			}
			b.WriteString(m.Role)
			b.WriteString(": ")
			b.WriteString(m.Content)
			b.WriteString("\n")
			lastRole = m.Role
			lastContent = m.Content
		}
		resp := strings.TrimSpace(e.ResponseContent)
		if resp != "" {
			if strings.EqualFold(strings.TrimSpace(lastRole), "assistant") && strings.TrimSpace(lastContent) == resp {
				// Final reply already in Messages; response_content would duplicate.
			} else {
				b.WriteString("Assistant: ")
				b.WriteString(e.ResponseContent)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
