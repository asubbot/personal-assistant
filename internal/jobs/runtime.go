package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	OverlapSingleInstance = "single_instance"
	TimeoutCancelAfter    = "cancel_after_limit"
)

type Runner interface {
	Run(ctx context.Context, job Job) error
}

type RuntimeConfig struct {
	RunTimeout time.Duration
	Now        func() time.Time
	Logger     *slog.Logger
}

type Runtime struct {
	store  *Store
	runner Runner
	cfg    RuntimeConfig

	mu     sync.Mutex
	active map[string]struct{}
}

func NewRuntime(store *Store, runner Runner, cfg RuntimeConfig) *Runtime {
	if cfg.RunTimeout <= 0 {
		cfg.RunTimeout = 5 * time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Runtime{
		store:  store,
		runner: runner,
		cfg:    cfg,
		active: make(map[string]struct{}),
	}
}

// ComputeNextRun returns the next trigger timestamp for a job after "from".
func ComputeNextRun(job Job, from time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(job.TimeZone)
	if err != nil {
		return time.Time{}, fmt.Errorf("jobs runtime: invalid timezone %q: %w", job.TimeZone, err)
	}
	sched, err := cron.ParseStandard(job.ScheduleExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("jobs runtime: invalid cron expr %q: %w", job.ScheduleExpr, err)
	}
	next := sched.Next(from.In(loc))
	return next.UTC(), nil
}

// EvaluateDue loads active jobs, computes due state, and triggers asynchronous execution.
func (r *Runtime) EvaluateDue(ctx context.Context) error {
	now := r.cfg.Now().UTC()
	jobs, err := r.store.ListJobs(ctx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := r.evaluateJob(ctx, job, now); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) evaluateJob(ctx context.Context, job Job, now time.Time) error {
	if job.Status != StatusActive {
		return nil
	}
	if job.NextRunAt == nil {
		return r.updateNextRun(ctx, job, now)
	}
	if job.NextRunAt.After(now) {
		return nil
	}
	if err := r.updateNextRun(ctx, job, now); err != nil {
		return err
	}
	if r.isActive(job.ID) && job.OverlapPolicy == OverlapSingleInstance {
		return r.recordOverlapSkip(ctx, job.ID, now)
	}
	r.markActive(job.ID, true)
	go r.runJob(ctx, job, "schedule")
	return nil
}

// RunNow triggers immediate execution for a single job.
func (r *Runtime) RunNow(ctx context.Context, jobID string) error {
	job, err := r.store.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	now := r.cfg.Now().UTC()
	if r.isActive(job.ID) && job.OverlapPolicy == OverlapSingleInstance {
		return r.recordOverlapSkip(ctx, job.ID, now)
	}
	r.markActive(job.ID, true)
	go r.runJob(ctx, job, "run_now")
	return nil
}

func (r *Runtime) updateNextRun(ctx context.Context, job Job, now time.Time) error {
	next, err := ComputeNextRun(job, now)
	if err != nil {
		return err
	}
	return r.store.SetJobNextRun(ctx, job.ID, &next)
}

func (r *Runtime) recordOverlapSkip(ctx context.Context, jobID string, now time.Time) error {
	_, err := r.store.RecordRun(ctx, JobRunInput{
		JobID:              jobID,
		TriggerType:        "schedule",
		StartedAt:          now,
		FinishedAt:         ptrTime(now),
		Outcome:            "skipped_overlap",
		FailureReasonClass: "overlap",
	})
	if err != nil {
		return err
	}
	if err := r.store.SetJobLastRunStatus(ctx, jobID, "skipped_overlap"); err != nil {
		return err
	}
	r.cfg.Logger.Info(
		"jobs audit",
		"actor_user_id", 0,
		"job_id", jobID,
		"operation", "run_lifecycle",
		"outcome", "skipped_overlap",
	)
	return nil
}

func (r *Runtime) runJob(parent context.Context, job Job, triggerType string) {
	started := r.cfg.Now().UTC()
	defer r.markActive(job.ID, false)
	r.cfg.Logger.Info(
		"jobs audit",
		"actor_user_id", 0,
		"job_id", job.ID,
		"operation", "run_lifecycle",
		"outcome", "started",
		"trigger_type", triggerType,
	)

	persistCtx := context.WithoutCancel(parent)
	ctx := persistCtx
	var cancel context.CancelFunc
	if job.TimeoutPolicy == TimeoutCancelAfter {
		ctx, cancel = context.WithTimeout(ctx, r.cfg.RunTimeout)
		defer cancel()
	}

	outcome := "success"
	reason := ""
	if r.runner != nil {
		if err := r.runner.Run(ctx, job); err != nil {
			outcome = "failure"
			if ctx.Err() == context.DeadlineExceeded {
				reason = "timeout"
			} else {
				reason = "execution_error"
			}
			r.cfg.Logger.Warn("scheduled job run failed", "job_id", job.ID, "error", err, "reason", reason)
		}
	}
	finished := r.cfg.Now().UTC()
	_, err := r.store.RecordRun(persistCtx, JobRunInput{
		JobID:              job.ID,
		TriggerType:        triggerType,
		StartedAt:          started,
		FinishedAt:         &finished,
		Outcome:            outcome,
		FailureReasonClass: reason,
	})
	if err != nil {
		r.cfg.Logger.Error("record scheduled run", "job_id", job.ID, "error", err)
		return
	}
	if err := r.store.SetJobLastRunStatus(persistCtx, job.ID, outcome); err != nil {
		r.cfg.Logger.Error("set last run status", "job_id", job.ID, "error", err)
		return
	}
	r.cfg.Logger.Info(
		"jobs audit",
		"actor_user_id", 0,
		"job_id", job.ID,
		"operation", "run_lifecycle",
		"outcome", outcome,
		"failure_reason_class", reason,
		"trigger_type", triggerType,
	)
}

func (r *Runtime) isActive(jobID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.active[jobID]
	return ok
}

func (r *Runtime) markActive(jobID string, v bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v {
		r.active[jobID] = struct{}{}
		return
	}
	delete(r.active, jobID)
}

func ptrTime(t time.Time) *time.Time { return &t }
