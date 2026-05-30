package tools

import (
	"context"
	"pa/internal/embedding"
	"pa/internal/memory"
	"pa/internal/sqlitepragma"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Covers AC-16.005: underMemoryRoot rejects paths outside memory_dir (shared with write_memory).
func TestUnderMemoryRoot_rejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere", "notes.md")
	if underMemoryRoot(root, outside) {
		t.Fatalf("expected false for path outside root: root=%q path=%q", root, outside)
	}
}

// Covers AC-16.006: read_memory returns distinct headings for automatic summary vs notes.md body when both exist.
func TestReadMemoryTool_singleDate_summaryAndNotesSections(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	day := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	if err := store.WriteDaySummary(ctx, day, "auto text"); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendDayNote(ctx, day, "manual line", "", time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC), 4096, 1<<20); err != nil {
		t.Fatal(err)
	}
	tool := NewReadMemoryTool(store, 31, 262144)
	out, err := tool.Run(ctx, map[string]any{"date": "2026-04-03"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "### Automatic summary") || !strings.Contains(out, "auto text") {
		t.Fatalf("missing summary section: %q", out)
	}
	if !strings.Contains(out, "### Manual notes") || !strings.Contains(out, "manual line") {
		t.Fatalf("missing notes section: %q", out)
	}
}

// Covers AC-16.007: read_memory omits calendar days that have neither summary nor notes (no empty day placeholder).
func TestReadMemoryTool_range_skipsMiddleEmptyDay(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	d1 := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := store.WriteDaySummary(ctx, d1, "first"); err != nil {
		t.Fatal(err)
	}
	d3 := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	if err := store.AppendDayNote(ctx, d3, "third", "", time.Date(2026, 4, 12, 8, 0, 0, 0, time.UTC), 4096, 1<<20); err != nil {
		t.Fatal(err)
	}
	tool := NewReadMemoryTool(store, 31, 262144)
	out, err := tool.Run(ctx, map[string]any{"from": "2026-04-10", "to": "2026-04-12"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "## 2026-04-11") {
		t.Fatalf("middle empty day must be absent: %q", out)
	}
}

// Covers AC-16.016: after write_memory, notes vector search returns the new notes id.
func TestWriteMemoryTool_indexesNotesVector(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	vecPath := filepath.Join(t.TempDir(), "notes.sqlite")
	notesVec, err := sqlite.NewWithTable(vecPath, 4, sqlite.TableNotes, sqlitepragma.RecommendedPolicy(false))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = notesVec.Close() }()
	emb := &stubEmbedder{out: []float32{1, 0, 0, 0}}
	tool := NewWriteMemoryTool(store, notesVec, emb, 4096, 1<<20)
	_, err = tool.Run(context.Background(), map[string]any{
		"text": "unique-ep016-note-phrase",
		"date": "2026-04-20",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := notesVec.Search(context.Background(), []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range res {
		if strings.Contains(r.Text, "unique-ep016-note-phrase") && strings.HasPrefix(r.ID, "notes:2026-04-20") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("notes search missing new row: %+v", res)
	}
}

// Covers AC-35.019: write_memory refuses to index chunk text containing a forbidden PA marker line.
func TestWriteMemoryTool_rejectsForbiddenMarkerLine(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	vecPath := filepath.Join(t.TempDir(), "notes.sqlite")
	notesVec, err := sqlite.NewWithTable(vecPath, 4, sqlite.TableNotes, sqlitepragma.RecommendedPolicy(false))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = notesVec.Close() }()
	emb := &stubEmbedder{out: []float32{1, 0, 0, 0}}
	tool := NewWriteMemoryTool(store, notesVec, emb, 4096, 1<<20)
	_, err = tool.Run(context.Background(), map[string]any{
		"text": "before\n<<<PA_BEGIN_CONTEXT>>>\nafter",
		"date": "2026-04-21",
	})
	if err == nil {
		t.Fatal("expected error for forbidden PA marker line in indexed text")
	}
}

// Covers AC-16.017: write_memory with kind=preference records preference in notes.md.
func TestWriteMemoryTool_kindPreferenceInFile(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	tool := NewWriteMemoryTool(store, nil, nil, 4096, 1<<20)
	_, err = tool.Run(context.Background(), map[string]any{
		"text": "my preference text",
		"date": "2026-04-21",
		"kind": "preference",
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := store.ReadDayNotes(context.Background(), time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "kind=preference") {
		t.Fatalf("expected kind=preference in notes.md: %q", body)
	}
}

// Covers AC-16.004: write_memory surfaces max_append_bytes when entry exceeds configured limit.
func TestWriteMemoryTool_rejectsOversizedTextPerAppendLimit(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	tool := NewWriteMemoryTool(store, nil, nil, 30, 1<<20)
	_, err = tool.Run(context.Background(), map[string]any{"text": strings.Repeat("x", 80), "date": "2026-04-22"})
	if err == nil || !strings.Contains(err.Error(), "max_append_bytes") {
		t.Fatalf("expected max_append_bytes error, got %v", err)
	}
}

// Covers AC-16.018: write_memory registers as a native tool with description and JSON schema.
func TestWriteMemoryTool_llmSurfaceNonEmpty(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	tool := NewWriteMemoryTool(store, nil, nil, 1024, 1<<20)
	if tool.Name() != "write_memory" {
		t.Fatalf("name %q", tool.Name())
	}
	if strings.TrimSpace(tool.Description()) == "" {
		t.Fatal("empty description")
	}
	if len(tool.ParamsSchema()) == 0 {
		t.Fatal("empty params schema")
	}
}

type stubEmbedder struct{ out []float32 }

func (s *stubEmbedder) Embed(context.Context, string) ([]float32, error) { return s.out, nil }

var _ embedding.Embedder = (*stubEmbedder)(nil)
