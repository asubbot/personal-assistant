package memoryjob

import (
	"context"
	"pa/internal/memory"
	"pa/internal/vector"
	"strings"
	"testing"
	"time"
)

type vecCapture struct {
	lastAddID   string
	lastAddText string
}

func (v *vecCapture) Add(ctx context.Context, id string, embedding []float32, text string) error {
	v.lastAddID = id
	v.lastAddText = text
	return nil
}

func (v *vecCapture) Delete(context.Context, string) error { return nil }

func (v *vecCapture) Clear(context.Context) error { return nil }
func (v *vecCapture) Search(context.Context, []float32, int) ([]vector.SearchResult, error) {
	return nil, nil
}
func (v *vecCapture) Exists(context.Context, string) (bool, error) { return false, nil }
func (v *vecCapture) Close() error                                 { return nil }

type embedFixed struct{}

func (embedFixed) Embed(context.Context, string) ([]float32, error) {
	return []float32{1, 0, 0, 0}, nil
}

// Covers AC-02.017: ReindexDaySummary embeds an existing day summary file into the vector store without calling the summarization LLM.
func TestReindexDaySummary_fromExistingFile(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	body := "already written summary"
	if err := store.WriteDaySummary(context.Background(), day, body); err != nil {
		t.Fatal(err)
	}
	cap := &vecCapture{}
	if err := ReindexDaySummary(context.Background(), store, cap, embedFixed{}, time.UTC, "2026-05-01"); err != nil {
		t.Fatal(err)
	}
	if cap.lastAddID != "summary:day:2026-05-01" {
		t.Errorf("Add id = %q", cap.lastAddID)
	}
	if cap.lastAddText == "" {
		t.Fatal("empty vector text")
	}
	if !strings.Contains(cap.lastAddText, "2026-05-01") || !strings.Contains(cap.lastAddText, body) {
		t.Errorf("Add text = %q, want date and body", cap.lastAddText)
	}
}
