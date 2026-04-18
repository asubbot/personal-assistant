//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/core"
	"pa/internal/jobs"
	"pa/internal/sqlitepragma"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockChatSender struct {
	mu     sync.Mutex
	chatID int64
	text   string
	err    error
}

func (m *mockChatSender) SendMessageToChat(_ context.Context, chatID int64, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatID = chatID
	m.text = text
	return m.err
}

func (m *mockChatSender) Snapshot() (int64, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.chatID, m.text
}

type mockMessageHandler struct {
	reply string
	err   error
}

func (m *mockMessageHandler) HandleMessage(_ context.Context, _ int64, _ string, text string) (string, error) {
	return m.reply, m.err
}

var _ core.MessageHandler = (*mockMessageHandler)(nil)

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
	runner := jobs.NewDeliveryRunner(&mockMessageHandler{reply: "AI digest ready"}, sender, slog.New(slog.DiscardHandler))
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

// Covers AC-19.023, AC-25.001: end-to-end flow delivers digest, lists job, and removes it via delete confirmation.
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
	return "" // unreachable when the test fails; required for vet on all return paths
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
	runner := jobs.NewDeliveryRunner(&mockMessageHandler{reply: "AI digest ready"}, sender, slog.New(slog.DiscardHandler))
	runtime := jobs.NewRuntime(store, runner, jobs.RuntimeConfig{
		RunTimeout: 2 * time.Second,
		Logger:     slog.New(slog.DiscardHandler),
	})
	return jobs.NewManager(store, runtime, slog.New(slog.DiscardHandler))
}

// Covers AC-20.001, AC-20.002, AC-20.003, AC-20.005, AC-20.006, AC-20.008, AC-25.001: job created via native tool path is manageable with /jobs and delivers on run-now.
func TestEP020_E2E_CreateManageRunNowDelivery(t *testing.T) {
	ctx := context.Background()
	store, err := jobs.Open(filepath.Join(t.TempDir(), "jobs.sqlite"), sqlitepragma.RecommendedPolicy(true))
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

// Covers AC-20.001, AC-20.002, AC-20.003, AC-20.005, AC-20.006, AC-20.008, AC-25.001: explicit create spec is manageable via /jobs and delivers on run-now.
func TestEP020_E2E_StrictTemplateCreateManageRunNowDelivery(t *testing.T) {
	ctx := context.Background()
	store, err := jobs.Open(filepath.Join(t.TempDir(), "jobs.sqlite"), sqlitepragma.RecommendedPolicy(true))
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
