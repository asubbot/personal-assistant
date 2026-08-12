// Package memoryjob runs automatic summarization, catch-up, and vector reconciliation (EP-002).
package memoryjob

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/embedding"
	"pa/internal/lifecyclelog"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/memory"
	"pa/internal/patime"
	"pa/internal/summarize"
	"pa/internal/vector"
	"strings"
	"sync"
	"time"
)

// Priority constants: lower runs first (REQ-02.015).
const (
	PriorityReconcile = 4
	PriorityCatchUp   = 5
	PriorityScheduled = 10
)

// Built-in summarization timing (not configurable; EP-002).
const (
	summarizeHour     = 1
	summarizeMinute   = 0
	jobTimeoutSeconds = 1800
	// reconciliationScanDays is how many past calendar days startup reconciliation compares
	// (day summary file vs vector Exists). Bounded day-only window per EP-002 system design
	// (e.g. last 90 days in ai-sdlc-artefacts/epics/EP-002/ep-system-design.md); not JSON-configurable.
	reconciliationScanDays = 90
	schedulerTickSeconds   = 60
)

// Deps wires summarization and reconciliation.
type Deps struct {
	Cfg         *config.Config
	Loc         *time.Location
	Memory      *memory.Store
	Vector      vector.Store
	Embedder    embedding.Embedder
	LLMProvider llm.Provider
	Logger      *slog.Logger
	// Now returns the wall clock for scheduling and catch-up jobs; nil means time.Now (production).
	Now func() time.Time
	// UserTurnActive reports whether an interactive user LLM turn is in progress (REQ-02.015).
	// Nil defaults to core.UserTurnInProgress.
	UserTurnActive func() bool
}

// Runner owns a priority queue and a worker goroutine.
type Runner struct {
	deps     Deps
	mu       sync.Mutex
	pq       priorityQueue
	queued   map[string]struct{}
	seq      int64
	wake     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	stop     context.CancelFunc

	lastDailyFireKey string
	lastMonthFireKey string
	lastYearFireKey  string
}

func (r *Runner) now() time.Time {
	if r.deps.Now != nil {
		return r.deps.Now()
	}
	return time.Now()
}

func (r *Runner) userTurnActive() bool {
	if r.deps.UserTurnActive != nil {
		return r.deps.UserTurnActive()
	}
	return core.UserTurnInProgress()
}

type jobItem struct {
	priority  int
	seq       int64
	name      string
	key       string
	attempt   int
	notBefore time.Time
	delays    []time.Duration
	run       func(context.Context) error
}

type priorityQueue []*jobItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].priority != pq[j].priority {
		return pq[i].priority < pq[j].priority
	}
	return pq[i].seq < pq[j].seq
}

func (pq priorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *priorityQueue) Push(x any) {
	// type-assertion: safe — container/heap only pushes the queue's private *jobItem values
	*pq = append(*pq, x.(*jobItem))
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	it := old[n-1]
	*pq = old[0 : n-1]
	return it
}

// Start launches the worker; call Stop on shutdown.
func Start(ctx context.Context, deps Deps) *Runner {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.Loc == nil {
		deps.Loc = time.UTC
	}
	runCtx, cancel := context.WithCancel(ctx)
	r := &Runner{
		deps:   deps,
		wake:   make(chan struct{}, 1),
		done:   make(chan struct{}),
		stop:   cancel,
		queued: make(map[string]struct{}),
	}
	heap.Init(&r.pq)
	go r.loop(runCtx)
	return r
}

func (r *Runner) loop(ctx context.Context) {
	defer close(r.done)
	tick := time.NewTimer(time.Second)
	defer tick.Stop()
	r.enqueueStartup()
	for {
		select {
		case <-ctx.Done():
			return
		// unguarded-shared-field: safe — wake is immutable after construction and channels are concurrency-safe
		case <-r.wake:
			r.drain(ctx)
		case <-tick.C:
			r.onTick(ctx)
			tick.Reset(time.Duration(schedulerTickSeconds) * time.Second)
		}
	}
}

func (r *Runner) onTick(ctx context.Context) {
	now := r.now()
	loc := r.deps.Loc
	r.maybeEnqueueDaily(now, loc)
	r.maybeEnqueueMonthRollup(now, loc)
	r.maybeEnqueueYearRollup(now, loc)
	r.drain(ctx)
}

// maybeEnqueueDaily fires once per local calendar day key when the tick falls in hour summarizeHour
// (e.g. any 01:xx with summarizeMinute 0), not only at minute 00; lastDailyFireKey prevents repeats.
func (r *Runner) maybeEnqueueDaily(now time.Time, loc *time.Location) {
	t := now.In(loc)
	if t.Hour() != summarizeHour || t.Minute() < summarizeMinute {
		return
	}
	fireKey := t.Format("2006-01-02")
	if r.lastDailyFireKey == fireKey {
		return
	}
	r.lastDailyFireKey = fireKey
	day := previousCalendarDayNoon(loc, now)
	r.enqueueDayRetry(PriorityScheduled, "summarize_yesterday", day, r.jobSummarizeDayFor)
}

func (r *Runner) maybeEnqueueMonthRollup(now time.Time, loc *time.Location) {
	t := now.In(loc)
	if !patime.IsFirstDayOfMonth(loc, now) || t.Hour() != summarizeHour || t.Minute() < summarizeMinute {
		return
	}
	monthKey := fmt.Sprintf("%04d-%02d", t.Year(), t.Month())
	if r.lastMonthFireKey == monthKey {
		return
	}
	r.lastMonthFireKey = monthKey
	py, pm := patime.PreviousMonth(loc, now)
	r.Enqueue(PriorityScheduled, "summarize_prev_month", func(ctx context.Context) error {
		return r.runMonth(ctx, py, int(pm))
	})
}

func (r *Runner) maybeEnqueueYearRollup(now time.Time, loc *time.Location) {
	t := now.In(loc)
	if !patime.IsFirstDayOfYear(loc, now) || t.Hour() != summarizeHour || t.Minute() < summarizeMinute {
		return
	}
	yearKey := fmt.Sprintf("%04d", t.Year())
	if r.lastYearFireKey == yearKey {
		return
	}
	r.lastYearFireKey = yearKey
	py := patime.PreviousYear(loc, now)
	r.Enqueue(PriorityScheduled, "summarize_prev_year", func(ctx context.Context) error {
		return r.runYear(ctx, py)
	})
}

func (r *Runner) enqueueStartup() {
	r.Enqueue(PriorityReconcile, "reconciliation_scan", func(ctx context.Context) error {
		return r.runReconciliationScan(ctx)
	})
	day := previousCalendarDayNoon(r.deps.Loc, r.now())
	r.enqueueDayRetry(PriorityCatchUp, "catchup_day", day, r.jobCatchUpDayFor)
	r.Enqueue(PriorityCatchUp, "catchup_month", r.jobCatchUpMonth)
	r.Enqueue(PriorityCatchUp, "catchup_year", r.jobCatchUpYear)
	select {
	// unguarded-shared-field: safe — wake is immutable after construction and channels are concurrency-safe
	case r.wake <- struct{}{}:
	default:
	}
}

// Enqueue adds a job (lower priority number runs earlier).
func (r *Runner) Enqueue(priority int, name string, run func(context.Context) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enqueueLocked(&jobItem{priority: priority, name: name, run: run})
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runner) enqueueDayRetry(priority int, name string, day time.Time, run func(context.Context, time.Time) error) {
	loc := r.deps.Loc
	if loc == nil {
		loc = time.UTC
	}
	dayStr := day.In(loc).Format("2006-01-02")
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enqueueLocked(&jobItem{
		priority: priority,
		name:     name,
		key:      "day:" + dayStr,
		delays:   []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, 60 * time.Minute},
		run: func(ctx context.Context) error {
			return run(ctx, day)
		},
	})
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runner) enqueueLocked(it *jobItem) {
	if r.queued == nil {
		r.queued = make(map[string]struct{})
	}
	if it.key != "" {
		if _, exists := r.queued[it.key]; exists {
			return
		}
		r.queued[it.key] = struct{}{}
	}
	r.seq++
	it.seq = r.seq
	heap.Push(&r.pq, it)
}

func isRetryableDayJobError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "invalid") || strings.Contains(s, "parse") || strings.Contains(s, "validation") {
		return false
	}
	return true
}

func retryDelay(it *jobItem) (time.Duration, bool) {
	if it == nil || len(it.delays) == 0 {
		return 0, false
	}
	if it.attempt >= len(it.delays) {
		return 0, false
	}
	return it.delays[it.attempt], true
}

func (r *Runner) scheduleRetry(it *jobItem, err error) bool {
	delay, ok := retryDelay(it)
	if !ok || !isRetryableDayJobError(err) {
		return false
	}
	nextAttempt := it.attempt + 1
	it.attempt = nextAttempt
	it.notBefore = r.now().Add(delay)
	r.deps.Logger.Warn("memory job retry scheduled",
		"job", it.name,
		"key", it.key,
		"attempt", nextAttempt,
		"delay", delay.String(),
	)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enqueueLocked(it)
	select {
	case r.wake <- struct{}{}:
	default:
	}
	return true
}

func (r *Runner) drain(ctx context.Context) {
	for {
		it := r.popNextJob()
		if it == nil {
			return
		}
		if r.deferUntilNotBefore(it) {
			return
		}
		if r.deferDuringUserTurn(it) {
			return
		}
		r.runOne(ctx, it)
	}
}

func (r *Runner) popNextJob() *jobItem {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pq.Len() == 0 {
		return nil
	}
	// type-assertion: safe — priorityQueue.Pop always returns its private *jobItem element type
	it := heap.Pop(&r.pq).(*jobItem)
	if it.key != "" {
		delete(r.queued, it.key)
	}
	return it
}

func (r *Runner) requeue(it *jobItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enqueueLocked(it)
}

func (r *Runner) deferUntilNotBefore(it *jobItem) bool {
	if it.notBefore.IsZero() || !r.now().Before(it.notBefore) {
		return false
	}
	r.requeue(it)
	return true
}

func (r *Runner) deferDuringUserTurn(it *jobItem) bool {
	if it.priority < PriorityCatchUp || !r.userTurnActive() {
		return false
	}
	// Silent push back without waking — avoids tight poll loop (REQ-02.015).
	// Job retries on the next scheduler tick (≤60 s) or next real enqueue.
	r.deps.Logger.Debug("memory job deferred during user turn", "job", it.name, "priority", it.priority)
	r.requeue(it)
	return true
}

func (r *Runner) runOne(ctx context.Context, it *jobItem) {
	timeout := time.Duration(jobTimeoutSeconds) * time.Second
	if timeout < 30*time.Second {
		timeout = 30 * time.Second
	}
	jctx, cancel := context.WithTimeout(ctx, timeout)
	started := time.Now()
	lifecyclelog.Info(r.deps.Logger, "memory_job", "job_start", 0, "lifecycle", "job", it.name)
	err := it.run(jctx)
	cancel()
	dur := time.Since(started)
	if err != nil {
		lifecyclelog.Error(r.deps.Logger, "memory_job", "job_complete", dur, err, "lifecycle", "job", it.name)
		r.deps.Logger.Error("memory job failed", "job", it.name, "error", err)
		if r.scheduleRetry(it, err) {
			return
		}
		if len(it.delays) > 0 && isRetryableDayJobError(err) && it.attempt >= len(it.delays) {
			r.deps.Logger.Error("memory job retry exhausted", "job", it.name, "key", it.key, "attempts", it.attempt)
		}
		return
	}
	lifecyclelog.Info(r.deps.Logger, "memory_job", "job_complete", dur, "lifecycle", "job", it.name)
}

func previousCalendarDayNoon(loc *time.Location, now time.Time) time.Time {
	y, m, d := patime.PreviousCalendarDate(loc, now)
	return patime.NoonOnCalendar(loc, y, m, d)
}

func (r *Runner) jobSummarizeDayFor(ctx context.Context, day time.Time) error {
	return r.runDayDirect(ctx, day)
}

func (r *Runner) jobCatchUpDayFor(ctx context.Context, day time.Time) error {
	loc := r.deps.Loc
	need, err := dayNeedsCatchUp(ctx, r.deps.Cfg, r.deps.Memory, loc, day)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}
	return r.runDayDirect(ctx, day)
}

func (r *Runner) jobCatchUpDay(ctx context.Context) error {
	return r.jobCatchUpDayFor(ctx, previousCalendarDayNoon(r.deps.Loc, r.now()))
}

func (r *Runner) jobCatchUpMonth(ctx context.Context) error {
	loc := r.deps.Loc
	py, pm := patime.PreviousMonth(loc, r.now())
	need, err := r.monthNeedsCatchUp(ctx, py, int(pm))
	if err != nil {
		return err
	}
	if !need {
		return nil
	}
	return r.runMonth(ctx, py, int(pm))
}

func (r *Runner) jobCatchUpYear(ctx context.Context) error {
	py := patime.PreviousYear(r.deps.Loc, r.now())
	need, err := r.yearNeedsCatchUp(ctx, py)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}
	return r.runYear(ctx, py)
}

func (r *Runner) monthNeedsCatchUp(ctx context.Context, year, month int) (bool, error) {
	sections, err := summarize.GatherDaySummariesForMonth(ctx, r.deps.Memory, year, month)
	if err != nil {
		return false, err
	}
	if len(sections) == 0 {
		return false, nil
	}
	prev, err := r.deps.Memory.ReadMonthSummary(ctx, year, month)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(prev) == "", nil
}

func (r *Runner) yearNeedsCatchUp(ctx context.Context, year int) (bool, error) {
	var anyMonth bool
	for m := 1; m <= 12; m++ {
		s, err := r.deps.Memory.ReadMonthSummary(ctx, year, m)
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(s) != "" {
			anyMonth = true
			break
		}
	}
	if !anyMonth {
		return false, nil
	}
	ys, err := r.deps.Memory.ReadYearSummary(ctx, year)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(ys) == "", nil
}

func dayNeedsCatchUp(ctx context.Context, cfg *config.Config, mem *memory.Store, loc *time.Location, day time.Time) (bool, error) {
	entries, err := llmlog.ReadEntriesForDay(cfg.Paths.LLMLogDir, day, loc)
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		return false, nil
	}
	s, err := mem.ReadDaySummary(ctx, day)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(s) == "", nil
}

func (r *Runner) runDayDirect(ctx context.Context, day time.Time) error {
	err := summarize.Day(ctx, day, summarize.DayConfig{
		LLMLogDir:   r.deps.Cfg.Paths.LLMLogDir,
		LLMProvider: r.deps.LLMProvider,
		MemoryStore: r.deps.Memory,
		Embedder:    r.deps.Embedder,
		VectorStore: r.deps.Vector,
		Logger:      r.deps.Logger,
		Loc:         r.deps.Loc,
	})
	if summarize.IsVectorIndexAfterFileWrite(err) {
		dateStr := day.In(r.deps.Loc).Format("2006-01-02")
		r.Enqueue(PriorityReconcile, "reconcile_day:"+dateStr, func(ctx context.Context) error {
			return ReindexDaySummary(ctx, r.deps.Memory, r.deps.Vector, r.deps.Embedder, r.deps.Loc, dateStr)
		})
		return nil
	}
	return err
}

func (r *Runner) runMonth(ctx context.Context, year, month int) error {
	err := summarize.Month(ctx, year, month, summarize.MonthConfig{
		LLMProvider: r.deps.LLMProvider,
		MemoryStore: r.deps.Memory,
		Embedder:    r.deps.Embedder,
		VectorStore: r.deps.Vector,
		Logger:      r.deps.Logger,
	})
	if summarize.IsVectorIndexAfterFileWrite(err) {
		id := summarize.VectorIDPrefixMonth + fmt.Sprintf("%04d-%02d", year, month)
		r.Enqueue(PriorityReconcile, "reconcile_month:"+id, func(ctx context.Context) error {
			return ReindexMonthSummary(ctx, r.deps.Memory, r.deps.Vector, r.deps.Embedder, year, month)
		})
		return nil
	}
	return err
}

func (r *Runner) runYear(ctx context.Context, year int) error {
	err := summarize.Year(ctx, year, summarize.YearConfig{
		LLMProvider: r.deps.LLMProvider,
		MemoryStore: r.deps.Memory,
		Embedder:    r.deps.Embedder,
		VectorStore: r.deps.Vector,
		Logger:      r.deps.Logger,
	})
	if summarize.IsVectorIndexAfterFileWrite(err) {
		r.Enqueue(PriorityReconcile, "reconcile_year:"+fmt.Sprintf("%04d", year), func(ctx context.Context) error {
			return ReindexYearSummary(ctx, r.deps.Memory, r.deps.Vector, r.deps.Embedder, year)
		})
		return nil
	}
	return err
}

func (r *Runner) runReconciliationScan(ctx context.Context) error {
	n := reconciliationScanDays
	if n < 1 {
		return nil
	}
	loc := r.deps.Loc
	now := r.now().In(loc)
	for i := 1; i <= n; i++ {
		t := now.AddDate(0, 0, -i)
		y, m, d := t.Date()
		day := time.Date(y, m, d, 12, 0, 0, 0, loc)
		dateStr := day.Format("2006-01-02")
		text, err := r.deps.Memory.ReadDaySummary(ctx, day)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		id := summarize.VectorIDPrefixDay + dateStr
		ok, err := r.deps.Vector.Exists(ctx, id)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		if err := ReindexDaySummary(ctx, r.deps.Memory, r.deps.Vector, r.deps.Embedder, loc, dateStr); err != nil {
			r.deps.Logger.Error("reconcile day", "day", dateStr, "error", err)
		}
	}
	return nil
}

// Stop cancels the runner context; wait on Done() for exit.
func (r *Runner) Stop() {
	r.stopOnce.Do(r.stop)
}

// Done returns a channel closed when the worker exits.
func (r *Runner) Done() <-chan struct{} { return r.done }
