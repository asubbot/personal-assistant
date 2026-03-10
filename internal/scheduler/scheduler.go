package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"pa/internal/tools"
)

const actionNotify = "notify"

// Task is one scheduled task from the JSON file (REQ-009).
// Name must be unique across all tasks in the file.
type Task struct {
	Name     string         `json:"name"`     // unique task identifier (required)
	Schedule string         `json:"schedule"` // cron e.g. "0 9 * * *" or "@every 1h"
	Action   string         `json:"action"`   // tool name or "notify"
	Params   map[string]any `json:"params"`
}

// Notifier sends a notification (e.g. Telegram). Optional; when action is "notify", scheduler calls it.
type Notifier interface {
	SendMessage(ctx context.Context, text string) error
}

// Config holds scheduler dependencies.
type Config struct {
	Registry *tools.Registry
	Notifier Notifier
	Logger   *slog.Logger
}

// cronInterface is the minimal interface for the cron runner (Start/Stop).
type cronInterface interface {
	Start()
	Stop() context.Context
}

// Scheduler runs tasks at schedule by invoking tools or notifier (AC-020, AC-021).
type Scheduler struct {
	cfg  Config
	cron cronInterface
}

// LoadTasks reads the scheduled tasks JSON file. Returns nil slice and nil error if path is empty or file missing.
func LoadTasks(path string) ([]Task, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scheduler: read tasks file: %w", err)
	}
	var list []Task
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("scheduler: parse tasks JSON: %w", err)
	}
	seen := make(map[string]struct{})
	for i := range list {
		name := list[i].Name
		if name == "" {
			return nil, fmt.Errorf("scheduler: task at index %d has empty name", i)
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("scheduler: duplicate task name %q", name)
		}
		seen[name] = struct{}{}
	}
	return list, nil
}

// New builds a scheduler and adds cron entries for each task. Call Start() to run, Stop() on ctx done.
func New(tasks []Task, cfg Config) (*Scheduler, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	s := &Scheduler{cfg: cfg}
	runner, err := newCronRunner(tasks, cfg, s.executeTask)
	if err != nil {
		return nil, err
	}
	s.cron = runner
	return s, nil
}

// executeTask runs one task: resolve action to tool or notify, validate params, run (AC-021).
func (s *Scheduler) executeTask(ctx context.Context, task Task) {
	if task.Action == "" {
		s.cfg.Logger.Warn("scheduler: task has empty action", "task", task.Name, "schedule", task.Schedule)
		return
	}
	attrs := []any{"task", task.Name, "schedule", task.Schedule}
	if task.Action == actionNotify {
		if s.cfg.Notifier == nil {
			s.cfg.Logger.Warn("scheduler: notify action but no notifier configured", attrs...)
			return
		}
		msg := ""
		if task.Params != nil {
			if m, ok := task.Params["message"].(string); ok {
				msg = m
			}
		}
		if err := s.cfg.Notifier.SendMessage(ctx, msg); err != nil {
			s.cfg.Logger.Error("scheduler: notify failed", append(attrs, "error", err)...)
		}
		return
	}
	attrs = append(attrs, "action", task.Action)
	tool, ok := s.cfg.Registry.Get(task.Action)
	if !ok {
		s.cfg.Logger.Warn("scheduler: unknown action", attrs...)
		return
	}
	if err := tools.ValidateParams(tool.ParamsSchema(), task.Params); err != nil {
		s.cfg.Logger.Warn("scheduler: invalid params", append(attrs, "error", err)...)
		return
	}
	if task.Params != nil {
		attrs = append(attrs, "params", task.Params)
	}
	result, err := tool.Run(ctx, task.Params)
	if err != nil {
		s.cfg.Logger.Error("scheduler: tool run failed", append(attrs, "error", err)...)
		return
	}
	s.cfg.Logger.Info("scheduler: task completed", append(attrs, "result_len", len(result))...)
}

// Start starts the cron scheduler (non-blocking).
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop stops the cron scheduler and returns a context that is cancelled when stopped.
func (s *Scheduler) Stop() context.Context {
	return s.cron.Stop()
}
