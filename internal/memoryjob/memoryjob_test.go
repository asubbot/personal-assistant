package memoryjob

import (
	"container/heap"
	"context"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/memory"
	"path/filepath"
	"testing"
	"time"
)

// Supporting AC-02.016: among jobs eligible to run (no user-turn block), lower numeric priority runs first.
func TestRunner_drain_orderLowerPriorityFirst(t *testing.T) {
	r := &Runner{
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	var order []int
	r.Enqueue(10, "bg", func(context.Context) error {
		order = append(order, 10)
		return nil
	})
	r.Enqueue(0, "low_pri", func(context.Context) error {
		order = append(order, 0)
		return nil
	})
	r.drain(context.Background())
	if len(order) != 2 || order[0] != 0 || order[1] != 10 {
		t.Fatalf("run order = %v, want [0 10]", order)
	}
}

// Covers AC-02.016: scheduled summarization (priority 10) is not executed while UserTurnActive; runs after it clears.
func TestRunner_drain_defersScheduledDuringUserTurn(t *testing.T) {
	userTurn := true
	var ran int
	r := &Runner{
		deps: Deps{
			UserTurnActive: func() bool { return userTurn },
			Logger:         slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.Enqueue(PriorityScheduled, "summarize_test", func(context.Context) error {
		ran++
		return nil
	})
	r.drain(context.Background())
	if ran != 0 {
		t.Fatalf("expected 0 runs during user turn, got %d", ran)
	}
	userTurn = false
	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("expected 1 run after user turn, got %d", ran)
	}
}

// Covers AC-02.016: catch-up priority (5) is not executed while UserTurnActive; runs after it clears.
func TestRunner_drain_defersCatchUpDuringUserTurn(t *testing.T) {
	userTurn := true
	var ran int
	r := &Runner{
		deps: Deps{
			UserTurnActive: func() bool { return userTurn },
			Logger:         slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.Enqueue(PriorityCatchUp, "catchup_test", func(context.Context) error {
		ran++
		return nil
	})
	r.drain(context.Background())
	if ran != 0 {
		t.Fatalf("expected 0 runs during user turn, got %d", ran)
	}
	userTurn = false
	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("expected 1 run after user turn, got %d", ran)
	}
}

// Covers AC-02.016 / design trade-off: reconciliation (priority 4) still runs while UserTurnActive.
func TestRunner_drain_reconcileNotDeferredDuringUserTurn(t *testing.T) {
	userTurn := true
	var ran int
	r := &Runner{
		deps: Deps{
			UserTurnActive: func() bool { return userTurn },
			Logger:         slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.Enqueue(PriorityReconcile, "recon_test", func(context.Context) error {
		ran++
		return nil
	})
	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("expected reconcile to run during user turn, got ran=%d", ran)
	}
}

func TestDayNeedsCatchUp_noLogFile(t *testing.T) {
	ctx := context.Background()
	logDir := t.TempDir()
	memRoot := t.TempDir()
	loc := time.UTC
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2031, 5, 10, 12, 0, 0, 0, loc)
	cfg := &config.Config{Paths: config.Paths{LLMLogDir: logDir}}
	need, err := dayNeedsCatchUp(ctx, cfg, store, loc, day)
	if err != nil {
		t.Fatalf("dayNeedsCatchUp: %v", err)
	}
	if need {
		t.Fatal("expected no catch-up without llm log file")
	}
}

func TestDayNeedsCatchUp_logEntriesMissingSummary(t *testing.T) {
	ctx := context.Background()
	logDir := t.TempDir()
	memRoot := t.TempDir()
	loc := time.UTC
	day := time.Date(2031, 5, 11, 12, 0, 0, 0, loc)
	dateStr := day.In(loc).Format("2006-01-02")
	logPath := filepath.Join(logDir, "llm-"+dateStr+".jsonl")
	line := `{"request_id":"r1","messages":[],"response_content":"hi","usage":{},"duration_ms":1}` + "\n"
	if err := os.WriteFile(logPath, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Paths: config.Paths{LLMLogDir: logDir}}
	need, err := dayNeedsCatchUp(ctx, cfg, store, loc, day)
	if err != nil {
		t.Fatalf("dayNeedsCatchUp: %v", err)
	}
	if !need {
		t.Fatal("expected catch-up when log has entries but day summary is missing")
	}
}

func TestDayNeedsCatchUp_corruptJSONL_returnsError(t *testing.T) {
	ctx := context.Background()
	logDir := t.TempDir()
	memRoot := t.TempDir()
	loc := time.UTC
	day := time.Date(2031, 5, 12, 12, 0, 0, 0, loc)
	dateStr := day.In(loc).Format("2006-01-02")
	logPath := filepath.Join(logDir, "llm-"+dateStr+".jsonl")
	if err := os.WriteFile(logPath, []byte("not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Paths: config.Paths{LLMLogDir: logDir}}
	_, err = dayNeedsCatchUp(ctx, cfg, store, loc, day)
	if err == nil {
		t.Fatal("expected error from corrupt llm jsonl")
	}
}

func TestMonthNeedsCatchUp_noDaySummaries(t *testing.T) {
	ctx := context.Background()
	memRoot := t.TempDir()
	loc := time.UTC
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{deps: Deps{Memory: store}}
	need, err := r.monthNeedsCatchUp(ctx, 2032, 4)
	if err != nil {
		t.Fatalf("monthNeedsCatchUp: %v", err)
	}
	if need {
		t.Fatal("expected no catch-up when there are no day summaries")
	}
}

func TestMonthNeedsCatchUp_gatherReadError(t *testing.T) {
	ctx := context.Background()
	memRoot := t.TempDir()
	loc := time.UTC
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	day1 := time.Date(2032, 6, 1, 12, 0, 0, 0, loc)
	if err := store.WriteDaySummary(ctx, day1, "day one"); err != nil {
		t.Fatal(err)
	}
	// Make day 2 summary path a directory so ReadDaySummary returns a non-NotExist error.
	badPath := filepath.Join(memRoot, "2032", "06", "02", "summary.md")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatal(err)
	}
	r := &Runner{deps: Deps{Memory: store}}
	_, err = r.monthNeedsCatchUp(ctx, 2032, 6)
	if err == nil {
		t.Fatal("expected error when gathering day summaries hits unreadable path")
	}
}

func TestYearNeedsCatchUp_readYearSummaryError(t *testing.T) {
	ctx := context.Background()
	memRoot := t.TempDir()
	loc := time.UTC
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteMonthSummary(ctx, 2033, 2, "feb"); err != nil {
		t.Fatal(err)
	}
	yearSummaryPath := filepath.Join(memRoot, "2033", "summary.md")
	if err := os.MkdirAll(yearSummaryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	r := &Runner{deps: Deps{Memory: store}}
	_, err = r.yearNeedsCatchUp(ctx, 2033)
	if err == nil {
		t.Fatal("expected error when year summary path is not a readable file")
	}
}
