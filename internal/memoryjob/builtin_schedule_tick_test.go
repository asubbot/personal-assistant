package memoryjob

import (
	"container/heap"
	"context"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/memory"
	"pa/internal/summarize"
	"pa/internal/vector"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tickVec is a minimal vector.Store for built-in schedule tests.
type tickVec struct {
	addIDs []string
}

func (v *tickVec) Add(ctx context.Context, id string, embedding []float32, text string) error {
	v.addIDs = append(v.addIDs, id)
	return nil
}

func (v *tickVec) Delete(context.Context, string) error { return nil }

func (v *tickVec) Clear(context.Context) error { return nil }

func (v *tickVec) Search(context.Context, []float32, int) ([]vector.SearchResult, error) {
	return nil, nil
}

func (v *tickVec) Exists(context.Context, string) (bool, error) { return false, nil }

func (v *tickVec) Close() error { return nil }

type tickEmbedder struct{}

func (tickEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return []float32{1, 0, 0}, nil
}

// Covers AC-02.004: with memory_dir, llm_log_dir, embedder, and vector store wired, a single in-process onTick
// at local 01:xx enqueues and runs built-in day summarization (no external cron).
func TestOnTick_builtinDayScheduleWritesMemoryAndVector(t *testing.T) {
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
	vec := &tickVec{}
	emb := tickEmbedder{}
	fixedNow := time.Date(2026, 4, 10, 1, 5, 0, 0, time.UTC)

	r := &Runner{
		deps: Deps{
			Cfg: &config.Config{
				Paths: config.Paths{
					LLMLogDir: llmDir,
					MemoryDir: memDir,
				},
			},
			Loc:         time.UTC,
			Memory:      mem,
			Vector:      vec,
			Embedder:    emb,
			LLMProvider: &catchupFakeLLM{content: "Scheduled day summary.\n\n## S\n\n- fact"},
			Logger:      slog.New(slog.DiscardHandler),
			Now:         func() time.Time { return fixedNow },
		},
	}
	heap.Init(&r.pq)

	r.onTick(context.Background())

	got, err := mem.ReadDaySummary(context.Background(), logDay)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Scheduled day summary") {
		t.Fatalf("day summary: %q", got)
	}
	wantID := summarize.VectorIDPrefixDay + "2026-04-09"
	if len(vec.addIDs) != 1 || vec.addIDs[0] != wantID {
		t.Fatalf("vector adds: %v want [%s]", vec.addIDs, wantID)
	}
}
