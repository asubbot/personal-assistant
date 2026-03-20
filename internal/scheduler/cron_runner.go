package scheduler

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// cronRunner wraps robfig/cron and runs executeTask for each task.
type cronRunner struct {
	cron    *cron.Cron
	execute func(ctx context.Context, task Task)
	ctx     context.Context
	logger  *slog.Logger
}

func newCronRunner(tasks []Task, cfg Config, execute func(context.Context, Task)) (*cronRunner, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	r := &cronRunner{
		execute: execute,
		ctx:     context.Background(),
		logger:  cfg.Logger,
		cron:    cron.New(),
	}
	for i, task := range tasks {
		if task.Schedule == "" {
			cfg.Logger.Warn("scheduler: task has empty schedule", "index", i)
			continue
		}
		_, err := r.cron.AddFunc(task.Schedule, func() {
			r.execute(r.ctx, task)
		})
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *cronRunner) Start() {
	r.cron.Start()
}

func (r *cronRunner) Stop() context.Context {
	return r.cron.Stop()
}
