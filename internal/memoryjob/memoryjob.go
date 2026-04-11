// Package memoryjob runs automatic summarization, catch-up, and vector reconciliation (EP-002).
package memoryjob

import (
	"container/heap"
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/embedding"
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
	summarizeHour          = 1
	summarizeMinute        = 0
	jobTimeoutSeconds      = 1800
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
	priority int
	seq      int64
	name     string
	run      func(context.Context) error
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
		deps: deps,
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: cancel,
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
	r.Enqueue(PriorityScheduled, "summarize_yesterday", r.jobSummarizeYesterday)
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
	r.Enqueue(PriorityCatchUp, "catchup_day", r.jobCatchUpDay)
	r.Enqueue(PriorityCatchUp, "catchup_month", r.jobCatchUpMonth)
	r.Enqueue(PriorityCatchUp, "catchup_year", r.jobCatchUpYear)
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// Enqueue adds a job (lower priority number runs earlier).
func (r *Runner) Enqueue(priority int, name string, run func(context.Context) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	heap.Push(&r.pq, &jobItem{priority: priority, seq: r.seq, name: name, run: run})
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runner) drain(ctx context.Context) {
	for {
		r.mu.Lock()
		if r.pq.Len() == 0 {
			r.mu.Unlock()
			return
		}
		it := heap.Pop(&r.pq).(*jobItem)
		r.mu.Unlock()

		if it.priority >= PriorityCatchUp && r.userTurnActive() {
			// Re-queue and back off (REQ-02.015).
			if lg := r.deps.Logger; lg != nil {
				lg.Debug("memory job requeued during user turn", "job", it.name, "priority", it.priority, "reason", "user_turn")
			}
			time.Sleep(200 * time.Millisecond)
			r.Enqueue(it.priority, it.name, it.run)
			return
		}

		timeout := time.Duration(jobTimeoutSeconds) * time.Second
		if timeout < 30*time.Second {
			timeout = 30 * time.Second
		}
		jctx, cancel := context.WithTimeout(ctx, timeout)
		err := it.run(jctx)
		cancel()
		if err != nil {
			r.deps.Logger.Error("memory job failed", "job", it.name, "error", err)
		}
	}
}

func (r *Runner) jobSummarizeYesterday(ctx context.Context) error {
	loc := r.deps.Loc
	y, m, d := patime.PreviousCalendarDate(loc, r.now())
	day := patime.NoonOnCalendar(loc, y, m, d)
	return r.runDayDirect(ctx, day)
}

func (r *Runner) jobCatchUpDay(ctx context.Context) error {
	loc := r.deps.Loc
	y, m, d := patime.PreviousCalendarDate(loc, r.now())
	day := patime.NoonOnCalendar(loc, y, m, d)
	if !dayNeedsCatchUp(ctx, r.deps.Cfg, r.deps.Memory, loc, day) {
		return nil
	}
	return r.runDayDirect(ctx, day)
}

func (r *Runner) jobCatchUpMonth(ctx context.Context) error {
	loc := r.deps.Loc
	py, pm := patime.PreviousMonth(loc, r.now())
	if !r.monthNeedsCatchUp(ctx, py, int(pm)) {
		return nil
	}
	return r.runMonth(ctx, py, int(pm))
}

func (r *Runner) jobCatchUpYear(ctx context.Context) error {
	py := patime.PreviousYear(r.deps.Loc, r.now())
	if !r.yearNeedsCatchUp(ctx, py) {
		return nil
	}
	return r.runYear(ctx, py)
}

func (r *Runner) monthNeedsCatchUp(ctx context.Context, year, month int) bool {
	sections, err := summarize.GatherDaySummariesForMonth(ctx, r.deps.Memory, year, month)
	if err != nil || len(sections) == 0 {
		return false
	}
	prev, err := r.deps.Memory.ReadMonthSummary(ctx, year, month)
	if err != nil {
		r.deps.Logger.Error("read month summary", "error", err)
		return false
	}
	return strings.TrimSpace(prev) == ""
}

func (r *Runner) yearNeedsCatchUp(ctx context.Context, year int) bool {
	var anyMonth bool
	for m := 1; m <= 12; m++ {
		s, err := r.deps.Memory.ReadMonthSummary(ctx, year, m)
		if err != nil {
			return false
		}
		if strings.TrimSpace(s) != "" {
			anyMonth = true
			break
		}
	}
	if !anyMonth {
		return false
	}
	ys, err := r.deps.Memory.ReadYearSummary(ctx, year)
	if err != nil {
		return false
	}
	return strings.TrimSpace(ys) == ""
}

func dayNeedsCatchUp(ctx context.Context, cfg *config.Config, mem *memory.Store, loc *time.Location, day time.Time) bool {
	entries, err := llmlog.ReadEntriesForDay(cfg.Paths.LLMLogDir, day, loc)
	if err != nil || len(entries) == 0 {
		return false
	}
	s, err := mem.ReadDaySummary(ctx, day)
	if err != nil {
		return false
	}
	return strings.TrimSpace(s) == ""
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
