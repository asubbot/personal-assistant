package main

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/jobs"
	"pa/internal/sqlitepragma"
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

// Covers AC-19.006, AC-19.007: scheduled run executes through message handler and delivers result.
func TestScheduledJobRunner_SuccessSendsResult(t *testing.T) {
	handler := &mockMessageHandler{reply: "Digest body"}
	sender := &mockChatSender{}
	r := &scheduledJobRunner{handler: handler, sender: sender}
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
	r := &scheduledJobRunner{handler: handler, sender: sender}
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
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	reply, err := h.HandleMessage(context.Background(), 1, "s1", "/jobs list")
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
	state.setReady(mgr)

	reply, err = h.HandleMessage(context.Background(), 1, "s1", "/jobs list")
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
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	reply, err := h.HandleMessage(context.Background(), 1, "s1", "hello")
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
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	reply, err := h.HandleMessage(context.Background(), 1, "s1", "/jobsx list")
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
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mgr := jobs.NewManager(st, nil, slog.New(slog.DiscardHandler))
	state.setReady(mgr)

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
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	state.setReady(jobs.NewManager(st, nil, slog.New(slog.DiscardHandler)))

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
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath, sqlitepragma.RecommendedPolicy(true))
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	state.setReady(jobs.NewManager(st, nil, slog.New(slog.DiscardHandler)))

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
