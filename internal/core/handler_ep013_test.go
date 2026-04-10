package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/promptmarkers"
	"pa/internal/runtimeskills"
	"pa/internal/skillindex"
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

type constEmb4 struct{}

func (constEmb4) Embed(context.Context, string) ([]float32, error) {
	return []float32{1, 0, 0, 0}, nil
}

func TestHandleMessage_runtimeSkills_injectsPlaybook(t *testing.T) {
	// Covers AC-13.006, AC-13.013
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "skills.sqlite")
	st, err := sqlitevec.NewWithTable(db, 4, sqlitevec.TableSkills)
	if err != nil {
		t.Fatal(err)
	}

	pkg := &runtimeskills.Package{
		ID: "s1", Name: "AlphaSkill", Description: "alpha beta gamma",
		Body: "UNIQUE_PLAYBOOK_BODY_EP013", Tools: []string{"run_echo"},
	}
	if err := skillindex.Build(ctx, []*runtimeskills.Package{pkg}, constEmb4{}, st); err != nil {
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
	toolVec := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo"}}}
	ti := &mockToolIndex{store: toolVec, ready: true}

	rs := &config.RuntimeSkillsConfig{
		Enabled: true, MaxSkillsPerTurn: 2, ToolVectorTopKCap: 10,
	}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		skillIndex:                 skIdx,
		embedder:                   constEmb4{},
		runtimeSkillsCfg:           rs,
		skillPackagesByID:          map[string]*runtimeskills.Package{"s1": pkg},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     testDiscardLogger(),
		firstProviderSupportsTools: true,
	}

	_, err = h.HandleMessage(ctx, 1, "alpha beta gamma query")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.lastMessages[0].Content
	if !strings.Contains(sys, promptmarkers.BeginRuntimeSkills) || !strings.Contains(sys, "UNIQUE_PLAYBOOK_BODY_EP013") {
		prefix := sys
		if len(prefix) > 300 {
			prefix = prefix[:300]
		}
		t.Fatalf("missing runtime skills block: %q", prefix)
	}
}

func TestHandleMessage_toolBlocksBeforeRetrievedMarkers(t *testing.T) {
	// Covers AC-13.004, REQ-13.016 (tool / Hermes blocks before RETRIEVED_CONTEXT; tail ordering)
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
	toolVec := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "t1"}}}
	ti := &mockToolIndex{store: toolVec, ready: true}
	memVec := &mockVectorStore{searchResults: []vector.SearchResult{{Text: "memory snippet"}}}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		vectorStore:                memVec,
		embedder:                   constEmb4{},
		logger:                     testDiscardLogger(),
		maxDynamicSystemRunes:      4000,
		vectorSearchTopK:           5,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		firstProviderSupportsTools: true,
	}
	_, err := h.HandleMessage(ctx, 1, "q")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.lastMessages[0].Content
	iTool := strings.Index(sys, promptmarkers.BeginToolInstructions)
	iRet := strings.Index(sys, promptmarkers.BeginRetrievedContext)
	if iTool < 0 || iRet < 0 || iTool >= iRet {
		t.Fatalf("want tool block before retrieved: tool@%d retrieved@%d", iTool, iRet)
	}
}

func TestHandleMessage_trustAndRetrievedMarkers(t *testing.T) {
	// Covers AC-13.004
	ctx := context.Background()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	vs := &mockVectorStore{searchResults: []vector.SearchResult{{Text: "past fact"}}}
	h := &conversationHandler{
		router:                mustRouterSingle(t, provider),
		vectorStore:           vs,
		embedder:              constEmb4{},
		logger:                testDiscardLogger(),
		maxDynamicSystemRunes: 4000,
		vectorSearchTopK:      5,
	}
	_, err := h.HandleMessage(ctx, 1, "hi")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.lastMessages[0].Content
	if !strings.HasPrefix(strings.TrimSpace(sys), systemprompt.TrustPolicy) {
		t.Fatalf("system should start with trust policy")
	}
	if !strings.Contains(sys, promptmarkers.BeginRetrievedContext) || !strings.Contains(sys, "past fact") {
		t.Fatalf("missing retrieved markers")
	}
}

func TestHandleMessage_toolInstructionsMarkers(t *testing.T) {
	// Covers AC-13.005
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
	toolVec := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "t1"}}}
	ti := &mockToolIndex{store: toolVec, ready: true}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		embedder:                   constEmb4{},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     testDiscardLogger(),
		firstProviderSupportsTools: true,
	}
	_, err := h.HandleMessage(ctx, 1, "q")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.lastMessages[0].Content
	if !strings.Contains(sys, promptmarkers.BeginToolInstructions) || !strings.Contains(sys, "SYS_PROMPT_UNIQUE") {
		t.Fatalf("missing tool instruction markers")
	}
}

func TestHandleMessage_toolUnion_alwaysInclude(t *testing.T) {
	// Covers AC-13.007
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "s2.sqlite")
	st, err := sqlitevec.NewWithTable(db, 4, sqlitevec.TableSkills)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &runtimeskills.Package{
		ID: "s1", Name: "S", Description: "qqq", Body: "b", Tools: []string{"run_echo"},
	}
	if err := skillindex.Build(ctx, []*runtimeskills.Package{pkg}, constEmb4{}, st); err != nil {
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
	toolVec := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo"}}}
	ti := &mockToolIndex{store: toolVec, ready: true}
	rs := &config.RuntimeSkillsConfig{
		Enabled: true, MaxSkillsPerTurn: 2, ToolVectorTopKCap: 10,
	}
	var gotNames []string
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
		if opts != nil {
			for _, td := range opts.Tools {
				gotNames = append(gotNames, td.Name)
			}
		}
		return &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}, nil
	}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		skillIndex:                 skIdx,
		embedder:                   constEmb4{},
		runtimeSkillsCfg:           rs,
		toolsCfg:                   &config.ToolsConfig{AlwaysInclude: []string{"test_tool"}},
		skillPackagesByID:          map[string]*runtimeskills.Package{"s1": pkg},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     testDiscardLogger(),
		firstProviderSupportsTools: true,
	}
	_, err = h.HandleMessage(ctx, 1, "qqq")
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

// addSpyVec counts vector Store.Add calls (AC-13.009).
type addSpyVec struct {
	*mockVectorStore
	addCalls int
}

func (a *addSpyVec) Add(ctx context.Context, id string, emb []float32, chunk string) error {
	a.addCalls++
	if a.mockVectorStore != nil {
		return a.mockVectorStore.Add(ctx, id, emb, chunk)
	}
	return nil
}

func TestIndexTurn_rejectsForbiddenMarkerLine(t *testing.T) {
	// Covers AC-13.009
	spy := &addSpyVec{mockVectorStore: &mockVectorStore{}}
	h := &conversationHandler{
		vectorStore: spy,
		embedder:    constEmb4{},
		logger:      testDiscardLogger(),
	}
	err := h.indexTurn(context.Background(), "hello\n"+promptmarkers.BeginRetrievedContext, "reply")
	if err == nil {
		t.Fatal("expected error")
	}
	if spy.addCalls != 0 {
		t.Fatalf("Add called %d times, want 0", spy.addCalls)
	}
}

func TestHandleMessage_runtimeSkillsDisabled_toolPreselectionOnly(t *testing.T) {
	// Covers AC-13.008
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "skills.sqlite")
	st, err := sqlitevec.NewWithTable(db, 4, sqlitevec.TableSkills)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &runtimeskills.Package{
		ID: "hidden", Name: "HiddenSkill", Description: "unique skill phrase xyz",
		Body: "SECRET_SKILL_BODY", Tools: []string{},
	}
	if err := skillindex.Build(ctx, []*runtimeskills.Package{pkg}, constEmb4{}, st); err != nil {
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
	toolVec := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo"}}}
	ti := &mockToolIndex{store: toolVec, ready: true}

	rs := &config.RuntimeSkillsConfig{
		Enabled: false, MaxSkillsPerTurn: 2, ToolVectorTopKCap: 10,
	}
	var toolNames []string
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
		if opts != nil {
			for _, td := range opts.Tools {
				toolNames = append(toolNames, td.Name)
			}
		}
		return &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}, nil
	}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		skillIndex:                 skIdx,
		embedder:                   constEmb4{},
		runtimeSkillsCfg:           rs,
		skillPackagesByID:          map[string]*runtimeskills.Package{"hidden": pkg},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     testDiscardLogger(),
		firstProviderSupportsTools: true,
	}
	_, err = h.HandleMessage(ctx, 1, "unique skill phrase xyz")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.lastMessages[0].Content
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

func TestHandleMessage_hermesBlockBeforeRetrievedMarkers(t *testing.T) {
	// REQ-13.016: dynamic tail ordering with text-based (Hermes) tool path
	ctx := context.Background()
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"t1": {
				ID: "t1", IndexText: "Hermes tool one line", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	toolVec := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "t1"}}}
	ti := &mockToolIndex{store: toolVec, ready: true}
	memVec := &mockVectorStore{searchResults: []vector.SearchResult{{Text: "mem line"}}}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		vectorStore:                memVec,
		embedder:                   constEmb4{},
		logger:                     testDiscardLogger(),
		maxDynamicSystemRunes:      4000,
		vectorSearchTopK:           5,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	_, err := h.HandleMessage(ctx, 1, "q")
	if err != nil {
		t.Fatal(err)
	}
	sys := provider.lastMessages[0].Content
	iH := strings.Index(sys, promptmarkers.BeginHermesToolFormat)
	iR := strings.Index(sys, promptmarkers.BeginRetrievedContext)
	if iH < 0 || iR < 0 || iH >= iR {
		t.Fatalf("want Hermes block before retrieved: hermes@%d retrieved@%d", iH, iR)
	}
}

func TestHandleMessage_toolRound_preservesSystemContent(t *testing.T) {
	// Covers AC-13.012
	ctx := context.Background()
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "out"}
	var firstSys string
	call := 0
	provider := &mockProvider{}
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
	toolVec := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo"}}}
	ti := &mockToolIndex{store: toolVec, ready: true}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		embedder:                   constEmb4{},
		nodeRunner:                 runner,
		logger:                     testDiscardLogger(),
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		firstProviderSupportsTools: true,
	}
	_, err := h.HandleMessage(ctx, 1, "hi")
	if err != nil {
		t.Fatal(err)
	}
}
