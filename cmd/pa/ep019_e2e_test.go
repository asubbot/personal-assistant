package main

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/jobs"
	"pa/internal/sqlitepragma"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func extractDeleteToken(reply string) string {
	for _, line := range strings.Split(reply, "\n") {
		if strings.HasPrefix(line, "Token: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Token: "))
		}
	}
	return ""
}

func waitForSenderContains(t *testing.T, sender *mockChatSender, needle string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, text := sender.Snapshot()
		if strings.Contains(text, needle) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	_, text := sender.Snapshot()
	t.Fatalf("timeout waiting for sender text containing %q, got %q", needle, text)
}

func createDigestJob(t *testing.T, ctx context.Context, store *jobs.Store) jobs.Job {
	t.Helper()
	job, err := store.CreateJob(ctx, jobs.JobInput{
		Name:           "daily-digest",
		ScheduleExpr:   "* * * * *",
		TimeZone:       "UTC",
		Instruction:    "Collect AI digest",
		DeliveryChatID: 101,
		Status:         jobs.StatusActive,
		OverlapPolicy:  jobs.OverlapSingleInstance,
		TimeoutPolicy:  jobs.TimeoutCancelAfter,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	past := time.Now().UTC().Add(-time.Second)
	if err := store.SetJobNextRun(ctx, job.ID, &past); err != nil {
		t.Fatalf("SetJobNextRun: %v", err)
	}
	return job
}

func buildRuntimeAndManager(store *jobs.Store) (*jobs.Runtime, *jobs.Manager, *mockChatSender) {
	sender := &mockChatSender{}
	runner := &scheduledJobRunner{
		handler: &mockMessageHandler{reply: "AI digest ready"},
		sender:  sender,
		logger:  slog.New(slog.DiscardHandler),
	}
	runtime := jobs.NewRuntime(store, runner, jobs.RuntimeConfig{
		RunTimeout: 2 * time.Second,
		Logger:     slog.New(slog.DiscardHandler),
	})
	manager := jobs.NewManager(store, runtime, slog.New(slog.DiscardHandler))
	return runtime, manager, sender
}

func mustListContains(t *testing.T, ctx context.Context, manager *jobs.Manager, jobID string) {
	t.Helper()
	listReply, handled, err := manager.HandleCommand(ctx, 555, "/jobs list")
	if err != nil || !handled {
		t.Fatalf("list err=%v handled=%v", err, handled)
	}
	if !strings.Contains(listReply, jobID) {
		t.Fatalf("list reply does not include job id %q: %q", jobID, listReply)
	}
}

func deleteByConfirmation(t *testing.T, ctx context.Context, manager *jobs.Manager, jobID string) {
	t.Helper()
	deleteReply, handled, err := manager.HandleCommand(ctx, 555, "/jobs delete "+jobID)
	if err != nil || !handled {
		t.Fatalf("delete err=%v handled=%v", err, handled)
	}
	token := extractDeleteToken(deleteReply)
	if token == "" {
		t.Fatalf("delete reply has no token: %q", deleteReply)
	}
	confirmCmd := fmt.Sprintf("/jobs confirm-delete %s %s", jobID, token)
	confirmReply, handled, err := manager.HandleCommand(ctx, 555, confirmCmd)
	if err != nil || !handled {
		t.Fatalf("confirm-delete err=%v handled=%v", err, handled)
	}
	if !strings.Contains(confirmReply, "deleted") {
		t.Fatalf("confirm-delete reply: %q", confirmReply)
	}
}

func mustListExcludes(t *testing.T, ctx context.Context, manager *jobs.Manager, jobID string) {
	t.Helper()
	listAfter, handled, err := manager.HandleCommand(ctx, 555, "/jobs list")
	if err != nil || !handled {
		t.Fatalf("list-after err=%v handled=%v", err, handled)
	}
	if strings.Contains(listAfter, jobID) {
		t.Fatalf("job should be removed from list: %q", listAfter)
	}
}

// Covers AC-19.023: end-to-end flow delivers digest, lists job, and removes it via delete confirmation.
func TestEP019_E2E_DigestListDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := jobs.Open(filepath.Join(t.TempDir(), "jobs.sqlite"), sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	job := createDigestJob(t, ctx, store)
	runtime, manager, sender := buildRuntimeAndManager(store)
	if err := runtime.EvaluateDue(ctx); err != nil {
		t.Fatalf("EvaluateDue: %v", err)
	}
	waitForSenderContains(t, sender, "AI digest ready", 2*time.Second)
	mustListContains(t, ctx, manager, job.ID)
	deleteByConfirmation(t, ctx, manager, job.ID)
	mustListExcludes(t, ctx, manager, job.ID)
}
