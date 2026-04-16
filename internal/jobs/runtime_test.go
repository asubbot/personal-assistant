package jobs

import (
	"context"
	"testing"
	"time"
)

type runnerFunc func(ctx context.Context, job Job) error

func (f runnerFunc) Run(ctx context.Context, job Job) error { return f(ctx, job) }

func mustCreateJob(t *testing.T, st *Store, in JobInput) Job {
	t.Helper()
	j, err := st.CreateJob(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	return j
}

func waitForRun(t *testing.T, st *Store, jobID string, timeout time.Duration) *JobRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		last, err := st.GetLastRun(context.Background(), jobID)
		if err != nil {
			t.Fatalf("GetLastRun: %v", err)
		}
		if last != nil {
			return last
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout waiting for run")
	return nil
}

// Covers AC-19.003: scheduler computes next run using cron + timezone.
func TestComputeNextRun_UsesTimezone(t *testing.T) {
	job := Job{ScheduleExpr: "0 9 * * *", TimeZone: "Asia/Tokyo"}
	from := time.Date(2026, 4, 16, 8, 30, 0, 0, time.UTC)
	next, err := ComputeNextRun(job, from)
	if err != nil {
		t.Fatalf("ComputeNextRun: %v", err)
	}
	loc, _ := time.LoadLocation("Asia/Tokyo")
	local := next.In(loc)
	if local.Hour() != 9 || local.Minute() != 0 {
		t.Fatalf("next local = %s, want 09:00", local)
	}
}

// Covers AC-19.005: due trigger creates one run record.
// Covers AC-19.004: EvaluateDue updates next_run_at.
func TestRuntime_EvaluateDue_CreatesRunAndUpdatesNextRun(t *testing.T) {
	st := openTestStore(t)
	j := mustCreateJob(t, st, JobInput{
		Name:           "due-job",
		ScheduleExpr:   "* * * * *",
		TimeZone:       "UTC",
		Instruction:    "do",
		DeliveryChatID: 1,
		Status:         StatusActive,
		OverlapPolicy:  OverlapSingleInstance,
		TimeoutPolicy:  TimeoutCancelAfter,
	})
	now := time.Now().UTC().Round(0)
	if err := st.SetJobNextRun(context.Background(), j.ID, ptrTime(now.Add(-time.Second))); err != nil {
		t.Fatalf("SetJobNextRun: %v", err)
	}
	rt := NewRuntime(st, runnerFunc(func(ctx context.Context, job Job) error { return nil }), RuntimeConfig{
		RunTimeout: 100 * time.Millisecond,
		Now:        func() time.Time { return now },
	})
	if err := rt.EvaluateDue(context.Background()); err != nil {
		t.Fatalf("EvaluateDue: %v", err)
	}
	last := waitForRun(t, st, j.ID, time.Second)
	if last.Outcome != "success" {
		t.Fatalf("last outcome = %q, want success", last.Outcome)
	}
	got, err := st.GetJob(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.NextRunAt == nil || !got.NextRunAt.After(now) {
		t.Fatalf("NextRunAt = %v, want > now", got.NextRunAt)
	}
}

// Covers AC-19.010: overlap policy single_instance records skipped overlap run.
func TestRuntime_EvaluateDue_OverlapSingleInstanceSkips(t *testing.T) {
	st := openTestStore(t)
	j := mustCreateJob(t, st, JobInput{
		Name:           "overlap-job",
		ScheduleExpr:   "* * * * *",
		TimeZone:       "UTC",
		Instruction:    "do",
		DeliveryChatID: 1,
		Status:         StatusActive,
		OverlapPolicy:  OverlapSingleInstance,
		TimeoutPolicy:  TimeoutCancelAfter,
	})
	now := time.Now().UTC().Round(0)
	if err := st.SetJobNextRun(context.Background(), j.ID, ptrTime(now.Add(-time.Second))); err != nil {
		t.Fatalf("SetJobNextRun: %v", err)
	}
	block := make(chan struct{})
	rt := NewRuntime(st, runnerFunc(func(ctx context.Context, job Job) error {
		<-block
		return nil
	}), RuntimeConfig{
		RunTimeout: 2 * time.Second,
		Now:        func() time.Time { return now },
	})
	if err := rt.EvaluateDue(context.Background()); err != nil {
		t.Fatalf("EvaluateDue(1): %v", err)
	}
	// Force second due while first run is still active.
	if err := st.SetJobNextRun(context.Background(), j.ID, ptrTime(now.Add(-time.Second))); err != nil {
		t.Fatalf("SetJobNextRun(2): %v", err)
	}
	if err := rt.EvaluateDue(context.Background()); err != nil {
		t.Fatalf("EvaluateDue(2): %v", err)
	}
	close(block)
	time.Sleep(100 * time.Millisecond)
	runs, err := st.ListRuns(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) < 2 {
		t.Fatalf("runs len = %d, want >=2", len(runs))
	}
	foundSkip := false
	for _, r := range runs {
		if r.Outcome == "skipped_overlap" {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Fatalf("runs = %+v, expected skipped_overlap", runs)
	}
}

// Covers AC-19.009: timeout policy enforcement marks run with timeout reason.
func TestRuntime_EvaluateDue_TimeoutPolicy(t *testing.T) {
	st := openTestStore(t)
	j := mustCreateJob(t, st, JobInput{
		Name:           "timeout-job",
		ScheduleExpr:   "* * * * *",
		TimeZone:       "UTC",
		Instruction:    "do",
		DeliveryChatID: 1,
		Status:         StatusActive,
		OverlapPolicy:  OverlapSingleInstance,
		TimeoutPolicy:  TimeoutCancelAfter,
	})
	now := time.Now().UTC().Round(0)
	if err := st.SetJobNextRun(context.Background(), j.ID, ptrTime(now.Add(-time.Second))); err != nil {
		t.Fatalf("SetJobNextRun: %v", err)
	}
	rt := NewRuntime(st, runnerFunc(func(ctx context.Context, job Job) error {
		<-ctx.Done()
		return ctx.Err()
	}), RuntimeConfig{
		RunTimeout: 20 * time.Millisecond,
		Now:        func() time.Time { return time.Now().UTC() },
	})
	if err := rt.EvaluateDue(context.Background()); err != nil {
		t.Fatalf("EvaluateDue: %v", err)
	}
	last := waitForRun(t, st, j.ID, time.Second)
	if last.Outcome != "failure" || last.FailureReasonClass != "timeout" {
		t.Fatalf("last = %+v, want failure/timeout", *last)
	}
}

// Covers AC-19.014: run-now creates a run_now trigger run.
func TestRuntime_RunNow_RecordsRunNowTrigger(t *testing.T) {
	st := openTestStore(t)
	j := mustCreateJob(t, st, JobInput{
		Name:           "manual-run",
		ScheduleExpr:   "* * * * *",
		TimeZone:       "UTC",
		Instruction:    "do",
		DeliveryChatID: 1,
		Status:         StatusActive,
		OverlapPolicy:  OverlapSingleInstance,
		TimeoutPolicy:  TimeoutCancelAfter,
	})
	rt := NewRuntime(st, runnerFunc(func(ctx context.Context, job Job) error { return nil }), RuntimeConfig{
		RunTimeout: 100 * time.Millisecond,
		Now:        func() time.Time { return time.Now().UTC() },
	})
	if err := rt.RunNow(context.Background(), j.ID); err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	last := waitForRun(t, st, j.ID, time.Second)
	if last.TriggerType != "run_now" {
		t.Fatalf("last.TriggerType = %q, want run_now", last.TriggerType)
	}
}

// Covers AC-19.009 (Trace: REQ-19.009): NewRuntime applies a positive default RunTimeout when omitted so job runs use a bounded execution window.
func TestNewRuntime_DefaultRunTimeoutFiveMinutes(t *testing.T) {
	st := openTestStore(t)
	for _, timeout := range []time.Duration{0, -1 * time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			rt := NewRuntime(st, runnerFunc(func(ctx context.Context, job Job) error { return nil }), RuntimeConfig{
				RunTimeout: timeout,
				Now:        func() time.Time { return time.Now().UTC() },
			})
			if rt.cfg.RunTimeout != 5*time.Minute {
				t.Fatalf("RunTimeout = %v, want 5m", rt.cfg.RunTimeout)
			}
		})
	}
}
