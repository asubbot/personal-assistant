//go:build integration

package integration_test

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/core/toolfailure"
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"strings"
	"testing"
	"time"
)

// ep006CallbackLLM implements llm.Provider with a per-test callback.
type ep006CallbackLLM struct {
	completeFn func(context.Context, []llm.Message, *llm.CompletionOptions) (*llm.CompletionResult, error)
}

func (m *ep006CallbackLLM) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	if m.completeFn != nil {
		return m.completeFn(ctx, messages, opts)
	}
	return &llm.CompletionResult{Content: "ok"}, nil
}

type ep006NodeRunner struct {
	err error
}

func (e *ep006NodeRunner) RunOnNode(_ context.Context, _, _ string) (string, error) {
	if e.err != nil {
		return "", e.err
	}
	return "", nil
}

func ep006ToolCatalog() *toolcatalog.Catalog {
	return &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
}

func boolPtr(b bool) *bool { return &b }

// fakeAdapterSequential runs several user messages through the same handler (EP-006 integration: baseline reset across messages).
type fakeAdapterSequential struct {
	userID   int64
	messages []string
	results  chan result
}

func (a *fakeAdapterSequential) Run(ctx context.Context, handler core.MessageHandler) error {
	for _, text := range a.messages {
		reply, err := handler.HandleMessage(ctx, a.userID, text)
		a.results <- result{reply: reply, err: err}
	}
	return nil
}

func ep006TwoMessageProviders(t *testing.T) (p0, p1, p2 llm.Provider, c0, c1, c2 *int) {
	t.Helper()
	var n0, n1, n2 int
	p0 = &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		n0++
		return &llm.CompletionResult{Content: "should not run p0 at baseline 1", Usage: llm.Usage{}}, nil
	}}
	p1 = &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		n1++
		if n1 == 1 {
			return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "t1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
		}
		return &llm.CompletionResult{Content: "second message from baseline p1", Usage: llm.Usage{}}, nil
	}}
	p2 = &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		n2++
		return &llm.CompletionResult{Content: "first message from p2 after escalate", Usage: llm.Usage{}}, nil
	}}
	return p0, p1, p2, &n0, &n1, &n2
}

// TestEP006_Run_twoMessages_resetsBaselineAfterEscalation mirrors internal/core run_ep006_escalation_test via core.Run (AC-06.008).
func TestEP006_Run_twoMessages_resetsBaselineAfterEscalation(t *testing.T) {
	t.Parallel()
	catalog := ep006ToolCatalog()
	runner := &ep006NodeRunner{err: toolfailure.MayEscalate(errors.New("fail"))}
	p0, p1, p2, c0, c1, c2 := ep006TwoMessageProviders(t)

	cfg := &config.Config{
		Version: 1,
		LLMProviders: []config.LLMProvider{
			{Type: "a", Model: "m0", SupportsTools: boolPtr(true)},
			{Type: "b", Model: "m1", SupportsTools: boolPtr(true)},
			{Type: "c", Model: "m2", SupportsTools: boolPtr(true)},
		},
		Tools: &config.ToolsConfig{
			LLMEscalation: &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 3, BaselineIndex: 1},
		},
		ToolCatalog: catalog,
	}
	ensureCoreRunConfigRequiredSections(cfg)

	results := make(chan result, 2)
	adapter := &fakeAdapterSequential{userID: 1, messages: []string{"one", "two"}, results: results}
	logger := slog.Default()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, []llm.Provider{p0, p1, p2}, []string{"m0", "m1", "m2"}, nil, nil, nil, runner, nil, nil, nil)
		close(runDone)
	}()

	wantSubstr := []string{"first message from p2", "second message from baseline p1"}
	for i, want := range wantSubstr {
		select {
		case res := <-results:
			if res.err != nil {
				t.Fatalf("message %d: %v", i+1, res.err)
			}
			if !strings.Contains(res.reply, want) {
				t.Errorf("reply %d = %q, want substring %q", i+1, res.reply, want)
			}
		case <-time.After(integrationTimeout):
			t.Fatalf("timeout waiting for message %d", i+1)
		}
	}
	cancel()
	<-runDone

	if *c0 != 0 || *c1 != 2 || *c2 != 1 {
		t.Errorf("Complete p0=%d p1=%d p2=%d, want 0,2,1", *c0, *c1, *c2)
	}
}

// TestEP006_Run_toolEscalation_secondProviderCompletes exercises core.Run → handler → tool → MayEscalate → second provider (REQ-06.006 / AC-06.006).
func TestEP006_Run_toolEscalation_secondProviderCompletes(t *testing.T) {
	t.Parallel()
	catalog := ep006ToolCatalog()
	runner := &ep006NodeRunner{err: toolfailure.MayEscalate(errors.New("node error"))}
	var c0, c1, c2 int
	p0 := &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c0++
		return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
	}}
	p1 := &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c1++
		return &llm.CompletionResult{Content: "integration recovered on model 2", Usage: llm.Usage{}}, nil
	}}
	p2 := &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c2++
		return &llm.CompletionResult{Content: "wrong", Usage: llm.Usage{}}, nil
	}}

	cfg := &config.Config{
		Version: 1,
		LLMProviders: []config.LLMProvider{
			{Type: "a", Model: "m0", SupportsTools: boolPtr(true)},
			{Type: "b", Model: "m1", SupportsTools: boolPtr(true)},
			{Type: "c", Model: "m2", SupportsTools: boolPtr(true)},
		},
		Tools: &config.ToolsConfig{
			LLMEscalation: &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 3, BaselineIndex: 0},
		},
		ToolCatalog: catalog,
	}
	ensureCoreRunConfigRequiredSections(cfg)

	adapter := &fakeAdapter{userID: 1, text: "run echo", done: make(chan result, 1)}
	logger := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, []llm.Provider{p0, p1, p2}, []string{"m0", "m1", "m2"}, nil, nil, nil, runner, nil, nil, nil)
		close(runDone)
	}()

	select {
	case res := <-adapter.done:
		if res.err != nil {
			t.Fatalf("handler error: %v", res.err)
		}
		if !strings.Contains(res.reply, "integration recovered") {
			t.Errorf("reply = %q", res.reply)
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("no reply within %v (test timeout)", integrationTimeout)
	}

	cancel()
	<-runDone

	if c0 != 1 || c1 != 1 || c2 != 0 {
		t.Errorf("Complete calls p0=%d p1=%d p2=%d, want 1,1,0", c0, c1, c2)
	}
}

// TestEP006_Run_threeProviders_threeMessages_chainAndBaselineReset covers EP-006 Task 7 integration bullet:
// three-provider chain, qualifying tool failure advances to second provider for that message, then messages 2 and 3
// each start from baseline again (AC-06.005, AC-06.006, AC-06.008, AC-06.010 / REQ-06.013).
func TestEP006_Run_threeProviders_threeMessages_chainAndBaselineReset(t *testing.T) {
	t.Parallel()
	catalog := ep006ToolCatalog()
	runner := &ep006NodeRunner{err: toolfailure.MayEscalate(errors.New("node fail"))}

	var c0, c1, c2 int
	p0 := &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c0++
		switch c0 {
		case 1:
			// User message 1: baseline p0 requests tool; tool fails → next Complete is p1.
			return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "t1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
		case 2:
			return &llm.CompletionResult{Content: "msg2_final_from_baseline_p0", Usage: llm.Usage{}}, nil
		case 3:
			return &llm.CompletionResult{Content: "msg3_final_from_baseline_p0", Usage: llm.Usage{}}, nil
		default:
			return nil, errors.New("unexpected extra p0 Complete")
		}
	}}
	p1 := &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c1++
		return &llm.CompletionResult{Content: "msg1_final_after_escalation_p1", Usage: llm.Usage{}}, nil
	}}
	p2 := &ep006CallbackLLM{completeFn: func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		c2++
		return &llm.CompletionResult{Content: "unexpected_p2", Usage: llm.Usage{}}, nil
	}}

	cfg := &config.Config{
		Version: 1,
		LLMProviders: []config.LLMProvider{
			{Type: "a", Model: "m0", SupportsTools: boolPtr(true)},
			{Type: "b", Model: "m1", SupportsTools: boolPtr(true)},
			{Type: "c", Model: "m2", SupportsTools: boolPtr(true)},
		},
		Tools: &config.ToolsConfig{
			LLMEscalation: &config.LLMEscalationConfig{Enabled: true, MaxPerUserMessage: 3, BaselineIndex: 0},
		},
		ToolCatalog: catalog,
	}
	ensureCoreRunConfigRequiredSections(cfg)

	results := make(chan result, 3)
	adapter := &fakeAdapterSequential{userID: 1, messages: []string{"first", "second", "third"}, results: results}
	logger := slog.Default()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, []llm.Provider{p0, p1, p2}, []string{"m0", "m1", "m2"}, nil, nil, nil, runner, nil, nil, nil)
		close(runDone)
	}()

	wantSubstr := []string{
		"msg1_final_after_escalation_p1",
		"msg2_final_from_baseline_p0",
		"msg3_final_from_baseline_p0",
	}
	for i, want := range wantSubstr {
		select {
		case res := <-results:
			if res.err != nil {
				t.Fatalf("message %d: %v", i+1, res.err)
			}
			if !strings.Contains(res.reply, want) {
				t.Errorf("reply %d = %q, want substring %q", i+1, res.reply, want)
			}
		case <-time.After(integrationTimeout):
			t.Fatalf("timeout waiting for message %d", i+1)
		}
	}
	cancel()
	<-runDone

	if c0 != 3 || c1 != 1 || c2 != 0 {
		t.Errorf("Complete p0=%d p1=%d p2=%d, want 3,1,0 (three baseline-first completes on p0; one p1 after escalation on msg1)", c0, c1, c2)
	}
}
