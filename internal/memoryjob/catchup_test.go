package memoryjob

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/memory"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type catchupFakeLLM struct {
	content string
}

func (f *catchupFakeLLM) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	return &llm.CompletionResult{Content: f.content}, nil
}

func writeLLMLogEntryForDay(t *testing.T, llmDir string, day time.Time) {
	t.Helper()
	name := "llm-" + day.Format("2006-01-02") + ".jsonl"
	path := filepath.Join(llmDir, name)
	entry := &llmlog.Entry{
		RequestID:       "r1",
		Messages:        []llm.Message{{Role: "user", Content: "hi"}},
		ResponseContent: "reply",
		Usage:           llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
		DurationMs:      1,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(line, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Covers AC-02.005: catch-up day job writes missing day summary when LLM log exists (fixed clock; no vector).
func TestJobCatchUpDay_writesMissingDaySummary(t *testing.T) {
	base := t.TempDir()
	llmDir := filepath.Join(base, "llm")
	memDir := filepath.Join(base, "mem")
	if err := os.MkdirAll(llmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logDay := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	writeLLMLogEntryForDay(t, llmDir, logDay)

	mem, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	fixedNow := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		deps: Deps{
			Cfg:         &config.Config{Paths: config.Paths{LLMLogDir: llmDir}},
			Loc:         time.UTC,
			Memory:      mem,
			LLMProvider: &catchupFakeLLM{content: "Overview.\n\n## Facts\n\n- One fact."},
			Logger:      slog.Default(),
			Now:         func() time.Time { return fixedNow },
		},
	}
	if err := r.jobCatchUpDay(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := mem.ReadDaySummary(context.Background(), logDay)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Overview") {
		t.Fatalf("summary: %q", got)
	}
}

// Covers AC-02.006: catch-up month writes missing month summary when day summaries exist.
func TestJobCatchUpMonth_writesMissingMonthSummary(t *testing.T) {
	base := t.TempDir()
	memDir := filepath.Join(base, "mem")
	mem, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	d1 := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	if err := mem.WriteDaySummary(context.Background(), d1, "day one"); err != nil {
		t.Fatal(err)
	}
	if err := mem.WriteDaySummary(context.Background(), d2, "day two"); err != nil {
		t.Fatal(err)
	}

	fixedNow := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		deps: Deps{
			Cfg:         &config.Config{Paths: config.Paths{}},
			Loc:         time.UTC,
			Memory:      mem,
			LLMProvider: &catchupFakeLLM{content: "Month rollup.\n\n## M\n\n- m"},
			Logger:      slog.Default(),
			Now:         func() time.Time { return fixedNow },
		},
	}
	if err := r.jobCatchUpMonth(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := mem.ReadMonthSummary(context.Background(), 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Month rollup") {
		t.Fatalf("month summary: %q", got)
	}
}

// Covers AC-02.007: catch-up year writes missing year summary when at least one month summary exists.
func TestJobCatchUpYear_writesMissingYearSummary(t *testing.T) {
	base := t.TempDir()
	memDir := filepath.Join(base, "mem")
	mem, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if err := mem.WriteMonthSummary(context.Background(), 2026, 1, "january month text"); err != nil {
		t.Fatal(err)
	}

	fixedNow := time.Date(2027, 6, 1, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		deps: Deps{
			Cfg:         &config.Config{Paths: config.Paths{}},
			Loc:         time.UTC,
			Memory:      mem,
			LLMProvider: &catchupFakeLLM{content: "Year rollup.\n\n## Y\n\n- y"},
			Logger:      slog.Default(),
			Now:         func() time.Time { return fixedNow },
		},
	}
	if err := r.jobCatchUpYear(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := mem.ReadYearSummary(context.Background(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Year rollup") {
		t.Fatalf("year summary: %q", got)
	}
}
