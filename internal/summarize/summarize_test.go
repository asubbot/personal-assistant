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

// TestDay_noEntries_skips — no log entries for day: no memory write, no vector add, returns nil.
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

// TestDay_withEntries_callsLLMAndWrites — given log entries, one LLM call, memory write and vector Add with expected id.
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

// TestParseDayDate_explicitDate — -date=YYYY-MM-DD is parsed as that day UTC.
func TestParseDayDate_explicitDate(t *testing.T) {
	day, err := ParseDayDate("2026-03-12", "")
	if err != nil {
		t.Fatalf("ParseDayDate: %v", err)
	}
	if day.Year() != 2026 || day.Month() != 3 || day.Day() != 12 {
		t.Errorf("day = %v", day)
	}
	if day.Location() != time.UTC {
		t.Errorf("day location = %v", day.Location())
	}
}

// TestParseDayDate_emptyDate_usesYesterday — empty date with pa_timezone returns yesterday in that zone.
func TestParseDayDate_emptyDate_usesYesterday(t *testing.T) {
	day, err := ParseDayDate("", "UTC")
	if err != nil {
		t.Fatalf("ParseDayDate: %v", err)
	}
	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)
	if day.Year() != yesterday.Year() || day.Month() != yesterday.Month() || day.Day() != yesterday.Day() {
		t.Errorf("day = %v, want yesterday %v", day, yesterday)
	}
}

// TestParseDayDate_invalidDate_returnsError — invalid -date returns error.
func TestParseDayDate_invalidDate_returnsError(t *testing.T) {
	_, err := ParseDayDate("not-a-date", "")
	if err == nil {
		t.Fatal("ParseDayDate(invalid): expected error")
	}
}
