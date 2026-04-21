package memoryjob

import (
	"container/heap"
	"context"
	"errors"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/memory"
	"pa/internal/summarize"
	"pa/internal/vector"
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

// Covers AC-02.016. Supporting AC-33.007: scheduled summarization (priority 10) is not executed while UserTurnActive; runs after it clears.
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

// Covers AC-02.016. Supporting AC-33.007: catch-up priority (5) is not executed while UserTurnActive; runs after it clears.
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

// Covers AC-02.016: traceability for TestDayNeedsCatchUp_noLogFile.
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

// Covers AC-02.016: traceability for TestDayNeedsCatchUp_logEntriesMissingSummary.
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

// Covers AC-02.016: traceability for TestDayNeedsCatchUp_corruptJSONL_returnsError.
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

// Covers AC-02.016: traceability for TestMonthNeedsCatchUp_noDaySummaries.
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

// Covers AC-02.016: traceability for TestMonthNeedsCatchUp_gatherReadError.
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

// Covers AC-02.016: traceability for TestYearNeedsCatchUp_readYearSummaryError.
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

// testEmbedErr fails every Embed call (vector indexing after file write).
type testEmbedErr struct{ err error }

func (e testEmbedErr) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, e.err
}

type testLLMOK struct{ content string }

func (m testLLMOK) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	_ = ctx
	_ = messages
	_ = opts
	return &llm.CompletionResult{Content: m.content}, nil
}

type testVecOK struct{}

func (testVecOK) Add(ctx context.Context, id string, embedding []float32, text string) error {
	return nil
}
func (testVecOK) Delete(ctx context.Context, id string) error { return nil }
func (testVecOK) Search(ctx context.Context, queryEmbedding []float32, topK int) ([]vector.SearchResult, error) {
	return nil, nil
}
func (testVecOK) Exists(ctx context.Context, id string) (bool, error) { return false, nil }
func (testVecOK) Clear(ctx context.Context) error                     { return nil }
func (testVecOK) Close() error                                        { return nil }

// Covers AC-02.017: month summarization file write then vector failure enqueues reconcile_month job.
func TestRunMonth_embedFailsAfterFileWrite_enqueuesReconcileMonth(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	memRoot := t.TempDir()
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	d1 := time.Date(2026, 3, 5, 12, 0, 0, 0, loc)
	d2 := time.Date(2026, 3, 6, 12, 0, 0, 0, loc)
	if err := store.WriteDaySummary(ctx, d1, "alpha"); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteDaySummary(ctx, d2, "beta"); err != nil {
		t.Fatal(err)
	}

	r := &Runner{
		deps: Deps{
			Memory:      store,
			Loc:         loc,
			LLMProvider: testLLMOK{content: "month rollup"},
			Embedder:    testEmbedErr{err: errors.New("embed failed")},
			Vector:      testVecOK{},
			Logger:      slog.New(slog.DiscardHandler),
			Cfg:         &config.Config{},
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)

	if err := r.runMonth(ctx, 2026, 3); err != nil {
		t.Fatalf("runMonth: %v", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pq.Len() != 1 {
		t.Fatalf("queue len = %d, want 1 reconcile job", r.pq.Len())
	}
	it := heap.Pop(&r.pq).(*jobItem)
	want := "reconcile_month:" + summarize.VectorIDPrefixMonth + "2026-03"
	if it.name != want {
		t.Fatalf("job name = %q, want %q", it.name, want)
	}
}

// Covers AC-02.017: year summarization file write then vector failure enqueues reconcile_year job.
func TestRunYear_embedFailsAfterFileWrite_enqueuesReconcileYear(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	memRoot := t.TempDir()
	store, err := memory.NewStore(memRoot, loc)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteMonthSummary(ctx, 2027, 1, "jan"); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteMonthSummary(ctx, 2027, 2, "feb"); err != nil {
		t.Fatal(err)
	}

	r := &Runner{
		deps: Deps{
			Memory:      store,
			Loc:         loc,
			LLMProvider: testLLMOK{content: "year rollup"},
			Embedder:    testEmbedErr{err: errors.New("embed failed")},
			Vector:      testVecOK{},
			Logger:      slog.New(slog.DiscardHandler),
			Cfg:         &config.Config{},
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)

	if err := r.runYear(ctx, 2027); err != nil {
		t.Fatalf("runYear: %v", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pq.Len() != 1 {
		t.Fatalf("queue len = %d, want 1 reconcile job", r.pq.Len())
	}
	it := heap.Pop(&r.pq).(*jobItem)
	want := "reconcile_year:2027"
	if it.name != want {
		t.Fatalf("job name = %q, want %q", it.name, want)
	}
}

// Covers AC-33.001, AC-33.004, AC-33.007, AC-33.008, AC-33.011. Supporting AC-33.012: retry waits for notBefore and runs in existing queue loop.
func TestRunner_retryableDayJob_waitsThenRetries(t *testing.T) {
	baseNow := time.Date(2035, 1, 10, 10, 0, 0, 0, time.UTC)
	now := baseNow
	ran := 0
	r := &Runner{
		deps: Deps{
			Now:    func() time.Time { return now },
			Logger: slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	targetDay := time.Date(2035, 1, 9, 12, 0, 0, 0, time.UTC)
	r.enqueueDayRetry(PriorityCatchUp, "catchup_day", targetDay, func(context.Context, time.Time) error {
		ran++
		if ran == 1 {
			return errors.New("timeout from provider")
		}
		return nil
	})

	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("runs after first drain = %d, want 1", ran)
	}
	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("runs before notBefore = %d, want 1", ran)
	}
	now = now.Add(time.Minute)
	r.drain(context.Background())
	if ran != 2 {
		t.Fatalf("runs after backoff = %d, want 2", ran)
	}
}

// Covers AC-33.002, AC-33.005: retries stop after bounded attempt budget.
func TestRunner_retryableDayJob_exhaustsRetries(t *testing.T) {
	baseNow := time.Date(2035, 2, 10, 10, 0, 0, 0, time.UTC)
	now := baseNow
	ran := 0
	r := &Runner{
		deps: Deps{
			Now:    func() time.Time { return now },
			Logger: slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	targetDay := time.Date(2035, 2, 9, 12, 0, 0, 0, time.UTC)
	r.enqueueDayRetry(PriorityScheduled, "summarize_yesterday", targetDay, func(context.Context, time.Time) error {
		ran++
		return errors.New("temporary upstream failure")
	})

	for i := 0; i < 6; i++ {
		r.drain(context.Background())
		now = now.Add(61 * time.Minute)
	}
	if ran != 5 {
		t.Fatalf("runs = %d, want 5 (1 initial + 4 retries)", ran)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pq.Len() != 0 {
		t.Fatalf("queue len = %d, want 0 after exhaustion", r.pq.Len())
	}
}

// Covers AC-33.006: duplicate retry chains for the same day key are not queued.
func TestRunner_retryDayDedupe_preventsDuplicateQueueEntries(t *testing.T) {
	var first, second int
	r := &Runner{
		deps: Deps{
			Logger: slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	targetDay := time.Date(2035, 3, 9, 12, 0, 0, 0, time.UTC)
	r.enqueueDayRetry(PriorityCatchUp, "catchup_day", targetDay, func(context.Context, time.Time) error {
		first++
		return nil
	})
	r.enqueueDayRetry(PriorityCatchUp, "summarize_yesterday", targetDay, func(context.Context, time.Time) error {
		second++
		return nil
	})
	r.drain(context.Background())
	if first+second != 1 {
		t.Fatalf("executed jobs = %d, want 1", first+second)
	}
}

// Covers AC-33.004. Supporting AC-33.001 and AC-33.002: retries preserve the original day target across midnight.
func TestRunner_retryPreservesOriginalDayTargetAcrossMidnight(t *testing.T) {
	now := time.Date(2035, 4, 10, 23, 59, 0, 0, time.UTC)
	target := time.Date(2035, 4, 9, 12, 0, 0, 0, time.UTC)
	seen := make([]string, 0, 2)
	r := &Runner{
		deps: Deps{
			Now:    func() time.Time { return now },
			Logger: slog.New(slog.DiscardHandler),
			Loc:    time.UTC,
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.enqueueDayRetry(PriorityScheduled, "summarize_yesterday", target, func(ctx context.Context, day time.Time) error {
		_ = ctx
		seen = append(seen, day.Format("2006-01-02"))
		if len(seen) == 1 {
			return errors.New("timeout")
		}
		return nil
	})
	r.drain(context.Background())
	now = now.Add(2 * time.Minute)
	r.drain(context.Background())
	if len(seen) != 2 {
		t.Fatalf("seen runs = %d, want 2", len(seen))
	}
	if seen[0] != "2035-04-09" || seen[1] != "2035-04-09" {
		t.Fatalf("target days = %v, want [2035-04-09 2035-04-09]", seen)
	}
}

// Supporting AC-33.006: startup and scheduled enqueue paths dedupe same day target key.
func TestRunner_startupAndScheduledDaily_shareOneDayKey(t *testing.T) {
	now := time.Date(2035, 5, 10, 1, 5, 0, 0, time.UTC)
	r := &Runner{
		deps: Deps{
			Now:    func() time.Time { return now },
			Logger: slog.New(slog.DiscardHandler),
			Loc:    time.UTC,
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.enqueueStartup()
	r.maybeEnqueueDaily(now, time.UTC)

	r.mu.Lock()
	defer r.mu.Unlock()
	dayKeyCount := 0
	for _, it := range r.pq {
		if it.key == "day:2035-05-09" {
			dayKeyCount++
		}
	}
	if dayKeyCount != 1 {
		t.Fatalf("day key count = %d, want 1", dayKeyCount)
	}
}
