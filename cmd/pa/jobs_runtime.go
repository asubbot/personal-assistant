package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"pa/internal/core"
	"pa/internal/jobs"
	"pa/internal/sqlitepragma"
	"strconv"
	"strings"
	"sync"
	"time"
)

type chatSender interface {
	SendMessageToChat(ctx context.Context, chatID int64, text string) error
}

type jobsCommandHandler struct {
	base  core.MessageHandler
	state *jobsRuntimeState
}

func (h *jobsCommandHandler) HandleMessage(ctx context.Context, userID int64, sessionKey string, text string) (string, error) {
	ctxForBase := jobs.WithCreateContext(ctx, userID, parseDeliveryChatID(sessionKey, userID))
	if reply, handled, err := h.handleJobsCommand(ctx, userID, text); handled || err != nil {
		return reply, err
	}
	return h.base.HandleMessage(ctxForBase, userID, sessionKey, text)
}

func (h *jobsCommandHandler) handleJobsCommand(ctx context.Context, userID int64, text string) (string, bool, error) {
	mgr, ready, initFailed, handled := h.lookupManager(text)
	if !handled {
		return "", false, nil
	}
	if !ready {
		return "Scheduler is initializing. Please retry shortly.", true, nil
	}
	if initFailed {
		return "Scheduler is unavailable due to initialization error.", true, nil
	}
	if mgr == nil {
		return "Scheduler management is not configured.", true, nil
	}
	reply, managerHandled, err := mgr.HandleCommand(ctx, userID, text)
	return reply, managerHandled || err != nil, err
}

func (h *jobsCommandHandler) lookupManager(text string) (mgr *jobs.Manager, ready bool, initFailed bool, handled bool) {
	trimmed := strings.TrimSpace(text)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 || !isJobsCommandToken(fields[0]) {
		return nil, false, false, false
	}
	if h.state == nil {
		return nil, false, false, true
	}
	mgr, ready, initFailed = h.state.snapshot()
	return mgr, ready, initFailed, true
}

func isJobsCommandToken(token string) bool {
	lower := strings.ToLower(strings.TrimSpace(token))
	if lower == "/jobs" {
		return true
	}
	return strings.HasPrefix(lower, "/jobs@")
}

func parseDeliveryChatID(sessionKey string, fallback int64) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(sessionKey), 10, 64)
	if err != nil || v == 0 {
		return fallback
	}
	return v
}

type jobsRuntimeState struct {
	mu      sync.RWMutex
	manager *jobs.Manager
	ready   bool
	initErr error
}

func (s *jobsRuntimeState) setReady(manager *jobs.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manager = manager
	s.ready = true
	s.initErr = nil
}

func (s *jobsRuntimeState) setInitError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = true
	s.initErr = err
}

func (s *jobsRuntimeState) snapshot() (*jobs.Manager, bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.manager, s.ready, s.initErr != nil
}

type scheduledJobRunner struct {
	handler core.MessageHandler
	sender  chatSender
	logger  *slog.Logger
}

func (r *scheduledJobRunner) Run(ctx context.Context, job jobs.Job) error {
	if r.handler == nil {
		return fmt.Errorf("scheduled job runner: handler is nil")
	}
	logger := r.logger
	if logger == nil {
		logger = slog.Default()
	}
	sessionKey := fmt.Sprintf("scheduled-job:%s", job.ID)
	reply, err := r.handler.HandleMessage(ctx, job.DeliveryChatID, sessionKey, job.Instruction)
	notifyCtx := context.WithoutCancel(ctx)
	if err != nil {
		if r.sender != nil {
			msg := fmt.Sprintf("Scheduled job %s failed (%s).", job.ID, classifyJobFailure(err))
			if sendErr := r.sender.SendMessageToChat(notifyCtx, job.DeliveryChatID, msg); sendErr != nil {
				logger.Warn("scheduled job failure notification", "job_id", job.ID, "error", sendErr)
			}
		}
		logger.Info("jobs audit", "actor_user_id", 0, "job_id", job.ID, "operation", "delivery", "outcome", "failure_notified")
		return err
	}
	if r.sender != nil {
		body := strings.TrimSpace(reply)
		if body == "" {
			body = "(empty response)"
		}
		msg := fmt.Sprintf("Scheduled job %s result:\n%s", job.ID, body)
		if sendErr := r.sender.SendMessageToChat(notifyCtx, job.DeliveryChatID, msg); sendErr != nil {
			logger.Warn("scheduled job result notification", "job_id", job.ID, "error", sendErr)
		}
	}
	logger.Info("jobs audit", "actor_user_id", 0, "job_id", job.ID, "operation", "delivery", "outcome", "success_notified")
	return nil
}

func classifyJobFailure(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "execution_error"
}

func startJobsRuntimeLoop(ctx context.Context, rt *jobs.Runtime, logger *slog.Logger) {
	if rt == nil {
		return
	}
	if err := rt.EvaluateDue(ctx); err != nil {
		logger.Warn("scheduled jobs evaluate due", "error", err)
	}
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := rt.EvaluateDue(ctx); err != nil {
					logger.Warn("scheduled jobs evaluate due", "error", err)
				}
			}
		}
	}()
}

func initJobsRuntimeAsync(ctx context.Context, state *jobsRuntimeState, dbPath string, policy sqlitepragma.Policy, defaultTimeZone string, sender chatSender, baseHandler core.MessageHandler, logger *slog.Logger) {
	go func() {
		openCtx := context.WithoutCancel(ctx)
		//nolint:contextcheck // jobs.Open currently does not accept context.
		st, err := jobs.Open(dbPath, policy)
		if err != nil {
			logger.Error("scheduled jobs init", "error", err)
			state.setInitError(err)
			return
		}
		all, err := st.ListJobs(openCtx)
		if err != nil {
			logger.Error("scheduled jobs init list", "error", err)
			_ = st.Close()
			state.setInitError(err)
			return
		}
		logger.Info("scheduled jobs runtime enabled", "jobs_db_path", dbPath, "jobs_loaded", len(all))
		runner := &scheduledJobRunner{
			handler: baseHandler,
			sender:  sender,
			logger:  logger,
		}
		runtime := jobs.NewRuntime(st, runner, jobs.RuntimeConfig{
			RunTimeout: 5 * time.Minute,
			Logger:     logger,
		})
		manager := jobs.NewManager(st, runtime, logger)
		manager.SetDefaultTimeZone(defaultTimeZone)
		startJobsRuntimeLoop(ctx, runtime, logger)
		state.setReady(manager)
		go func() {
			<-ctx.Done()
			_ = st.Close()
		}()
	}()
}
