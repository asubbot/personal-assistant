package main

import (
	"context"
	"log/slog"
	"pa/internal/core"
	"pa/internal/jobs"
	"pa/internal/lifecyclelog"
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
		t0 := time.Now()
		openCtx := context.WithoutCancel(ctx)
		//nolint:contextcheck // jobs.Open currently does not accept context.
		st, err := jobs.Open(dbPath, policy)
		if err != nil {
			logger.Error("scheduled jobs init", "error", err)
			lifecyclelog.Error(logger, "jobs_runtime", "init", time.Since(t0), err, "lifecycle", "jobs_db_path", dbPath)
			state.setInitError(err)
			return
		}
		all, err := st.ListJobs(openCtx)
		if err != nil {
			logger.Error("scheduled jobs init list", "error", err)
			lifecyclelog.Error(logger, "jobs_runtime", "init", time.Since(t0), err, "lifecycle", "jobs_db_path", dbPath)
			_ = st.Close()
			state.setInitError(err)
			return
		}
		logger.Info("scheduled jobs runtime enabled", "jobs_db_path", dbPath, "jobs_loaded", len(all))
		runner := jobs.NewDeliveryRunner(baseHandler, sender, logger)
		runtime := jobs.NewRuntime(st, runner, jobs.RuntimeConfig{
			RunTimeout: 5 * time.Minute,
			Logger:     logger,
		})
		manager := jobs.NewManager(st, runtime, logger)
		manager.SetDefaultTimeZone(defaultTimeZone)
		startJobsRuntimeLoop(ctx, runtime, logger)
		state.setReady(manager)
		lifecyclelog.Info(logger, "jobs_runtime", "init", time.Since(t0), "lifecycle", "jobs_db_path", dbPath, "jobs_loaded", len(all))
		go func() {
			<-ctx.Done()
			_ = st.Close()
		}()
	}()
}
