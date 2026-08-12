package main

import (
	"context"
	"errors"
	"log/slog"
	"pa/cmd/pa/wire"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/jobs"
	"pa/internal/llm"
	"pa/internal/sqlitepragma"
	"pa/internal/toolindex"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
	text  string
	callN int
}

func (m *mockMessageHandler) HandleMessage(_ context.Context, _ int64, _ string, text string) (string, error) {
	m.text = text
	m.callN++
	return m.reply, m.err
}

// Supporting AC-20.002: parsed chat ID supplies the current-chat delivery target.
func TestParseDeliveryChatID(t *testing.T) {
	tests := []struct {
		name       string
		sessionKey string
		want       int64
		wantErr    bool
	}{
		{name: "positive", sessionKey: "123", want: 123},
		{name: "negative", sessionKey: "-100123", want: -100123},
		{name: "empty", sessionKey: "", wantErr: true},
		{name: "zero", sessionKey: "0", wantErr: true},
		{name: "non decimal", sessionKey: "s1", wantErr: true},
		{name: "scheduled job", sessionKey: "scheduled-job:x", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDeliveryChatID(tt.sessionKey)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseDeliveryChatID(%q) error = nil, want error", tt.sessionKey)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDeliveryChatID(%q): %v", tt.sessionKey, err)
			}
			if got != tt.want {
				t.Fatalf("parseDeliveryChatID(%q) = %d, want %d", tt.sessionKey, got, tt.want)
			}
		})
	}
}

// Supporting AC-19.002: malformed delivery session keys fail before command routing or base delegation.
func TestJobsCommandHandler_InvalidSessionKeyFailsFastEveryPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	readyState := wire.NewJobsRuntimeState()
	readyState.SetReady(jobs.NewManager(st, nil, slog.New(slog.DiscardHandler)))
	failedState := wire.NewJobsRuntimeState()
	failedState.SetFailed()

	tests := []struct {
		name       string
		sessionKey string
		text       string
		state      *wire.JobsRuntimeState
	}{
		{name: "base handler path", sessionKey: "s1", text: "hello", state: wire.NewJobsRuntimeState()},
		{name: "jobs initializing path", sessionKey: "", text: "/jobs list", state: wire.NewJobsRuntimeState()},
		{name: "jobs failed path", sessionKey: "0", text: "/jobs list", state: failedState},
		{name: "jobs ready list path", sessionKey: "scheduled-job:x", text: "/jobs list", state: readyState},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &mockMessageHandler{reply: "base"}
			h := &jobsCommandHandler{base: base, state: tt.state}

			reply, err := h.HandleMessage(context.Background(), 1, tt.sessionKey, tt.text)
			if err == nil {
				t.Fatal("HandleMessage error = nil, want error")
			}
			if reply != "" {
				t.Fatalf("HandleMessage reply = %q, want empty", reply)
			}
			if !strings.Contains(err.Error(), "parse delivery chat ID") {
				t.Fatalf("HandleMessage error = %q, want safe context", err)
			}
			if tt.sessionKey != "" && strings.Contains(err.Error(), tt.sessionKey) {
				t.Fatalf("HandleMessage error = %q, contains malformed session key", err)
			}
			if base.callN != 0 {
				t.Fatalf("base handler calls = %d, want 0", base.callN)
			}
		})
	}
}

// Covers AC-19.006, AC-19.007: scheduled run executes through message handler and delivers result.
func TestScheduledJobRunner_SuccessSendsResult(t *testing.T) {
	handler := &mockMessageHandler{reply: "Digest body"}
	sender := &mockChatSender{}
	r := jobs.NewDeliveryRunner(handler, sender, nil)
	job := jobs.Job{ID: "job-1", DeliveryChatID: 7, Instruction: "collect digest"}

	if err := r.Run(context.Background(), job); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if handler.text != "collect digest" {
		t.Fatalf("handler text = %q", handler.text)
	}
	chatID, text := sender.Snapshot()
	if chatID != 7 || !strings.Contains(text, "Digest body") {
		t.Fatalf("sender got chat=%d text=%q", chatID, text)
	}
}

// Covers AC-19.008: scheduled job failure is delivered with failure class.
func TestScheduledJobRunner_FailureSendsFailureClass(t *testing.T) {
	handler := &mockMessageHandler{err: errors.New("boom")}
	sender := &mockChatSender{}
	r := jobs.NewDeliveryRunner(handler, sender, nil)
	job := jobs.Job{ID: "job-2", DeliveryChatID: 9, Instruction: "collect digest"}

	err := r.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error")
	}
	chatID, text := sender.Snapshot()
	if chatID != 9 || !strings.Contains(text, "execution_error") {
		t.Fatalf("sender got chat=%d text=%q", chatID, text)
	}
}

// Covers AC-19.002, EP-021 AC-21.001: management command is gated until scheduler readiness is true.
func TestJobsCommandHandler_ReadinessGate(t *testing.T) {
	base := &mockMessageHandler{reply: "base"}
	state := wire.NewJobsRuntimeState()
	h := &jobsCommandHandler{base: base, state: state}

	reply, err := h.HandleMessage(context.Background(), 1, "1", "/jobs list")
	if err != nil {
		t.Fatalf("HandleMessage pre-ready: %v", err)
	}
	if !strings.Contains(reply, "initializing") {
		t.Fatalf("pre-ready reply = %q", reply)
	}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, openErr := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if openErr != nil {
		t.Fatalf("jobs.Open: %v", openErr)
	}
	t.Cleanup(func() { _ = st.Close() })
	mgr := jobs.NewManager(st, nil, slog.New(slog.DiscardHandler))
	state.SetReady(mgr)

	reply, err = h.HandleMessage(context.Background(), 1, "1", "/jobs list")
	if err != nil {
		t.Fatalf("HandleMessage ready: %v", err)
	}
	if !strings.Contains(reply, "No scheduled jobs configured.") {
		t.Fatalf("ready reply = %q", reply)
	}
}

// Covers AC-19.002, EP-021 AC-21.002: non-management traffic is not blocked by scheduler readiness.
func TestJobsCommandHandler_NonManagementBypassesReadiness(t *testing.T) {
	base := &mockMessageHandler{reply: "base ok"}
	state := wire.NewJobsRuntimeState()
	h := &jobsCommandHandler{base: base, state: state}

	reply, err := h.HandleMessage(context.Background(), 1, "1", "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "base ok" {
		t.Fatalf("reply = %q", reply)
	}
}

// Covers AC-19.002: /jobs command detection is strict and ignores /jobsx.
func TestJobsCommandHandler_StrictJobsCommandPrefix(t *testing.T) {
	base := &mockMessageHandler{reply: "base ok"}
	state := wire.NewJobsRuntimeState()
	h := &jobsCommandHandler{base: base, state: state}

	reply, err := h.HandleMessage(context.Background(), 1, "1", "/jobsx list")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "base ok" {
		t.Fatalf("reply = %q", reply)
	}
}

// Covers EP-021 AC-21.003: schedule-shaped free text is delegated to base in a single call (no wrapper-side create).
func TestJobsCommandHandler_ScheduleShapedMessageSingleBaseCall(t *testing.T) {
	base := &mockMessageHandler{reply: "assistant reply"}
	state := wire.NewJobsRuntimeState()
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mgr := jobs.NewManager(st, nil, slog.New(slog.DiscardHandler))
	state.SetReady(mgr)

	userText := "send me AI news digest at 08:15 every day"
	reply, err := h.HandleMessage(context.Background(), 101, "777", userText)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "assistant reply" {
		t.Fatalf("reply = %q", reply)
	}
	if base.callN != 1 || base.text != userText {
		t.Fatalf("base calls=%d text=%q", base.callN, base.text)
	}
	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("wrapper created jobs count=%d want 0", len(items))
	}
	_ = mgr
}

// Covers EP-021 AC-21.002, AC-20.007: authorized non-matching conversational message does not create jobs.
func TestJobsCommandHandler_NLCreateNonMatchingBypassesCreation(t *testing.T) {
	base := &mockMessageHandler{reply: "base reply"}
	state := wire.NewJobsRuntimeState()
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	state.SetReady(jobs.NewManager(st, nil, slog.New(slog.DiscardHandler)))

	reply, err := h.HandleMessage(context.Background(), 1, "42", "what is new in AI today?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "base reply" {
		t.Fatalf("reply = %q, want base reply", reply)
	}

	items, err := st.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("jobs count = %d, want 0", len(items))
	}
}

// Covers EP-021 AC-21.003: malformed clock in prose is still one base delegation (no wrapper retry).
func TestJobsCommandHandler_MalformedSchedulePhraseSingleBaseCall(t *testing.T) {
	base := &mockMessageHandler{reply: "base reply"}
	state := wire.NewJobsRuntimeState()
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	state.SetReady(jobs.NewManager(st, nil, slog.New(slog.DiscardHandler)))

	userText := "collect AI digest and send it at 9 every day"
	reply, err := h.HandleMessage(context.Background(), 1, "42", userText)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "base reply" {
		t.Fatalf("reply = %q, want base reply", reply)
	}
	if base.callN != 1 || base.text != userText {
		t.Fatalf("base call count = %d text=%q", base.callN, base.text)
	}
	_ = st
}

const (
	jobsMsgInitializing = "Scheduler is initializing. Please retry shortly."
	jobsMsgFailed       = "Scheduler is unavailable due to initialization error."
)

// Covers AC-42.002: failed jobs runtime returns stable /jobs message.
func TestJobsCommandHandler_FailedStateMessage(t *testing.T) {
	base := &mockMessageHandler{reply: "base"}
	state := wire.NewJobsRuntimeState()
	state.SetFailed()
	h := &jobsCommandHandler{base: base, state: state}

	reply, err := h.HandleMessage(context.Background(), 1, "1", "/jobs list")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != jobsMsgFailed {
		t.Fatalf("reply = %q, want %q", reply, jobsMsgFailed)
	}
}

// Covers AC-42.002: create_scheduled_job tool lookup via SnapshotLegacy in failed state.
func TestJobsRuntimeState_SnapshotLegacy_Failed(t *testing.T) {
	state := wire.NewJobsRuntimeState()
	state.SetFailed()
	mgr, ready, failed := state.SnapshotLegacy()
	if mgr != nil || !ready || !failed {
		t.Fatalf("SnapshotLegacy() = (%v, %v, %v), want (nil, true, true)", mgr, ready, failed)
	}
	tool := jobs.NewCreateScheduledJobToolWithRuntimeLookup(state.SnapshotLegacy)
	reply, err := tool.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if reply != jobsMsgFailed {
		t.Fatalf("reply = %q, want %q", reply, jobsMsgFailed)
	}
}

// Covers AC-42.003: readiness scheduled_jobs not OK while initializing.
func TestEvalReadiness_ScheduledJobsInitializing(t *testing.T) {
	idx := toolindex.NewIndex(noopVectorStore{})
	idx.SetReady(true)
	app := &paApplication{
		Cfg:          &config.Config{Paths: config.Paths{JobsDBPath: "/tmp/jobs.sqlite"}},
		LLMProviders: []llm.Provider{stubLLMProvider{}},
		Infra: paInfrastructure{
			MemVec: &core.MemoryVectors{
				Summaries: noopVectorStore{},
				Turns:     noopVectorStore{},
				Notes:     noopVectorStore{},
			},
			ToolIndex: idx,
		},
		JobsState: wire.NewJobsRuntimeState(),
	}
	body := app.EvalReadiness(context.Background())
	var jobsCheck *wire.ReadinessCheck
	for i := range body.Checks {
		if body.Checks[i].Name == "scheduled_jobs" {
			jobsCheck = &body.Checks[i]
			break
		}
	}
	if jobsCheck == nil {
		t.Fatal("missing scheduled_jobs check")
	}
	if jobsCheck.OK || !strings.Contains(jobsCheck.Detail, "initializing") {
		t.Fatalf("scheduled_jobs check = %+v", *jobsCheck)
	}
	if body.Ready {
		t.Fatal("expected overall not ready")
	}
}
