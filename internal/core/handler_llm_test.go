package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/memory"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"strings"
	"testing"
	"time"
)

// captureLLMLogWriter records the last Log call for assertion (AC-01.044).
type captureLLMLogWriter struct {
	lastModel string
}

func (c *captureLLMLogWriter) Log(entry *llmlog.Entry) {
	c.lastModel = entry.Model
}

// Covers AC-01.044, REQ-01.031, REQ-01.014: LLM log entry records the model/provider that produced the response (e.g. after fallback).
func TestHandleMessage_llmLogEntryRecordsResultModel(t *testing.T) {
	capLog := &captureLLMLogWriter{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi", Usage: llm.Usage{}, Model: "ollama/llama3"}}
	h := testHandlerDeps{
		router: mustRouterSingle(t, provider),
		logger: logger,
		llmLog: capLog,
		model:  "openai/gpt-4o", // default from first provider
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if capLog.lastModel != "ollama/llama3" {
		t.Errorf("LLM log entry Model = %q, want ollama/llama3 (result.Model when set)", capLog.lastModel)
	}
}

// Covers AC-01.044, REQ-01.031, REQ-01.014: when provider does not set result.Model, LLM log uses handler default (h.model).
func TestHandleMessage_llmLogEntryUsesDefaultModelWhenResultModelEmpty(t *testing.T) {
	capLog := &captureLLMLogWriter{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi", Usage: llm.Usage{}}}
	h := testHandlerDeps{
		router: mustRouterSingle(t, provider),
		logger: logger,
		llmLog: capLog,
		model:  "openai/gpt-4o",
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if capLog.lastModel != "openai/gpt-4o" {
		t.Errorf("LLM log entry Model = %q, want openai/gpt-4o (h.model when result.Model empty)", capLog.lastModel)
	}
}

// Covers AC-04.004, AC-04.008: tool_calls → execution → tool results → provider called again → final reply to user; errors surfaced in chat.
func TestHandleMessage_toolResultLoop_returnsFinalReply(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "hello from node"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Done. Result: hello from node.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "run echo hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "Done") && !strings.Contains(reply, "hello from node") {
		t.Errorf("HandleMessage: reply = %q, want final reply containing result", reply)
	}
	if callCount != 2 {
		t.Errorf("HandleMessage: provider.Complete calls = %d, want 2 (initial + after tool round)", callCount)
	}
	if runner.lastCommand != "echo hi" {
		t.Errorf("HandleMessage: RunOnNode command = %q, want echo hi", runner.lastCommand)
	}
}

// Covers AC-04.004: tool-result loop continues with tool outputs passed to follow-up completion.
func TestHandleMessage_toolResultLoop_largeToolOutput_truncatedForFollowUp(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	largeOut := strings.Repeat("z", maxToolResultPromptBytes+512)
	runner := &mockNodeRunner{stdout: largeOut}
	callCount := 0
	var secondCallMessages []llm.Message
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		secondCallMessages = append([]llm.Message(nil), messages...)
		return &llm.CompletionResult{Content: "done", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		catalog:               catalog,
		nodeRunner:            runner,
		logger:                slog.Default(),
		toolResultPromptBytes: maxToolResultPromptBytes,
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var found bool
	for _, m := range secondCallMessages {
		if m.Role != "tool" {
			continue
		}
		if strings.Contains(m.Content, "[tool output truncated:") {
			found = true
			if len(m.Content) >= len(largeOut) {
				t.Fatalf("tool message was not reduced; got=%d want<%d", len(m.Content), len(largeOut))
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected truncated tool message; messages=%+v", secondCallMessages)
	}
}

// Covers AC-04.006, AC-04.008: tool_call with invalid args (unknown tool) → no RunOnNode; error surfaced in chat (in messages to next provider call or in reply).
func TestHandleMessage_toolResultLoop_invalidArgs_noRunOnNode_errorInChat(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	callCount := 0
	var secondCallMessages []llm.Message
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			// Return tool_call with unknown tool id so validation fails.
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "unknown_tool", Arguments: `{"x":"y"}`}},
			}, nil
		}
		secondCallMessages = messages
		return &llm.CompletionResult{Content: "I could not run the tool: unknown tool.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "do something")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if runner.lastCommand != "" {
		t.Errorf("HandleMessage(invalid tool): RunOnNode must not be called; got lastCommand = %q", runner.lastCommand)
	}
	// Error must be visible: either in the tool result message passed to the second provider call, or in the final reply.
	var toolResultContainsError bool
	for _, m := range secondCallMessages {
		if m.Role == "tool" && strings.Contains(m.Content, "unknown tool") {
			toolResultContainsError = true
			break
		}
	}
	if !toolResultContainsError && !strings.Contains(reply, "unknown") {
		t.Errorf("HandleMessage: error must be in tool result or reply; secondCallMessages=%d, reply=%q", len(secondCallMessages), reply)
	}
}

// Covers AC-04.008: execution failure (run_on_node error) surfaced to user in chat (tool result content and/or final reply).
func TestHandleMessage_toolResultLoop_executionError_surfacedInChat(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	execErr := fmt.Errorf("command not in allowlist")
	runner := &mockNodeRunner{err: execErr}
	callCount := 0
	var secondCallMessages []llm.Message
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		secondCallMessages = messages
		return &llm.CompletionResult{Content: "The tool failed: command not in allowlist.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var toolResultContainsError bool
	for _, m := range secondCallMessages {
		if m.Role == "tool" && strings.Contains(m.Content, "allowlist") {
			toolResultContainsError = true
			break
		}
	}
	if !toolResultContainsError {
		t.Errorf("HandleMessage: execution error must appear in tool result message; messages = %v", secondCallMessages)
	}
	if !strings.Contains(reply, "allowlist") && !strings.Contains(reply, "failed") {
		t.Errorf("HandleMessage: reply should surface execution error; got %q", reply)
	}
}

// Covers AC-38.006
// Covers REQ-04.006: loop stops after maxToolRounds to avoid infinite loop when provider keeps returning tool_calls.
// Covers AC-01.002: traceability for TestHandleMessage_toolResultLoop_maxToolRounds_cap.
func TestHandleMessage_toolResultLoop_maxToolRounds_cap(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "ok"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		// Always return tool_calls so the loop would run forever without a cap.
		return &llm.CompletionResult{
			Content:   "",
			Usage:     llm.Usage{},
			ToolCalls: []llm.ToolCall{{ID: "call_x", Name: "run_echo", Arguments: `{"msg": "x"}`}},
		}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "run")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if callCount > maxToolRounds {
		t.Errorf("HandleMessage: Complete calls = %d, must be at most maxToolRounds=%d", callCount, maxToolRounds)
	}
	_ = reply
}

// Covers AC-04.013, REQ-04.016, AC-30.010: tool invocations are traceable; invoked_via is never hermes on the native path.
func TestHandleMessage_toolInvocation_loggedWithIdArgumentsAndResult(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "hello from node"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Done.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     logger,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "run echo hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var found bool
	for _, r := range cap.records {
		if r.msg != "tool invocation" {
			continue
		}
		found = true
		if r.attrs["tool_id"] != "run_echo" {
			t.Errorf("tool invocation log: tool_id = %q, want run_echo", r.attrs["tool_id"])
		}
		if r.attrs["arguments"] != `{"msg": "hi"}` {
			t.Errorf("tool invocation log: arguments = %q", r.attrs["arguments"])
		}
		if r.attrs["result"] != "hello from node" {
			t.Errorf("tool invocation log: result = %q, want hello from node", r.attrs["result"])
		}
		if r.attrs["invoked_via"] != "tool_calls" {
			t.Errorf("tool invocation log: invoked_via = %q, want tool_calls", r.attrs["invoked_via"])
		}
		break
	}
	if !found {
		t.Errorf("expected one Info \"tool invocation\" record; got records: %+v", cap.records)
	}
}

// Covers REQ-01.026: logRedactor applies to INFO tool invocation attrs (arguments, result, error).
// Covers AC-01.002: traceability for TestHandleMessage_toolInvocation_redactsInfoLogAttrs.
func TestHandleMessage_toolInvocation_redactsInfoLogAttrs(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "node says secret"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "secret"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Done.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:      mustRouterSingle(t, provider),
		catalog:     catalog,
		nodeRunner:  runner,
		logger:      logger,
		logRedactor: redactor,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var argsLog, resLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" {
			argsLog = r.attrs["arguments"]
			resLog = r.attrs["result"]
			break
		}
	}
	if argsLog == "" && resLog == "" {
		t.Fatalf("expected tool invocation log; records=%+v", cap.records)
	}
	if strings.Contains(argsLog, "secret") || !strings.Contains(argsLog, "[REDACTED]") {
		t.Errorf("arguments attr should be redacted; got %q", argsLog)
	}
	if strings.Contains(resLog, "secret") || !strings.Contains(resLog, "[REDACTED]") {
		t.Errorf("result attr should be redacted; got %q", resLog)
	}
}

// Covers AC-16.019: write_memory tool invocation arguments are passed through the same log redactor as other native tools.
func TestHandleMessage_writeMemory_toolInvocation_redactsArguments(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	memDir := t.TempDir()
	store, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	reg := tools.NewRegistry()
	reg.Register(tools.NewWriteMemoryTool(store, nil, nil, 4096, 1<<20))
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "w1", Name: "write_memory", Arguments: `{"text":"note secret body","date":"2026-04-14"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "saved", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		nativeRegistry:             reg,
		logger:                     logger,
		logRedactor:                redactor,
		firstProviderSupportsTools: true,
		maxDynamicSystemRunes:      defaultMaxDynamicSystemRunes,
		memoryVectorTopK:           testMemoryVectorTopK(10),
	}.handler()
	_, err = h.HandleMessage(context.Background(), 1, "", "remember this")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var argsLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["tool_id"] == "write_memory" {
			argsLog = r.attrs["arguments"]
			break
		}
	}
	if argsLog == "" {
		t.Fatalf("expected write_memory tool invocation log; records=%+v", cap.records)
	}
	if strings.Contains(argsLog, "secret") || !strings.Contains(argsLog, "[REDACTED]") {
		t.Errorf("write_memory arguments should be redacted; got %q", argsLog)
	}
}

// Covers REQ-01.026: INFO tool invocation error string is redacted (e.g. remote stderr from noderunner).
// Covers AC-01.002: traceability for TestHandleMessage_toolInvocation_redactsErrorAttr.
func TestHandleMessage_toolInvocation_redactsErrorAttr(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: errors.New("stderr: secret failure")}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "after tool", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:      mustRouterSingle(t, provider),
		catalog:     catalog,
		nodeRunner:  runner,
		logger:      logger,
		logRedactor: redactor,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var errLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["error"] != "" {
			errLog = r.attrs["error"]
			break
		}
	}
	if errLog == "" {
		t.Fatalf("expected tool invocation with error attr; records=%+v", cap.records)
	}
	if strings.Contains(errLog, "secret") || !strings.Contains(errLog, "[REDACTED]") {
		t.Errorf("error attr should be redacted; got %q", errLog)
	}
}

// Covers AC-04.013: tool invocation that fails (e.g. validation) is logged with error.
func TestHandleMessage_toolInvocation_loggedWithError(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "unknown_tool", Arguments: `{}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Tool failed.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     logger,
	}.handler()

	_, _ = h.HandleMessage(context.Background(), 1, "", "do something")
	var found bool
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["tool_id"] == "unknown_tool" {
			found = true
			if r.attrs["error"] == "" {
				t.Error("tool invocation (failed): expected error attr in log")
			}
			if r.attrs["invoked_via"] != "tool_calls" {
				t.Errorf("invoked_via = %q, want tool_calls", r.attrs["invoked_via"])
			}
			break
		}
	}
	if !found {
		t.Errorf("expected \"tool invocation\" record with error; got records: %+v", cap.records)
	}
}
