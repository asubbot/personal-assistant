package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type RuntimeAPI interface {
	RunNow(ctx context.Context, jobID string) error
}

// Manager serves Telegram management commands for scheduled jobs.
type Manager struct {
	store           *Store
	runtime         RuntimeAPI
	logger          *slog.Logger
	defaultTimeZone string
	now             func() time.Time
}

func NewManager(store *Store, runtime RuntimeAPI, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	mgr := &Manager{
		store:           store,
		runtime:         runtime,
		logger:          logger,
		defaultTimeZone: "UTC",
		now:             func() time.Time { return time.Now().UTC() },
	}
	return mgr
}

func (m *Manager) SetDefaultTimeZone(tz string) {
	if m == nil {
		return
	}
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return
	}
	m.defaultTimeZone = tz
}

// HandleCommand accepts `/jobs ...` commands.
func (m *Manager) HandleCommand(ctx context.Context, userID int64, text string) (string, bool, error) {
	cmd, args, ok := parseJobsCommand(text)
	if !ok {
		return "", false, nil
	}
	if cmd == "" {
		m.audit(userID, "", "help", "success")
		return m.helpText(), true, nil
	}
	spec, found := m.commandSpecs()[cmd]
	if !found {
		m.audit(userID, "", "unknown", "invalid_command")
		return fmt.Sprintf("Unknown /jobs command %q.\n%s", cmd, m.helpText()), true, nil
	}
	if len(args) < spec.minArgs {
		m.audit(userID, "", cmd, "invalid_args")
		return spec.usage, true, nil
	}
	return spec.run(ctx, userID, args)
}

func (m *Manager) CreateScheduledJobFromSpec(
	ctx context.Context,
	userID int64,
	deliveryChatID int64,
	instruction string,
	hour int,
	minute int,
	timezone string,
	creationPath string,
) (string, Job, error) {
	creationPath = strings.TrimSpace(creationPath)
	if creationPath == "" {
		creationPath = "deterministic_parser"
	}
	instruction = strings.TrimSpace(instruction)
	if reply, ok := m.validateCreateSpec(userID, instruction, hour, minute, creationPath); !ok {
		// empty-struct-return: safe — soft validation: reply text, no Job, nil err (AC-21.005)
		return reply, Job{}, nil
	}

	now := m.now().UTC()
	tz, target := m.resolveCreateDefaults(timezone, deliveryChatID, userID)
	scheduleExpr := fmt.Sprintf("%d %d * * *", minute, hour)
	created, next, err := m.persistCreateSpec(ctx, userID, instruction, hour, minute, tz, target, now, creationPath)
	if err != nil {
		return "", Job{}, err
	}

	m.audit(userID, created.ID, "create_nl", "success", "creation_path", creationPath, "parsed_schedule", scheduleExpr)
	reply := fmt.Sprintf(
		"Scheduled job created.\njob_id: %s\nschedule: %s\ntimezone: %s\nnext_run: %s\ninstruction: %s",
		created.ID, scheduleExpr, tz, next.UTC().Format(time.RFC3339), instruction,
	)
	return reply, created, nil
}

func (m *Manager) validateCreateSpec(userID int64, instruction string, hour int, minute int, creationPath string) (string, bool) {
	if instruction == "" {
		m.audit(userID, "", "create_nl", "invalid_instruction", "creation_path", creationPath)
		return "Instruction must be non-empty.", false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		m.audit(userID, "", "create_nl", "invalid_time", "creation_path", creationPath)
		return "Invalid schedule format. Use: <instruction> and send it at HH:MM every day", false
	}
	return "", true
}

func (m *Manager) resolveCreateDefaults(timezone string, deliveryChatID int64, userID int64) (string, int64) {
	tz := strings.TrimSpace(timezone)
	if tz == "" {
		tz = m.defaultTimeZone
	}
	if strings.TrimSpace(tz) == "" {
		tz = "UTC"
	}
	target := deliveryChatID
	if target == 0 {
		target = userID
	}
	return tz, target
}

func (m *Manager) persistCreateSpec(
	ctx context.Context,
	userID int64,
	instruction string,
	hour int,
	minute int,
	timezone string,
	target int64,
	now time.Time,
	creationPath string,
) (Job, time.Time, error) {
	name := fmt.Sprintf("nl-%d-%02d%02d", now.Unix(), hour, minute)
	scheduleExpr := fmt.Sprintf("%d %d * * *", minute, hour)
	next, err := ComputeNextRun(Job{
		Name:         name,
		ScheduleExpr: scheduleExpr,
		TimeZone:     timezone,
	}, now)
	if err != nil {
		m.audit(userID, "", "create_nl", "internal_error", "creation_path", creationPath)
		return Job{}, time.Time{}, err
	}
	created, err := m.store.CreateJob(ctx, JobInput{
		Name:           name,
		ScheduleExpr:   scheduleExpr,
		TimeZone:       timezone,
		Instruction:    instruction,
		DeliveryChatID: target,
		Status:         StatusActive,
		OverlapPolicy:  OverlapSingleInstance,
		TimeoutPolicy:  TimeoutCancelAfter,
		NextRunAt:      &next,
	})
	if err != nil {
		m.audit(userID, "", "create_nl", "internal_error", "creation_path", creationPath)
		return Job{}, time.Time{}, err
	}
	return created, next, nil
}

type jobsCommandSpec struct {
	minArgs int
	usage   string
	run     func(ctx context.Context, userID int64, args []string) (string, bool, error)
}

// multi-write-no-transaction: safe — the map registers mutually exclusive command callbacks
func (m *Manager) commandSpecs() map[string]jobsCommandSpec {
	return map[string]jobsCommandSpec{
		"list": {
			minArgs: 0,
			usage:   "Usage: /jobs list",
			run: func(ctx context.Context, userID int64, _ []string) (string, bool, error) {
				return m.list(ctx, userID)
			},
		},
		"show": {
			minArgs: 1,
			usage:   "Usage: /jobs show <job_id>",
			run: func(ctx context.Context, userID int64, args []string) (string, bool, error) {
				return m.show(ctx, userID, args[0])
			},
		},
		"pause": {
			minArgs: 1,
			usage:   "Usage: /jobs pause <job_id>",
			run: func(ctx context.Context, userID int64, args []string) (string, bool, error) {
				return m.pause(ctx, userID, args[0])
			},
		},
		"resume": {
			minArgs: 1,
			usage:   "Usage: /jobs resume <job_id>",
			run: func(ctx context.Context, userID int64, args []string) (string, bool, error) {
				return m.resume(ctx, userID, args[0])
			},
		},
		"run-now": {
			minArgs: 1,
			usage:   "Usage: /jobs run-now <job_id>",
			run: func(ctx context.Context, userID int64, args []string) (string, bool, error) {
				return m.runNow(ctx, userID, args[0])
			},
		},
		"delete": {
			minArgs: 1,
			usage:   "Usage: /jobs delete <job_id>",
			run: func(ctx context.Context, userID int64, args []string) (string, bool, error) {
				return m.beginDelete(ctx, userID, args[0])
			},
		},
		"confirm-delete": {
			minArgs: 2,
			usage:   "Usage: /jobs confirm-delete <job_id> <token>",
			run: func(ctx context.Context, userID int64, args []string) (string, bool, error) {
				return m.confirmDelete(ctx, userID, args[0], args[1])
			},
		},
	}
}

func parseJobsCommand(text string) (command string, args []string, ok bool) {
	trimmed := strings.TrimSpace(text)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 || !isJobsCommandToken(fields[0]) {
		return "", nil, false
	}
	if len(fields) <= 1 {
		return "", nil, true
	}
	return strings.ToLower(fields[1]), fields[2:], true
}

func isJobsCommandToken(token string) bool {
	lower := strings.ToLower(strings.TrimSpace(token))
	if lower == "/jobs" {
		return true
	}
	return strings.HasPrefix(lower, "/jobs@")
}

func (m *Manager) list(ctx context.Context, userID int64) (string, bool, error) {
	items, err := m.store.ListJobs(ctx)
	if err != nil {
		m.audit(userID, "", "list", "internal_error")
		return "", true, err
	}
	if len(items) == 0 {
		m.audit(userID, "", "list", "success")
		return "No scheduled jobs configured.", true, nil
	}
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, "Scheduled jobs:")
	for _, j := range items {
		next := "-"
		if j.NextRunAt != nil {
			next = j.NextRunAt.UTC().Format(time.RFC3339)
		}
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | %s | next=%s", j.ID, j.ScheduleExpr, j.TimeZone, j.Status, next))
	}
	m.audit(userID, "", "list", "success")
	return strings.Join(lines, "\n"), true, nil
}

func (m *Manager) show(ctx context.Context, userID int64, jobID string) (string, bool, error) {
	j, err := m.store.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			m.audit(userID, jobID, "show", "not_found")
			return fmt.Sprintf("Job %q not found.", jobID), true, nil
		}
		m.audit(userID, jobID, "show", "internal_error")
		return "", true, err
	}
	last, err := m.store.GetLastRun(ctx, jobID)
	if err != nil {
		m.audit(userID, jobID, "show", "internal_error")
		return "", true, err
	}
	lastLine := "none"
	if last != nil {
		lastLine = fmt.Sprintf("%s/%s (%s)", last.TriggerType, last.Outcome, last.StartedAt.UTC().Format(time.RFC3339))
	}
	next := "-"
	if j.NextRunAt != nil {
		next = j.NextRunAt.UTC().Format(time.RFC3339)
	}
	reply := fmt.Sprintf(
		"Job %s\nname: %s\nschedule: %s\ntimezone: %s\nstatus: %s\ndelivery_chat_id: %d\nnext_run: %s\nlast_run: %s\ninstruction: %s",
		j.ID, j.Name, j.ScheduleExpr, j.TimeZone, j.Status, j.DeliveryChatID, next, lastLine, j.Instruction,
	)
	m.audit(userID, jobID, "show", "success")
	return reply, true, nil
}

func (m *Manager) pause(ctx context.Context, userID int64, jobID string) (string, bool, error) {
	if err := m.store.SetJobStatus(ctx, jobID, StatusPaused); err != nil {
		if errors.Is(err, ErrNotFound) {
			m.audit(userID, jobID, "pause", "not_found")
			return fmt.Sprintf("Job %q not found.", jobID), true, nil
		}
		m.audit(userID, jobID, "pause", "internal_error")
		return "", true, err
	}
	m.audit(userID, jobID, "pause", "success")
	return fmt.Sprintf("Job %s paused.", jobID), true, nil
}

func (m *Manager) resume(ctx context.Context, userID int64, jobID string) (string, bool, error) {
	if err := m.store.SetJobStatus(ctx, jobID, StatusActive); err != nil {
		if errors.Is(err, ErrNotFound) {
			m.audit(userID, jobID, "resume", "not_found")
			return fmt.Sprintf("Job %q not found.", jobID), true, nil
		}
		m.audit(userID, jobID, "resume", "internal_error")
		return "", true, err
	}
	m.audit(userID, jobID, "resume", "success")
	return fmt.Sprintf("Job %s resumed.", jobID), true, nil
}

func (m *Manager) runNow(ctx context.Context, userID int64, jobID string) (string, bool, error) {
	if m.runtime == nil {
		m.audit(userID, jobID, "run_now", "runtime_unavailable")
		return "", true, fmt.Errorf("jobs manager: runtime is not configured")
	}
	if err := m.runtime.RunNow(ctx, jobID); err != nil {
		if errors.Is(err, ErrNotFound) {
			m.audit(userID, jobID, "run_now", "not_found")
			return fmt.Sprintf("Job %q not found.", jobID), true, nil
		}
		m.audit(userID, jobID, "run_now", "internal_error")
		return "", true, err
	}
	m.audit(userID, jobID, "run_now", "success")
	return fmt.Sprintf("Job %s started.", jobID), true, nil
}

func (m *Manager) beginDelete(ctx context.Context, userID int64, jobID string) (string, bool, error) {
	if _, err := m.store.GetJob(ctx, jobID); err != nil {
		if errors.Is(err, ErrNotFound) {
			m.audit(userID, jobID, "delete_begin", "not_found")
			return fmt.Sprintf("Job %q not found.", jobID), true, nil
		}
		m.audit(userID, jobID, "delete_begin", "internal_error")
		return "", true, err
	}
	ch, err := m.store.CreateDeleteChallenge(ctx, jobID, userID, 5*time.Minute)
	if err != nil {
		m.audit(userID, jobID, "delete_begin", "internal_error")
		return "", true, err
	}
	m.audit(userID, jobID, "delete_begin", "challenge_issued")
	reply := fmt.Sprintf("Delete challenge created for %s.\nToken: %s\nConfirm with: /jobs confirm-delete %s %s",
		jobID, ch.Token, jobID, ch.Token)
	return reply, true, nil
}

func (m *Manager) confirmDelete(ctx context.Context, userID int64, jobID, token string) (string, bool, error) {
	ch, err := m.store.GetDeleteChallenge(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			m.audit(userID, jobID, "delete_confirm", "token_not_found")
			return "Invalid delete token.", true, nil
		}
		m.audit(userID, jobID, "delete_confirm", "internal_error")
		return "", true, err
	}
	if ch.JobID != jobID {
		m.audit(userID, jobID, "delete_confirm", "job_mismatch")
		return "Delete token does not match the provided job id.", true, nil
	}
	if ch.RequestedByUserID != userID {
		m.audit(userID, jobID, "delete_confirm", "actor_mismatch")
		return "Delete token belongs to another operator.", true, nil
	}
	if ch.ExpiresAt.Before(m.now()) {
		m.audit(userID, jobID, "delete_confirm", "token_expired")
		return "Delete token has expired.", true, nil
	}
	if err := m.store.DeleteJob(ctx, jobID); err != nil {
		if errors.Is(err, ErrNotFound) {
			m.audit(userID, jobID, "delete_confirm", "not_found")
			return fmt.Sprintf("Job %q not found.", jobID), true, nil
		}
		m.audit(userID, jobID, "delete_confirm", "internal_error")
		return "", true, err
	}
	m.audit(userID, jobID, "delete_confirm", "success")
	return fmt.Sprintf("Job %s deleted.", jobID), true, nil
}

func (m *Manager) helpText() string {
	return strings.Join([]string{
		"Usage: /jobs <command>",
		"Commands:",
		"- /jobs list",
		"- /jobs show <job_id>",
		"- /jobs pause <job_id>",
		"- /jobs resume <job_id>",
		"- /jobs run-now <job_id>",
		"- /jobs delete <job_id>",
		"- /jobs confirm-delete <job_id> <token>",
	}, "\n")
}

func (m *Manager) audit(actorID int64, jobID, operation, outcome string, extra ...any) {
	if m == nil || m.logger == nil {
		return
	}
	attrs := []any{
		"actor_user_id", actorID,
		"job_id", jobID,
		"operation", operation,
		"outcome", outcome,
	}
	attrs = append(attrs, extra...)
	m.logger.Info("jobs audit", attrs...)
}
