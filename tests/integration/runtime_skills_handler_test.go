//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/llm"
	"pa/internal/promptmarkers"
	"pa/internal/runtimeskills"
	"pa/internal/skillindex"
	"pa/internal/sqlitepragma"
	"pa/internal/systemprompt"
	"pa/internal/toolcatalog"
	"pa/internal/vector"
	"path/filepath"
	"strings"
	"testing"

	sqlitevec "pa/internal/vector/sqlite"
)

func testDiscardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// Covers AC-13.006, AC-13.013: selected runtime skill playbook appears inside RUNTIME_SKILLS markers.
func TestRuntimeSkills_handler_injectsPlaybook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "skills.sqlite")
	st, err := sqlitevec.NewWithTable(db, 4, sqlitevec.TableSkills, sqlitepragma.RecommendedPolicy(false))
	if err != nil {
		t.Fatal(err)
	}

	pkg := &runtimeskills.Package{
		ID: "s1", Name: "AlphaSkill", Description: "alpha beta gamma",
		Body: "UNIQUE_PLAYBOOK_BODY_RT_SKILLS", Tools: []string{"run_echo"},
	}
	if err := skillindex.Build(ctx, []*runtimeskills.Package{pkg}, core.IntegrationConstEmbedder{}, st); err != nil {
		t.Fatal(err)
	}
	skIdx := skillindex.NewIndex(st)
	skIdx.SetReady(true)
	t.Cleanup(func() { _ = skIdx.Close() })

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	toolVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo"}}}
	ti := &core.IntegrationMockToolIndex{StoreObj: toolVec, ReadyFlag: true}

	rs := &config.RuntimeSkillsConfig{
		Enabled: true, MaxSkillsPerTurn: 2, ToolVectorTopKCap: 10,
	}
	provider := &core.IntegrationMockLLM{Result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                     core.IntegrationMustRouterSingle(t, provider),
		Catalog:                    catalog,
		ToolIndex:                  ti,
		SkillIndex:                 skIdx,
		Embedder:                   core.IntegrationConstEmbedder{},
		RuntimeSkillsCfg:           rs,
		SkillPackagesByID:          map[string]*runtimeskills.Package{"s1": pkg},
		ToolSearchTopK:             10,
		ToolMinCount:               1,
		ToolFallbackCap:            50,
		Logger:                     testDiscardLogger(),
		FirstProviderSupportsTools: true,
	})

	_, err = h.HandleMessage(ctx, 1, "", "alpha beta gamma query")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.LastMessages[0].Content
	if !strings.Contains(sys, promptmarkers.BeginRuntimeSkills) || !strings.Contains(sys, "UNIQUE_PLAYBOOK_BODY_RT_SKILLS") {
		prefix := sys
		if len(prefix) > 300 {
			prefix = prefix[:300]
		}
		t.Fatalf("missing runtime skills block: %q", prefix)
	}
}

// Covers AC-13.004, REQ-13.016: tool blocks precede RETRIEVED_CONTEXT in the dynamic system tail.
func TestRuntimeSkills_handler_toolBlocksBeforeRetrievedMarkers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"t1": {
				ID: "t1", IndexText: "T1", SystemPrompt: "SYS_UNIQUE_ORDER",
				Template: "echo x", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	toolVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{ID: "t1"}}}
	ti := &core.IntegrationMockToolIndex{StoreObj: toolVec, ReadyFlag: true}
	memVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{Text: "memory snippet"}}}
	provider := &core.IntegrationMockLLM{Result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                     core.IntegrationMustRouterSingle(t, provider),
		Catalog:                    catalog,
		ToolIndex:                  ti,
		VectorStore:                memVec,
		Embedder:                   core.IntegrationConstEmbedder{},
		Logger:                     testDiscardLogger(),
		MaxDynamicSystemRunes:      4000,
		VectorSearchTopK:           5,
		ToolSearchTopK:             10,
		ToolMinCount:               1,
		ToolFallbackCap:            50,
		FirstProviderSupportsTools: true,
	})
	_, err := h.HandleMessage(ctx, 1, "", "q")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.LastMessages[0].Content
	iTool := strings.Index(sys, promptmarkers.BeginToolInstructions)
	iRet := strings.Index(sys, promptmarkers.BeginRetrievedContext)
	if iTool < 0 || iRet < 0 || iTool >= iRet {
		t.Fatalf("want tool block before retrieved: tool@%d retrieved@%d", iTool, iRet)
	}
}

// Covers AC-13.004: trust policy prefix and retrieved context markers.
func TestRuntimeSkills_handler_trustAndRetrievedMarkers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	provider := &core.IntegrationMockLLM{Result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	vs := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{Text: "past fact"}}}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                core.IntegrationMustRouterSingle(t, provider),
		VectorStore:           vs,
		Embedder:              core.IntegrationConstEmbedder{},
		Logger:                testDiscardLogger(),
		MaxDynamicSystemRunes: 4000,
		VectorSearchTopK:      5,
	})
	_, err := h.HandleMessage(ctx, 1, "", "hi")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.LastMessages[0].Content
	if !strings.HasPrefix(strings.TrimSpace(sys), systemprompt.TrustPolicy) {
		t.Fatalf("system should start with trust policy")
	}
	if !strings.Contains(sys, promptmarkers.BeginRetrievedContext) || !strings.Contains(sys, "past fact") {
		t.Fatalf("missing retrieved markers")
	}
}

// Covers AC-13.005: per-tool instructions wrapped in TOOL_INSTRUCTIONS markers.
func TestRuntimeSkills_handler_toolInstructionsMarkers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"t1": {
				ID: "t1", IndexText: "T1", SystemPrompt: "SYS_PROMPT_UNIQUE",
				Template: "echo x", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	toolVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{ID: "t1"}}}
	ti := &core.IntegrationMockToolIndex{StoreObj: toolVec, ReadyFlag: true}
	provider := &core.IntegrationMockLLM{Result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                     core.IntegrationMustRouterSingle(t, provider),
		Catalog:                    catalog,
		ToolIndex:                  ti,
		Embedder:                   core.IntegrationConstEmbedder{},
		ToolSearchTopK:             10,
		ToolMinCount:               1,
		ToolFallbackCap:            50,
		Logger:                     testDiscardLogger(),
		FirstProviderSupportsTools: true,
	})
	_, err := h.HandleMessage(ctx, 1, "", "q")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.LastMessages[0].Content
	if !strings.Contains(sys, promptmarkers.BeginToolInstructions) || !strings.Contains(sys, "SYS_PROMPT_UNIQUE") {
		t.Fatalf("missing tool instruction markers")
	}
}

// Covers AC-13.007: tools.always_include union with skill-linked and vector-selected tools.
func TestRuntimeSkills_handler_toolUnionAlwaysInclude(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "s2.sqlite")
	st, err := sqlitevec.NewWithTable(db, 4, sqlitevec.TableSkills, sqlitepragma.RecommendedPolicy(false))
	if err != nil {
		t.Fatal(err)
	}
	pkg := &runtimeskills.Package{
		ID: "s1", Name: "S", Description: "qqq", Body: "b", Tools: []string{"run_echo"},
	}
	if err := skillindex.Build(ctx, []*runtimeskills.Package{pkg}, core.IntegrationConstEmbedder{}, st); err != nil {
		t.Fatal(err)
	}
	skIdx := skillindex.NewIndex(st)
	skIdx.SetReady(true)
	t.Cleanup(func() { _ = skIdx.Close() })

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"test_tool": {
				ID: "test_tool", IndexText: "TT", Template: "echo {{arg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "arg", Type: "string", Required: true}},
			},
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	toolVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{ID: "run_echo"}}}
	ti := &core.IntegrationMockToolIndex{StoreObj: toolVec, ReadyFlag: true}
	rs := &config.RuntimeSkillsConfig{
		Enabled: true, MaxSkillsPerTurn: 2, ToolVectorTopKCap: 10,
	}
	var gotNames []string
	provider := &core.IntegrationMockLLM{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
		if opts != nil {
			for _, td := range opts.Tools {
				gotNames = append(gotNames, td.Name)
			}
		}
		return &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}, nil
	}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                     core.IntegrationMustRouterSingle(t, provider),
		Catalog:                    catalog,
		ToolIndex:                  ti,
		SkillIndex:                 skIdx,
		Embedder:                   core.IntegrationConstEmbedder{},
		RuntimeSkillsCfg:           rs,
		ToolsCfg:                   &config.ToolsConfig{AlwaysInclude: []string{"test_tool"}},
		SkillPackagesByID:          map[string]*runtimeskills.Package{"s1": pkg},
		ToolSearchTopK:             10,
		ToolMinCount:               1,
		ToolFallbackCap:            50,
		Logger:                     testDiscardLogger(),
		FirstProviderSupportsTools: true,
	})
	_, err = h.HandleMessage(ctx, 1, "", "qqq")
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, n := range gotNames {
		set[n] = true
	}
	if !set["test_tool"] || !set["run_echo"] {
		t.Fatalf("want test_tool and run_echo in tools, got %v", gotNames)
	}
}

// Covers AC-13.009: forbidden PA marker lines must not be indexed into vector stores.
func TestRuntimeSkills_handler_indexTurnRejectsForbiddenMarkerLine(t *testing.T) {
	t.Parallel()
	spy := &core.IntegrationMockVectorStore{}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		VectorStore: spy,
		Embedder:    core.IntegrationConstEmbedder{},
		Logger:      testDiscardLogger(),
	})
	err := core.IntegrationIndexTurn(h, context.Background(), "hello\n"+promptmarkers.BeginRetrievedContext, "reply")
	if err == nil {
		t.Fatal("expected error")
	}
	if spy.AddCalls != 0 {
		t.Fatalf("Add called %d times, want 0", spy.AddCalls)
	}
}

// Covers AC-13.008: when runtime skills disabled, no RUNTIME_SKILLS block; tool pre-selection still runs.
func TestRuntimeSkills_handler_disabledNoPlaybookToolPreselectionStillRuns(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "skills.sqlite")
	st, err := sqlitevec.NewWithTable(db, 4, sqlitevec.TableSkills, sqlitepragma.RecommendedPolicy(false))
	if err != nil {
		t.Fatal(err)
	}
	pkg := &runtimeskills.Package{
		ID: "hidden", Name: "HiddenSkill", Description: "unique skill phrase xyz",
		Body: "SECRET_SKILL_BODY", Tools: []string{},
	}
	if err := skillindex.Build(ctx, []*runtimeskills.Package{pkg}, core.IntegrationConstEmbedder{}, st); err != nil {
		t.Fatal(err)
	}
	skIdx := skillindex.NewIndex(st)
	skIdx.SetReady(true)
	t.Cleanup(func() { _ = skIdx.Close() })

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	toolVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo"}}}
	ti := &core.IntegrationMockToolIndex{StoreObj: toolVec, ReadyFlag: true}

	rs := &config.RuntimeSkillsConfig{
		Enabled: false, MaxSkillsPerTurn: 2, ToolVectorTopKCap: 10,
	}
	var toolNames []string
	provider := &core.IntegrationMockLLM{Result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
		if opts != nil {
			for _, td := range opts.Tools {
				toolNames = append(toolNames, td.Name)
			}
		}
		return &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}, nil
	}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                     core.IntegrationMustRouterSingle(t, provider),
		Catalog:                    catalog,
		ToolIndex:                  ti,
		SkillIndex:                 skIdx,
		Embedder:                   core.IntegrationConstEmbedder{},
		RuntimeSkillsCfg:           rs,
		SkillPackagesByID:          map[string]*runtimeskills.Package{"hidden": pkg},
		ToolSearchTopK:             10,
		ToolMinCount:               1,
		ToolFallbackCap:            50,
		Logger:                     testDiscardLogger(),
		FirstProviderSupportsTools: true,
	})
	_, err = h.HandleMessage(ctx, 1, "", "unique skill phrase xyz")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.LastMessages[0].Content
	if strings.Contains(sys, promptmarkers.BeginRuntimeSkills) || strings.Contains(sys, "SECRET_SKILL_BODY") {
		t.Fatalf("runtime skills block must be absent when disabled")
	}
	found := false
	for _, n := range toolNames {
		if n == "run_echo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want run_echo from tool pre-selection, got %v", toolNames)
	}
}

// REQ-13.016: Hermes block appears before RETRIEVED_CONTEXT when text-based tools are enabled.
// Covers AC-13.004: traceability for TestRuntimeSkills_handler_hermesBlockBeforeRetrievedMarkers.
func TestRuntimeSkills_handler_hermesBlockBeforeRetrievedMarkers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"t1": {
				ID: "t1", IndexText: "Hermes tool one line", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	toolVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{ID: "t1"}}}
	ti := &core.IntegrationMockToolIndex{StoreObj: toolVec, ReadyFlag: true}
	memVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{Text: "mem line"}}}
	provider := &core.IntegrationMockLLM{Result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                     core.IntegrationMustRouterSingle(t, provider),
		Catalog:                    catalog,
		ToolIndex:                  ti,
		VectorStore:                memVec,
		Embedder:                   core.IntegrationConstEmbedder{},
		Logger:                     testDiscardLogger(),
		MaxDynamicSystemRunes:      4000,
		VectorSearchTopK:           5,
		ToolSearchTopK:             10,
		ToolMinCount:               1,
		ToolFallbackCap:            50,
		TextBasedEnabled:           true,
		FirstProviderSupportsTools: false,
	})
	_, err := h.HandleMessage(ctx, 1, "", "q")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.LastMessages[0].Content
	iH := strings.Index(sys, promptmarkers.BeginHermesToolFormat)
	iR := strings.Index(sys, promptmarkers.BeginRetrievedContext)
	if iH < 0 || iR < 0 || iH >= iR {
		t.Fatalf("want Hermes block before retrieved: hermes@%d retrieved@%d", iH, iR)
	}
}

// Covers AC-13.012: system message content is unchanged across tool rounds.
func TestRuntimeSkills_handler_toolRoundPreservesSystemContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &core.IntegrationMockNodeRunner{Stdout: "out"}
	var firstSys string
	call := 0
	provider := &core.IntegrationMockLLM{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		call++
		if call == 1 {
			firstSys = messages[0].Content
			return &llm.CompletionResult{ToolCalls: []llm.ToolCall{{ID: "1", Name: "run_echo", Arguments: `{"msg":"a"}`}}}, nil
		}
		if messages[0].Content != firstSys {
			t.Errorf("system content changed between rounds")
		}
		return &llm.CompletionResult{Content: "done", Usage: llm.Usage{}}, nil
	}
	toolVec := &core.IntegrationMockVectorStore{SearchResults: []vector.SearchResult{{ID: "run_echo"}}}
	ti := &core.IntegrationMockToolIndex{StoreObj: toolVec, ReadyFlag: true}
	h := core.NewIntegrationConversationHandler(core.IntegrationConversationParams{
		Router:                     core.IntegrationMustRouterSingle(t, provider),
		Catalog:                    catalog,
		ToolIndex:                  ti,
		Embedder:                   core.IntegrationConstEmbedder{},
		NodeRunner:                 runner,
		Logger:                     testDiscardLogger(),
		ToolSearchTopK:             10,
		ToolMinCount:               1,
		ToolFallbackCap:            50,
		FirstProviderSupportsTools: true,
	})
	_, err := h.HandleMessage(ctx, 1, "", "hi")
	if err != nil {
		t.Fatal(err)
	}
}
