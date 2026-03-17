package summarize

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/memory"
	"pa/internal/vector"
	"path/filepath"
	"testing"
	"time"
)

type mockLLM struct {
	content string
	calls   int
}

func (m *mockLLM) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.calls++
	return &llm.CompletionResult{Content: m.content}, nil
}

type mockEmbedder struct {
	vec []float32
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.vec, nil
}

type mockVectorStore struct {
	adds    []string // ids added
	deletes []string // ids deleted
}

func (m *mockVectorStore) Add(ctx context.Context, id string, embedding []float32, text string) error {
	m.adds = append(m.adds, id)
	return nil
}

func (m *mockVectorStore) Delete(ctx context.Context, id string) error {
	m.deletes = append(m.deletes, id)
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, queryEmbedding []float32, topK int) ([]vector.SearchResult, error) {
	return nil, nil
}

func (m *mockVectorStore) Close() error { return nil }

// Covers AC-01.011, AC-01.012 (US-06): day summarization with no log entries skips write; no memory or vector update.
func TestDay_noEntries_skips(t *testing.T) {
	dir := t.TempDir()
	llmLogDir := filepath.Join(dir, "llm_logs")
	if err := os.MkdirAll(llmLogDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	cfg := DayConfig{
		LLMLogDir:   llmLogDir,
		LLMProvider: &mockLLM{content: "summary"},
		MemoryStore: memStore,
		Logger:      slog.Default(),
	}

	err = Day(context.Background(), day, cfg)
	if err != nil {
		t.Fatalf("Day(no entries): %v", err)
	}

	content, _ := memStore.ReadDaySummary(context.Background(), day)
	if content != "" {
		t.Errorf("expected no summary written when no entries; got %q", content)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): day summarization with log entries writes summary to memory and vector store (calendar path, expected id).
func TestDay_withEntries_callsLLMAndWrites(t *testing.T) {
	dir := t.TempDir()
	llmLogDir := filepath.Join(dir, "llm_logs")
	if err := os.MkdirAll(llmLogDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Write one entry for 2026-03-12
	logPath := filepath.Join(llmLogDir, "llm-2026-03-12.jsonl")
	entry := llmlog.Entry{
		RequestID:       "r1",
		Messages:        []llm.Message{{Role: "user", Content: "hello"}},
		ResponseContent: "hi there",
		Usage:           llm.Usage{},
		DurationMs:      1,
	}
	line, _ := json.Marshal(entry)
	if err := os.WriteFile(logPath, append(line, '\n'), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	llmMock := &mockLLM{content: "User said hello; assistant replied."}
	vecMock := &mockVectorStore{}

	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	cfg := DayConfig{
		LLMLogDir:   llmLogDir,
		LLMProvider: llmMock,
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		VectorStore: vecMock,
		Logger:      slog.Default(),
	}

	err = Day(context.Background(), day, cfg)
	if err != nil {
		t.Fatalf("Day: %v", err)
	}

	if llmMock.calls != 1 {
		t.Errorf("LLM calls = %d, want 1", llmMock.calls)
	}

	content, err := memStore.ReadDaySummary(context.Background(), day)
	if err != nil {
		t.Fatalf("ReadDaySummary: %v", err)
	}
	if content != "User said hello; assistant replied." {
		t.Errorf("memory summary = %q", content)
	}

	wantID := "summary:day:2026-03-12"
	if len(vecMock.deletes) != 1 || vecMock.deletes[0] != wantID {
		t.Errorf("vector deletes = %v, want [%s]", vecMock.deletes, wantID)
	}
	if len(vecMock.adds) != 1 || vecMock.adds[0] != wantID {
		t.Errorf("vector adds = %v, want [%s]", vecMock.adds, wantID)
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — YYYY-MM-DD parses as day scope.
func TestParseSummarizeScope_day(t *testing.T) {
	scope, err := ParseSummarizeScope("2026-03-12")
	if err != nil {
		t.Fatalf("ParseSummarizeScope: %v", err)
	}
	if scope.Kind != "day" {
		t.Fatalf("Kind = %q, want day", scope.Kind)
	}
	if scope.Day.Year() != 2026 || scope.Day.Month() != 3 || scope.Day.Day() != 12 {
		t.Errorf("Day = %v", scope.Day)
	}
	if scope.Day.Location() != time.UTC {
		t.Errorf("Day location = %v", scope.Day.Location())
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — YYYY-MM parses as month scope.
func TestParseSummarizeScope_month(t *testing.T) {
	scope, err := ParseSummarizeScope("2026-03")
	if err != nil {
		t.Fatalf("ParseSummarizeScope: %v", err)
	}
	if scope.Kind != "month" || scope.Year != 2026 || scope.Month != 3 {
		t.Errorf("scope = %+v", scope)
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — YYYY parses as year scope.
func TestParseSummarizeScope_year(t *testing.T) {
	scope, err := ParseSummarizeScope("2026")
	if err != nil {
		t.Fatalf("ParseSummarizeScope: %v", err)
	}
	if scope.Kind != "year" || scope.Year != 2026 {
		t.Errorf("scope = %+v", scope)
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — empty value returns error.
func TestParseSummarizeScope_empty_returnsError(t *testing.T) {
	_, err := ParseSummarizeScope("")
	if err == nil {
		t.Fatal("ParseSummarizeScope(empty): expected error")
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — invalid format returns error.
func TestParseSummarizeScope_invalid_returnsError(t *testing.T) {
	_, err := ParseSummarizeScope("not-a-date")
	if err == nil {
		t.Fatal("ParseSummarizeScope(invalid): expected error")
	}
}

// Covers AC-01.011, AC-01.012 (US-06): month summarization with no day summaries skips write.
func TestMonth_noDaySummaries_skips(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	cfg := MonthConfig{
		LLMProvider: &mockLLM{content: "month summary"},
		MemoryStore: memStore,
		Logger:      slog.Default(),
	}

	err = Month(context.Background(), 2026, 3, cfg)
	if err != nil {
		t.Fatalf("Month(no day summaries): %v", err)
	}

	content, _ := memStore.ReadMonthSummary(context.Background(), 2026, 3)
	if content != "" {
		t.Errorf("expected no month summary when no day summaries; got %q", content)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): month summarization writes month summary to memory (YYYY/MM/summary.md) and vector store.
func TestMonth_withDaySummaries_callsLLMAndWrites(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	if err := memStore.WriteDaySummary(context.Background(), day1, "Day 1 summary."); err != nil {
		t.Fatalf("WriteDaySummary 1: %v", err)
	}
	if err := memStore.WriteDaySummary(context.Background(), day2, "Day 15 summary."); err != nil {
		t.Fatalf("WriteDaySummary 2: %v", err)
	}

	llmMock := &mockLLM{content: "March overview: two active days."}
	vecMock := &mockVectorStore{}

	cfg := MonthConfig{
		LLMProvider: llmMock,
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{vec: []float32{0, 1, 0, 0}},
		VectorStore: vecMock,
		Logger:      slog.Default(),
	}

	err = Month(context.Background(), 2026, 3, cfg)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}

	if llmMock.calls != 1 {
		t.Errorf("LLM calls = %d, want 1", llmMock.calls)
	}

	content, err := memStore.ReadMonthSummary(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("ReadMonthSummary: %v", err)
	}
	if content != "March overview: two active days." {
		t.Errorf("memory month summary = %q", content)
	}

	wantID := "summary:month:2026-03"
	if len(vecMock.deletes) != 1 || vecMock.deletes[0] != wantID {
		t.Errorf("vector deletes = %v, want [%s]", vecMock.deletes, wantID)
	}
	if len(vecMock.adds) != 1 || vecMock.adds[0] != wantID {
		t.Errorf("vector adds = %v, want [%s]", vecMock.adds, wantID)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): year summarization with no month summaries skips write.
func TestYear_noMonthSummaries_skips(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	cfg := YearConfig{
		LLMProvider: &mockLLM{content: "year summary"},
		MemoryStore: memStore,
		Logger:      slog.Default(),
	}

	err = Year(context.Background(), 2026, cfg)
	if err != nil {
		t.Fatalf("Year(no month summaries): %v", err)
	}

	content, _ := memStore.ReadYearSummary(context.Background(), 2026)
	if content != "" {
		t.Errorf("expected no year summary when no month summaries; got %q", content)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): year summarization writes year summary to memory (YYYY/summary.md) and vector store.
func TestYear_withMonthSummaries_callsLLMAndWrites(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if err := memStore.WriteMonthSummary(context.Background(), 2026, 1, "January summary."); err != nil {
		t.Fatalf("WriteMonthSummary 1: %v", err)
	}
	if err := memStore.WriteMonthSummary(context.Background(), 2026, 6, "June summary."); err != nil {
		t.Fatalf("WriteMonthSummary 2: %v", err)
	}

	llmMock := &mockLLM{content: "2026 overview: active in Jan and Jun."}
	vecMock := &mockVectorStore{}

	cfg := YearConfig{
		LLMProvider: llmMock,
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{vec: []float32{0, 0, 1, 0}},
		VectorStore: vecMock,
		Logger:      slog.Default(),
	}

	err = Year(context.Background(), 2026, cfg)
	if err != nil {
		t.Fatalf("Year: %v", err)
	}

	if llmMock.calls != 1 {
		t.Errorf("LLM calls = %d, want 1", llmMock.calls)
	}

	content, err := memStore.ReadYearSummary(context.Background(), 2026)
	if err != nil {
		t.Fatalf("ReadYearSummary: %v", err)
	}
	if content != "2026 overview: active in Jan and Jun." {
		t.Errorf("memory year summary = %q", content)
	}

	wantID := "summary:year:2026"
	if len(vecMock.deletes) != 1 || vecMock.deletes[0] != wantID {
		t.Errorf("vector deletes = %v, want [%s]", vecMock.deletes, wantID)
	}
	if len(vecMock.adds) != 1 || vecMock.adds[0] != wantID {
		t.Errorf("vector adds = %v, want [%s]", vecMock.adds, wantID)
	}
}
