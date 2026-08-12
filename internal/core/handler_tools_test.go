package core

import (
	"context"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"pa/internal/vector"
	"strings"
	"testing"
)

// Covers AC-04.007, AC-04.010: valid tool call → substitute template → execute via RunOnNode; allowlist enforced by existing path.
func TestExecuteOneToolCall_ValidCall_RunsViaRunOnNode(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo on node",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "hello from node"}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()

	stdout, err := h.executeOneToolCall(context.Background(), "run_echo", `{"msg": "hello"}`)
	if err != nil {
		t.Fatalf("executeOneToolCall: %v", err)
	}
	if stdout != "hello from node" {
		t.Errorf("executeOneToolCall: stdout = %q, want hello from node", stdout)
	}
	if runner.lastNodeID != "nas" || runner.lastCommand != "echo hello" {
		t.Errorf("executeOneToolCall: RunOnNode called with (%q, %q), want (nas, echo hello)", runner.lastNodeID, runner.lastCommand)
	}
}

// Covers AC-09.008: native run_on_node dispatch when id not in catalog.
func TestExecuteOneToolCall_nativeRunOnNode(t *testing.T) {
	runner := &mockNodeRunner{stdout: "up"}
	reg := tools.NewRegistry()
	reg.Register(tools.NewRunOnNode(runner))
	h := testHandlerDeps{
		catalog:        &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		nativeRegistry: reg,
		nodeRunner:     runner,
		logger:         slog.Default(),
	}.handler()
	out, err := h.executeOneToolCall(context.Background(), "run_on_node", `{"node_id":"nas","command":"uptime"}`)
	if err != nil {
		t.Fatalf("executeOneToolCall: %v", err)
	}
	if out != "up" {
		t.Errorf("got %q", out)
	}
}

// Covers AC-01.002: traceability for TestRemoteCommandFromRunOnNodeArgs.
func TestRemoteCommandFromRunOnNodeArgs(t *testing.T) {
	if got := remoteCommandFromRunOnNodeArgs("run_on_node", `{"node_id":"nas","command":"  docker ps  "}`); got != "docker ps" {
		t.Errorf("got %q, want docker ps", got)
	}
	if got := remoteCommandFromRunOnNodeArgs("run_echo", `{"command":"x"}`); got != "" {
		t.Errorf("non-native tool: got %q, want empty", got)
	}
	if got := remoteCommandFromRunOnNodeArgs("run_on_node", `not json`); got != "" {
		t.Errorf("invalid json: got %q, want empty", got)
	}
}

// Covers AC-04.006: unknown tool → error, no RunOnNode called.
func TestExecuteOneToolCall_UnknownTool_ReturnsErrorNoRun(t *testing.T) {
	catalog := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()

	_, err := h.executeOneToolCall(context.Background(), "unknown", `{}`)
	if err == nil {
		t.Fatal("executeOneToolCall(unknown tool): expected error, got nil")
	}
	if runner.lastCommand != "" {
		t.Error("executeOneToolCall(unknown tool): RunOnNode must not be called")
	}
}

// Covers AC-04.012: changed tool-loop prompt behavior is covered by unit tests.
func TestTruncateToolResultForPrompt(t *testing.T) {
	h := testHandlerDeps{toolResultPromptBytes: maxToolResultPromptBytes}.handler()
	small := "ok"
	if got := h.truncateToolResultForPrompt(small); got != small {
		t.Fatalf("small content changed: got %q", got)
	}
	large := strings.Repeat("a", maxToolResultPromptBytes+73)
	got := h.truncateToolResultForPrompt(large)
	if got == large {
		t.Fatal("expected large content to be truncated")
	}
	if !strings.Contains(got, "[tool output truncated: 73 bytes omitted]") {
		t.Fatalf("missing truncation marker: %q", got[len(got)-80:])
	}
	if len(got) >= len(large) {
		t.Fatalf("expected shorter content after truncation; got=%d want<%d", len(got), len(large))
	}
}

// Covers AC-39.006
func TestTruncateToolResultForPrompt_usesConfiguredLimit(t *testing.T) {
	const customLimit = 4096
	h := testHandlerDeps{toolResultPromptBytes: customLimit}.handler()
	large := strings.Repeat("b", customLimit+37)
	got := h.truncateToolResultForPrompt(large)
	if !strings.Contains(got, "[tool output truncated: 37 bytes omitted]") {
		t.Fatalf("missing truncation marker for configured limit: %q", got[len(got)-80:])
	}
}

// Covers AC-04.003, AC-04.015: when toolIndex and catalog are set, completion request includes pre-selected tools in provider format.
func TestHandleMessage_requestContainsPreselectedTools(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_uptime": {
				ID:        "run_uptime",
				IndexText: "Run uptime on the node",
				Template:  "uptime",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "node_id", Type: "string", Required: true}},
			},
		},
	}
	// Tool index returns run_uptime from search (or fallback will include it from catalog).
	toolStore := &mockVectorStore{
		searchResults: []vector.SearchResult{{ID: "run_uptime", Text: "uptime", Score: 0.9}},
	}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}

	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     logger,
		firstProviderSupportsTools: true,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "check server status")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if provider.lastOpts == nil {
		t.Fatal("expected Complete to be called with non-nil opts (tools)")
	}
	if len(provider.lastOpts.Tools) == 0 {
		t.Errorf("expected pre-selected tools in request; got opts.Tools empty")
	}
	// First tool should be from catalog (pre-selection returned run_uptime).
	found := false
	for _, td := range provider.lastOpts.Tools {
		if td.Name == "run_uptime" {
			found = true
			if td.Description != "Run uptime on the node" {
				t.Errorf("ToolDef Description = %q, want Run uptime on the node", td.Description)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected tool run_uptime in opts.Tools; got %v", provider.lastOpts.Tools)
	}
}

// Covers AC-30.001: with catalog tools selected, native tool defs are attached on the completion path (REQ-30.002, REQ-30.016).
// AC-04.026 / REQ-04.032: first system message includes per-tool [id] blocks for non-empty system_prompt when tools are selected.
func TestHandleMessage_firstSystemMessage_includesSystemPromptSections(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	const marker = "UNIQUE_SYSTEM_PROMPT_MARKER_8264"
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"alpha_tool": {
				ID:           "alpha_tool",
				IndexText:    "Alpha capability for testing",
				SystemPrompt: marker + "\nSecond line of rules.",
				Template:     "echo alpha",
				NodeID:       "nas",
				Arguments:    []toolcatalog.ArgumentRule{},
			},
		},
	}
	toolStore := &mockVectorStore{
		searchResults: []vector.SearchResult{{ID: "alpha_tool", Text: "alpha", Score: 0.9}},
	}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     logger,
		firstProviderSupportsTools: true,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "use alpha")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(provider.lastMessages) < 1 {
		t.Fatal("expected at least one message to provider")
	}
	sys := provider.lastMessages[0]
	if sys.Role != "system" {
		t.Fatalf("first message role = %q, want system", sys.Role)
	}
	if !strings.Contains(sys.Content, "Tool instructions:") {
		t.Errorf("system message missing Tool instructions header: %q", sys.Content)
	}
	if !strings.Contains(sys.Content, "[alpha_tool]") {
		t.Errorf("system message missing [alpha_tool] section: %q", sys.Content)
	}
	if !strings.Contains(sys.Content, marker) {
		t.Errorf("system message missing system_prompt body: %q", sys.Content)
	}
	if provider.lastOpts == nil || len(provider.lastOpts.Tools) == 0 {
		t.Fatalf("expected native tool defs in completion options, got opts=%v", provider.lastOpts)
	}
}

// Covers AC-30.002: assistant free-text `<tool_call>` without native tool_calls does not execute catalog tools.
func TestHandleMessage_fakeToolCallMarkupWithoutNativeToolCalls_noToolExecution(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	provider := &mockProvider{result: &llm.CompletionResult{
		Content: `<tool_call>{"name":"run_echo","arguments":{"msg":"x"}}</tool_call>`,
		Usage:   llm.Usage{},
	}}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 runner,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}.handler()
	if _, err := h.HandleMessage(context.Background(), 1, "", "run echo"); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if runner.lastCommand != "" {
		t.Fatalf("RunOnNode must not run for markup-only assistant text; lastCommand=%q", runner.lastCommand)
	}
}

// Covers AC-04.029 (REQ-04.031): substituted command must pass cmdsafe.ValidateRemoteCommand before RunOnNode (e.g. `;` rejected as disallowed rune).
func TestExecuteOneToolCall_substitutedCommandWithMetachar_noRunOnNode(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()
	_, err := h.executeOneToolCall(context.Background(), "run_echo", `{"msg": "hi;rm -rf /"}`)
	if err == nil {
		t.Fatal("executeOneToolCall: expected error for metacharacter in substituted command")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
}

// Substituted command with a disallowed rune (e.g. tab) must not reach RunOnNode.
// Covers AC-01.002: traceability for TestExecuteOneToolCall_substitutedCommandWithDisallowedRune_noRunOnNode.
func TestExecuteOneToolCall_substitutedCommandWithDisallowedRune_noRunOnNode(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()
	_, err := h.executeOneToolCall(context.Background(), "run_echo", "{\"msg\": \"x\\ty\"}")
	if err == nil {
		t.Fatal("executeOneToolCall: expected error for tab in substituted command")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
}

// Catalog substitution passes cmdsafe gate in handler: INFO log includes tool_id, node_id, remote_command.
// Covers AC-01.002: traceability for TestExecuteOneToolCall_catalogCmdsafeRejection_logsRemoteCommand.
func TestExecuteOneToolCall_catalogCmdsafeRejection_logsRemoteCommand(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: logger}.handler()
	_, err := h.executeOneToolCall(context.Background(), "run_echo", `{"msg": "bad;cmd"}`)
	if err == nil {
		t.Fatal("executeOneToolCall: expected cmdsafe error")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
	var found bool
	for _, rec := range cap.records {
		if rec.msg != "catalog tool remote command rejected" {
			continue
		}
		found = true
		if rec.attrs["tool_id"] != "run_echo" || rec.attrs["node_id"] != "nas" {
			t.Errorf("attrs = %v", rec.attrs)
		}
		if !strings.Contains(rec.attrs["remote_command"], "bad") {
			t.Errorf("remote_command = %q", rec.attrs["remote_command"])
		}
		break
	}
	if !found {
		t.Fatalf("expected catalog tool remote command rejected log; records=%+v", cap.records)
	}
}
