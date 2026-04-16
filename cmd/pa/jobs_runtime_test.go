package main

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/jobs"
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

type sequentialMockMessageHandler struct {
	replies []string
	errs    []error
	texts   []string
	callN   int
}

func (m *sequentialMockMessageHandler) HandleMessage(_ context.Context, _ int64, _ string, text string) (string, error) {
	m.texts = append(m.texts, text)
	idx := m.callN
	m.callN++
	if idx < len(m.errs) && m.errs[idx] != nil {
		return "", m.errs[idx]
	}
	if idx < len(m.replies) {
		return m.replies[idx], nil
	}
	return "", nil
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

// Covers AC-19.002: management command is gated until scheduler readiness is true.
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
	st, openErr := jobs.Open(dbPath)
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

// Covers AC-19.002: non-management traffic is not blocked by scheduler readiness.
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

// Covers AC-20.009: send-first explicit schedule-intent message is routed to fallback create flow.
func TestJobsCommandHandler_NLCreateFallbackRouting(t *testing.T) {
	base := &mockMessageHandler{reply: "base ok"}
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath)
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mgr := jobs.NewManager(st, nil, slog.New(slog.DiscardHandler))
	state.setReady(mgr)

	reply, err := h.HandleMessage(context.Background(), 101, "777", "send me AI news digest at 08:15 every day")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "Scheduled job created.") {
		t.Fatalf("reply = %q", reply)
	}
	if base.text != "" {
		t.Fatalf("base handler should not be used for create path, got text=%q", base.text)
	}

	listReply, err := h.HandleMessage(context.Background(), 101, "777", "/jobs list")
	if err != nil {
		t.Fatalf("HandleMessage list: %v", err)
	}
	if !strings.Contains(listReply, "Scheduled jobs:") {
		t.Fatalf("list reply = %q", listReply)
	}
}

// Covers AC-20.007: authorized non-matching conversational message does not create jobs.
func TestJobsCommandHandler_NLCreateNonMatchingBypassesCreation(t *testing.T) {
	base := &mockMessageHandler{reply: "base reply"}
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath)
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

// Covers AC-20.004, AC-20.007: malformed schedule-intent message without strict/fallback match is passed to base handler.
func TestJobsCommandHandler_NLCreateMalformedFallsThroughToBase(t *testing.T) {
	base := &mockMessageHandler{reply: "base reply"}
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath)
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	state.setReady(jobs.NewManager(st, nil, slog.New(slog.DiscardHandler)))

	reply, err := h.HandleMessage(context.Background(), 1, "42", "collect AI digest and send it at 9 every day")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "base reply" {
		t.Fatalf("reply = %q, want base reply", reply)
	}
	if base.callN != 2 {
		t.Fatalf("base call count = %d, want 2", base.callN)
	}
	if !strings.Contains(base.text, "Now you MUST call create_scheduled_job tool.") {
		t.Fatalf("base retry text = %q", base.text)
	}
	_ = st
}

// Covers AC-20.004, AC-20.007: LLM fallback retries with explicit tool-call requirement when first reply lacks create confirmation.
func TestJobsCommandHandler_NLCreateMalformedRetriesFallbackPrompt(t *testing.T) {
	base := &sequentialMockMessageHandler{
		replies: []string{
			"I cannot create this directly.",
			"Scheduled job created.\njob_id: job-123\nschedule: 0 9 * * *",
		},
	}
	state := &jobsRuntimeState{}
	h := &jobsCommandHandler{base: base, state: state}

	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.Open(dbPath)
	if err != nil {
		t.Fatalf("jobs.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	state.setReady(jobs.NewManager(st, nil, slog.New(slog.DiscardHandler)))

	reply, err := h.HandleMessage(context.Background(), 1, "42", "collect AI digest and send it at 9 every day")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "Scheduled job created.") {
		t.Fatalf("reply = %q", reply)
	}
	if base.callN != 2 {
		t.Fatalf("base call count = %d, want 2", base.callN)
	}
	if len(base.texts) < 2 {
		t.Fatalf("base prompts = %d, want >=2", len(base.texts))
	}
	if !strings.Contains(base.texts[0], "Use the create_scheduled_job tool") {
		t.Fatalf("first prompt = %q", base.texts[0])
	}
	if !strings.Contains(base.texts[1], "Now you MUST call create_scheduled_job tool.") {
		t.Fatalf("second prompt = %q", base.texts[1])
	}
}
