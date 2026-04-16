package main

import (
	"context"
	"log/slog"
	"pa/internal/jobs"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustCreateJobViaExplicitTool(t *testing.T, ctx context.Context, manager *jobs.Manager, userID int64, chatID int64, instruction string, hour, minute int) {
	t.Helper()
	tool := jobs.NewCreateScheduledJobTool(manager)
	cctx := jobs.WithCreateContext(ctx, userID, chatID)
	createReply, err := tool.Run(cctx, map[string]any{
		"instruction": instruction,
		"hour":        float64(hour),
		"minute":      float64(minute),
	})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}
	if !strings.Contains(createReply, "Scheduled job created.") {
		t.Fatalf("create reply = %q", createReply)
	}
}

func mustListJobID(t *testing.T, ctx context.Context, manager *jobs.Manager, actorID int64) string {
	t.Helper()
	listReply, handled, err := manager.HandleCommand(ctx, actorID, "/jobs list")
	if err != nil || !handled {
		t.Fatalf("list err=%v handled=%v", err, handled)
	}
	if !strings.Contains(listReply, "Scheduled jobs:") {
		t.Fatalf("list reply = %q", listReply)
	}
	for _, line := range strings.Split(listReply, "\n") {
		if !strings.HasPrefix(line, "- job_") {
			continue
		}
		parts := strings.Split(line, " | ")
		if len(parts) == 0 {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(parts[0], "- "))
	}
	t.Fatalf("job id not found in list reply: %q", listReply)
	return ""
}

func mustShowInstruction(t *testing.T, ctx context.Context, manager *jobs.Manager, actorID int64, jobID string, instruction string) {
	t.Helper()
	showReply, handled, err := manager.HandleCommand(ctx, actorID, "/jobs show "+jobID)
	if err != nil || !handled {
		t.Fatalf("show err=%v handled=%v", err, handled)
	}
	if !strings.Contains(showReply, "instruction: "+instruction) {
		t.Fatalf("show reply = %q", showReply)
	}
}

func mustRunNowAndWaitDelivery(t *testing.T, ctx context.Context, manager *jobs.Manager, actorID int64, jobID string, sender *mockChatSender, expectedBody string) {
	t.Helper()
	runReply, handled, err := manager.HandleCommand(ctx, actorID, "/jobs run-now "+jobID)
	if err != nil || !handled {
		t.Fatalf("run-now err=%v handled=%v", err, handled)
	}
	if !strings.Contains(runReply, "started") {
		t.Fatalf("run-now reply = %q", runReply)
	}
	waitForSenderContains(t, sender, expectedBody, 2*time.Second)
}

func newRuntimeBoundManager(store *jobs.Store, sender *mockChatSender) *jobs.Manager {
	runner := &scheduledJobRunner{
		handler: &mockMessageHandler{reply: "AI digest ready"},
		sender:  sender,
		logger:  slog.New(slog.DiscardHandler),
	}
	runtime := jobs.NewRuntime(store, runner, jobs.RuntimeConfig{
		RunTimeout: 2 * time.Second,
		Logger:     slog.New(slog.DiscardHandler),
	})
	return jobs.NewManager(store, runtime, slog.New(slog.DiscardHandler))
}

// Covers AC-20.001, AC-20.002, AC-20.003, AC-20.005, AC-20.006, AC-20.008: job created via native tool path is manageable with /jobs and delivers on run-now.
func TestEP020_E2E_CreateManageRunNowDelivery(t *testing.T) {
	ctx := context.Background()
	store, err := jobs.Open(filepath.Join(t.TempDir(), "jobs.sqlite"))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	_, manager, sender := buildRuntimeAndManager(store)
	manager.SetDefaultTimeZone("UTC")
	mustCreateJobViaExplicitTool(t, ctx, manager, 555, 101, "AI news digest", 9, 0)
	jobID := mustListJobID(t, ctx, manager, 555)
	mustShowInstruction(t, ctx, manager, 555, jobID, "AI news digest")

	manager = newRuntimeBoundManager(store, sender)
	mustRunNowAndWaitDelivery(t, ctx, manager, 555, jobID, sender, "AI digest ready")
}

// Covers AC-20.001, AC-20.002, AC-20.003, AC-20.005, AC-20.006, AC-20.008: explicit create spec is manageable via /jobs and delivers on run-now.
func TestEP020_E2E_StrictTemplateCreateManageRunNowDelivery(t *testing.T) {
	ctx := context.Background()
	store, err := jobs.Open(filepath.Join(t.TempDir(), "jobs.sqlite"))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	_, manager, sender := buildRuntimeAndManager(store)
	manager.SetDefaultTimeZone("UTC")

	createReply, _, err := manager.CreateScheduledJobFromSpec(ctx, 555, 101, "Collect an AI news digest", 9, 0, "", "e2e_explicit")
	if err != nil {
		t.Fatalf("CreateScheduledJobFromSpec: %v", err)
	}
	if !strings.Contains(createReply, "Scheduled job created.") {
		t.Fatalf("create reply = %q", createReply)
	}

	jobID := mustListJobID(t, ctx, manager, 555)
	mustShowInstruction(t, ctx, manager, 555, jobID, "Collect an AI news digest")
	manager = newRuntimeBoundManager(store, sender)
	mustRunNowAndWaitDelivery(t, ctx, manager, 555, jobID, sender, "AI digest ready")
}
