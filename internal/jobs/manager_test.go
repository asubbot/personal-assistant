package jobs

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type runtimeStub struct {
	called bool
	jobID  string
	err    error
}

func (s *runtimeStub) RunNow(_ context.Context, jobID string) error {
	s.called = true
	s.jobID = jobID
	return s.err
}

func newManagerFixture(t *testing.T) (*Manager, *Store, *runtimeStub, Job) {
	t.Helper()
	st := openTestStore(t)
	ctx := context.Background()
	job, err := st.CreateJob(ctx, JobInput{
		Name:           "digest",
		ScheduleExpr:   "0 9 * * *",
		TimeZone:       "UTC",
		Instruction:    "Collect AI digest",
		DeliveryChatID: 99,
		Status:         StatusActive,
		OverlapPolicy:  OverlapSingleInstance,
		TimeoutPolicy:  TimeoutCancelAfter,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	next := time.Now().UTC().Add(time.Hour)
	if err := st.SetJobNextRun(ctx, job.ID, &next); err != nil {
		t.Fatalf("SetJobNextRun: %v", err)
	}
	run := &runtimeStub{}
	m := NewManager(st, run, slog.New(slog.DiscardHandler))
	return m, st, run, job
}

// Covers AC-19.011, AC-19.012: list and show commands return required fields.
func TestManager_ListAndShow(t *testing.T) {
	m, _, _, job := newManagerFixture(t)
	ctx := context.Background()

	reply, handled, err := m.HandleCommand(ctx, 1, "/jobs list")
	if err != nil || !handled {
		t.Fatalf("list err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, job.ID) {
		t.Fatalf("list reply missing job id: %q", reply)
	}

	reply, handled, err = m.HandleCommand(ctx, 1, "/jobs show "+job.ID)
	if err != nil || !handled {
		t.Fatalf("show err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "Collect AI digest") {
		t.Fatalf("show reply: %q", reply)
	}
	if !strings.Contains(reply, "delivery_chat_id: 99") {
		t.Fatalf("show reply missing delivery target: %q", reply)
	}
}

// Covers AC-19.013: pause and resume update persisted status.
func TestManager_PauseResume(t *testing.T) {
	m, st, _, job := newManagerFixture(t)
	ctx := context.Background()

	reply, handled, err := m.HandleCommand(ctx, 1, "/jobs pause "+job.ID)
	if err != nil || !handled {
		t.Fatalf("pause err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "paused") {
		t.Fatalf("pause reply: %q", reply)
	}

	paused, err := st.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob paused: %v", err)
	}
	if paused.Status != StatusPaused {
		t.Fatalf("paused status = %q", paused.Status)
	}

	reply, handled, err = m.HandleCommand(ctx, 1, "/jobs resume "+job.ID)
	if err != nil || !handled {
		t.Fatalf("resume err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "resumed") {
		t.Fatalf("resume reply: %q", reply)
	}
}

// Covers AC-19.014: run-now delegates to runtime API.
func TestManager_RunNow(t *testing.T) {
	m, _, run, job := newManagerFixture(t)
	ctx := context.Background()

	reply, handled, err := m.HandleCommand(ctx, 1, "/jobs run-now "+job.ID)
	if err != nil || !handled {
		t.Fatalf("run-now err=%v handled=%v", err, handled)
	}
	if !run.called || run.jobID != job.ID {
		t.Fatalf("runtime not called: %+v", run)
	}
	if !strings.Contains(reply, "started") {
		t.Fatalf("run-now reply: %q", reply)
	}
}

// Covers AC-19.015: unknown command and unknown id return deterministic responses.
func TestManager_UnknownAndMissingIDDeterministic(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	ctx := context.Background()

	reply, handled, err := m.HandleCommand(ctx, 1, "/jobs unknown")
	if err != nil || !handled {
		t.Fatalf("unknown err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "Unknown /jobs command") {
		t.Fatalf("reply = %q", reply)
	}

	reply, handled, err = m.HandleCommand(ctx, 1, "/jobs show missing-id")
	if err != nil || !handled {
		t.Fatalf("show missing err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "not found") {
		t.Fatalf("reply = %q", reply)
	}
}

// Covers AC-19.015: non-/jobs prefixes are not classified as management commands.
func TestManager_ParseJobsCommand_StrictPrefix(t *testing.T) {
	m, _, _, _ := newManagerFixture(t)
	ctx := context.Background()

	reply, handled, err := m.HandleCommand(ctx, 1, "/jobsx list")
	if err != nil {
		t.Fatalf("HandleCommand err=%v", err)
	}
	if handled {
		t.Fatalf("expected /jobsx to bypass manager, reply=%q", reply)
	}
}

// Covers AC-19.016, AC-19.017: delete challenge and confirm-delete remove the job.
func TestManager_DeleteConfirm_HappyPath(t *testing.T) {
	m, st, _, job := newManagerFixture(t)
	ctx := context.Background()

	reply, handled, err := m.HandleCommand(ctx, 777, "/jobs delete "+job.ID)
	if err != nil || !handled {
		t.Fatalf("delete err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "Token:") {
		t.Fatalf("delete reply = %q", reply)
	}
	token := ""
	lines := strings.Split(reply, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Token: ") {
			token = strings.TrimSpace(strings.TrimPrefix(line, "Token: "))
		}
	}
	if token == "" {
		t.Fatal("missing token in delete response")
	}

	reply, handled, err = m.HandleCommand(ctx, 777, "/jobs confirm-delete "+job.ID+" "+token)
	if err != nil || !handled {
		t.Fatalf("confirm-delete err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "deleted") {
		t.Fatalf("confirm-delete reply = %q", reply)
	}
	if _, err := st.GetJob(ctx, job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetJob after delete err=%v want ErrNotFound", err)
	}
}

// Covers AC-19.017: confirm-delete rejects mismatched actor and expired token.
func TestManager_DeleteConfirm_EdgeCases(t *testing.T) {
	m, _, _, job := newManagerFixture(t)
	ctx := context.Background()

	reply, handled, err := m.HandleCommand(ctx, 1, "/jobs delete "+job.ID)
	if err != nil || !handled {
		t.Fatalf("delete err=%v handled=%v", err, handled)
	}
	token := ""
	for _, line := range strings.Split(reply, "\n") {
		if strings.HasPrefix(line, "Token: ") {
			token = strings.TrimSpace(strings.TrimPrefix(line, "Token: "))
		}
	}
	if token == "" {
		t.Fatal("missing token")
	}

	reply, handled, err = m.HandleCommand(ctx, 2, "/jobs confirm-delete "+job.ID+" "+token)
	if err != nil || !handled {
		t.Fatalf("confirm-delete(actor mismatch) err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "another operator") {
		t.Fatalf("reply = %q", reply)
	}

	m.now = func() time.Time { return time.Now().UTC().Add(10 * time.Minute) }
	reply, handled, err = m.HandleCommand(ctx, 1, "/jobs confirm-delete "+job.ID+" "+token)
	if err != nil || !handled {
		t.Fatalf("confirm-delete(expired) err=%v handled=%v", err, handled)
	}
	if !strings.Contains(reply, "expired") {
		t.Fatalf("reply = %q", reply)
	}
}

// Covers AC-19.021: audit event includes actor, job, operation, outcome and timestamp fields.
func TestManager_AuditLogFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	m, _, _, job := newManagerFixture(t)
	m.logger = logger

	reply, handled, err := m.HandleCommand(context.Background(), 42, "/jobs show "+job.ID)
	if err != nil || !handled {
		t.Fatalf("show err=%v handled=%v reply=%q", err, handled, reply)
	}
	logText := buf.String()
	if !strings.Contains(logText, "jobs audit") {
		t.Fatalf("missing jobs audit log: %s", logText)
	}
	for _, token := range []string{"time=", "actor_user_id=42", "job_id=" + job.ID, "operation=show", "outcome=success"} {
		if !strings.Contains(logText, token) {
			t.Fatalf("missing %q in audit log: %s", token, logText)
		}
	}
}

// Covers AC-19.021: runtime-unavailable management failure emits audit outcome.
func TestManager_AuditLog_RuntimeUnavailable(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	st := openTestStore(t)
	m := NewManager(st, nil, logger)

	job, err := st.CreateJob(context.Background(), JobInput{
		Name:           "digest",
		ScheduleExpr:   "0 9 * * *",
		TimeZone:       "UTC",
		Instruction:    "Collect AI digest",
		DeliveryChatID: 99,
		Status:         StatusActive,
		OverlapPolicy:  OverlapSingleInstance,
		TimeoutPolicy:  TimeoutCancelAfter,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	_, handled, err := m.HandleCommand(context.Background(), 42, "/jobs run-now "+job.ID)
	if !handled || err == nil {
		t.Fatalf("run-now handled=%v err=%v", handled, err)
	}
	logText := buf.String()
	for _, token := range []string{"actor_user_id=42", "job_id=" + job.ID, "operation=run_now", "outcome=runtime_unavailable"} {
		if !strings.Contains(logText, token) {
			t.Fatalf("missing %q in audit log: %s", token, logText)
		}
	}
}

// Covers EP-021 AC-21.004: CreateScheduledJobFromSpec persists daily job with confirmation (replaces strict NL path).
func TestManager_CreateScheduledJobFromSpec_DirectCreatesJob(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	m.SetDefaultTimeZone("Europe/Berlin")

	reply, created, err := m.CreateScheduledJobFromSpec(
		context.Background(),
		11,
		22,
		"Collect an AI news digest",
		9,
		30,
		"",
		"test_explicit",
	)
	if err != nil {
		t.Fatalf("CreateScheduledJobFromSpec: %v", err)
	}
	for _, token := range []string{"Scheduled job created.", "schedule: 30 9 * * *", "timezone: Europe/Berlin", "instruction: Collect an AI news digest"} {
		if !strings.Contains(reply, token) {
			t.Fatalf("reply missing %q: %q", token, reply)
		}
	}
	if created.NextRunAt == nil {
		t.Fatal("created.NextRunAt is nil")
	}

	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(items))
	}
	if items[0].DeliveryChatID != 22 {
		t.Fatalf("delivery_chat_id = %d, want 22", items[0].DeliveryChatID)
	}
	if items[0].TimeZone != "Europe/Berlin" {
		t.Fatalf("timezone = %q, want Europe/Berlin", items[0].TimeZone)
	}
	if items[0].NextRunAt == nil {
		t.Fatal("persisted NextRunAt is nil")
	}
}

// Covers EP-021 AC-21.005: explicit tool rejects invalid clock with message and nil error.
func TestCreateScheduledJobTool_InvalidHourReturnsMessageNoError(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	tool := NewCreateScheduledJobTool(m)

	reply, err := tool.Run(context.Background(), map[string]any{
		"instruction":   "Do something",
		"hour":          float64(25),
		"minute":        float64(0),
		"actor_user_id": float64(1),
	})
	if err != nil {
		t.Fatalf("Run err = %v, want nil", err)
	}
	if !strings.Contains(reply, "Invalid schedule") {
		t.Fatalf("reply = %q", reply)
	}
	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("jobs count = %d, want 0", len(items))
	}
}

// Covers EP-021 AC-21.004, AC-21.009: native tool creates job and logs creation_path=native_tool_explicit.
func TestCreateScheduledJobTool_CreatesJobAndAuditsPath(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, logger)
	m.SetDefaultTimeZone("UTC")
	tool := NewCreateScheduledJobTool(m)

	reply, err := tool.Run(context.Background(), map[string]any{
		"instruction":      "AI news digest",
		"hour":             float64(8),
		"minute":           float64(15),
		"actor_user_id":    float64(77),
		"delivery_chat_id": float64(88),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(reply, "schedule: 15 8 * * *") {
		t.Fatalf("reply = %q", reply)
	}
	logText := buf.String()
	if !strings.Contains(logText, "creation_path=native_tool_explicit") {
		t.Fatalf("logs: %s", logText)
	}
}

// Covers EP-021 AC-21.004: native tool reads actor and delivery ids from context when params omit them.
func TestCreateScheduledJobTool_UsesContextActorAndDelivery(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	m.SetDefaultTimeZone("UTC")
	tool := NewCreateScheduledJobTool(m)

	ctx := WithCreateContext(context.Background(), 401, 777)
	reply, err := tool.Run(ctx, map[string]any{
		"instruction": "AI digest",
		"hour":        float64(10),
		"minute":      float64(20),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(reply, "Scheduled job created.") {
		t.Fatalf("reply = %q", reply)
	}

	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(items))
	}
	if items[0].DeliveryChatID != 777 {
		t.Fatalf("delivery_chat_id = %d, want 777", items[0].DeliveryChatID)
	}
}

// Covers EP-021 AC-21.005 (Trace: REQ-21.010): empty instruction returns a soft message and does not persist a job.
func TestCreateScheduledJobTool_EmptyInstructionReturnsMessageNoError(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	tool := NewCreateScheduledJobTool(m)

	reply, err := tool.Run(context.Background(), map[string]any{
		"instruction":   "   ",
		"hour":          float64(10),
		"minute":        float64(0),
		"actor_user_id": float64(1),
	})
	if err != nil {
		t.Fatalf("Run err = %v, want nil", err)
	}
	if !strings.Contains(reply, "Instruction must be non-empty") {
		t.Fatalf("reply = %q", reply)
	}
	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("jobs count = %d, want 0", len(items))
	}
}

// Covers EP-021 AC-21.005 (Trace: REQ-21.010): schema rejects non-number hour before schedule parsing (tools layer).
func TestCreateScheduledJobTool_StringHourFailsValidateParams(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	tool := NewCreateScheduledJobTool(m)

	_, err := tool.Run(context.Background(), map[string]any{
		"instruction":   "Do something",
		"hour":          "9",
		"minute":        float64(0),
		"actor_user_id": float64(1),
	})
	if err == nil || !strings.Contains(err.Error(), `param "hour" must be number`) {
		t.Fatalf("err = %v, want tools hour type error", err)
	}
	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("jobs count = %d, want 0", len(items))
	}
}

// Covers EP-021 AC-21.005: parseCreateScheduledJobArgs soft path when hour is not int/int64/float64 (e.g. future callers bypassing ValidateParams).
func TestParseCreateScheduledJobArgs_NonNumericHourReturnsSoftMessage(t *testing.T) {
	ctx := WithCreateContext(context.Background(), 1, 2)
	args, soft, err := parseCreateScheduledJobArgs(ctx, map[string]any{
		"instruction": "x",
		"hour":        uint(9),
		"minute":      float64(0),
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if soft == "" {
		t.Fatalf("want soft message, got args=%+v", args)
	}
	if !strings.Contains(soft, "hour and minute must be integers") {
		t.Fatalf("soft = %q", soft)
	}
}

// Covers EP-021 AC-21.004 (Trace: REQ-21.004, REQ-21.009): optional timezone param is persisted and audit keeps native_tool_explicit.
func TestCreateScheduledJobTool_ExplicitTimezoneInReplyAndStore(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, logger)
	m.SetDefaultTimeZone("UTC")
	tool := NewCreateScheduledJobTool(m)

	reply, err := tool.Run(context.Background(), map[string]any{
		"instruction":      "Digest",
		"hour":             float64(7),
		"minute":           float64(45),
		"timezone":         "Europe/Paris",
		"actor_user_id":    float64(2),
		"delivery_chat_id": float64(3),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(reply, "timezone: Europe/Paris") || !strings.Contains(reply, "schedule: 45 7 * * *") {
		t.Fatalf("reply = %q", reply)
	}
	logText := buf.String()
	if !strings.Contains(logText, "creation_path=native_tool_explicit") {
		t.Fatalf("audit log missing creation_path: %s", logText)
	}
	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(items))
	}
	if items[0].TimeZone != "Europe/Paris" {
		t.Fatalf("job timezone = %q, want Europe/Paris", items[0].TimeZone)
	}
}

// Covers EP-021 AC-21.004: CreateScheduledJobToolWithLookup resolves the manager from callback (same as cmd/pa wiring).
func TestCreateScheduledJobToolWithLookup_UsesCallbackManager(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	m.SetDefaultTimeZone("UTC")
	tool := NewCreateScheduledJobToolWithLookup(func() *Manager { return m })

	reply, err := tool.Run(context.Background(), map[string]any{
		"instruction":   "Task",
		"hour":          float64(12),
		"minute":        float64(0),
		"actor_user_id": float64(5),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(reply, "Scheduled job created.") {
		t.Fatalf("reply = %q", reply)
	}
}

// Covers EP-021 AC-21.004: lookup returning nil surfaces configuration error (no panic).
func TestCreateScheduledJobToolWithLookup_NilManagerReturnsError(t *testing.T) {
	tool := NewCreateScheduledJobToolWithLookup(func() *Manager { return nil })
	_, err := tool.Run(context.Background(), map[string]any{
		"instruction":   "Task",
		"hour":          float64(1),
		"minute":        float64(0),
		"actor_user_id": float64(1),
	})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v, want not configured", err)
	}
}

// Covers AC-27.004. Supporting AC-27.006: exercised under full make check.
func TestCreateScheduledJobTool_RuntimeLookup_NotReadyReturnsSoft(t *testing.T) {
	tool := NewCreateScheduledJobToolWithRuntimeLookup(func() (*Manager, bool, bool) {
		return nil, false, false
	})
	reply, err := tool.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	const want = "Scheduler is initializing. Please retry shortly."
	if reply != want {
		t.Fatalf("reply = %q, want %q", reply, want)
	}
}

// Covers AC-27.004. Supporting AC-27.006: exercised under full make check.
func TestCreateScheduledJobTool_RuntimeLookup_InitFailedReturnsSoft(t *testing.T) {
	tool := NewCreateScheduledJobToolWithRuntimeLookup(func() (*Manager, bool, bool) {
		return nil, true, true
	})
	reply, err := tool.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	const want = "Scheduler is unavailable due to initialization error."
	if reply != want {
		t.Fatalf("reply = %q, want %q", reply, want)
	}
}

// Covers EP-021 AC-21.004: missing actor_user_id and create context yields a hard error from the tool path.
func TestCreateScheduledJobTool_MissingActorReturnsError(t *testing.T) {
	st := openTestStore(t)
	m := NewManager(st, &runtimeStub{}, slog.New(slog.DiscardHandler))
	tool := NewCreateScheduledJobTool(m)

	_, err := tool.Run(context.Background(), map[string]any{
		"instruction": "Task",
		"hour":        float64(1),
		"minute":      float64(0),
	})
	if err == nil || !strings.Contains(err.Error(), "actor_user_id is required") {
		t.Fatalf("err = %v, want actor_user_id is required", err)
	}
}
