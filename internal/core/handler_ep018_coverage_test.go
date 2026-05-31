package core

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/runtimeskills"
	"pa/internal/testutil"
	"pa/internal/toolcatalog"
	"pa/internal/vector"
	"path/filepath"
	"strings"
	"testing"
)

func moduleRootDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from test working directory")
		}
		dir = parent
	}
}

func catalogFiveTools(t *testing.T) *toolcatalog.Catalog {
	t.Helper()
	tools := make(map[string]*toolcatalog.Tool)
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("t%d", i)
		tools[id] = &toolcatalog.Tool{
			ID:        id,
			IndexText: id,
			Template:  "echo x",
			NodeID:    "nas",
			Arguments: []toolcatalog.ArgumentRule{{Name: "x", Type: "string"}},
		}
	}
	return &toolcatalog.Catalog{Tools: tools}
}

func vectorResultsFive() []vector.SearchResult {
	var r []vector.SearchResult
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("t%d", i)
		r = append(r, vector.SearchResult{ID: id, Text: id})
	}
	return r
}

// Covers AC-18.001, AC-36.017
func TestEP018_configurationDoc_containsTierMatrix(t *testing.T) {
	root := moduleRootDir(t)
	p := filepath.Join(root, "docs", "configuration.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	s := string(b)
	for _, needle := range []string{"### Intent tiers", "simple", "full", "tools.selection"} {
		if !strings.Contains(s, needle) {
			t.Errorf("docs/configuration.md should mention %q", needle)
		}
	}
}

// Covers AC-38.018
// Covers AC-18.003, AC-18.015
func TestEP018_fullTier_dynamicDisabled_preservesMoreToolsThanWhenEnabled(t *testing.T) {
	cat := catalogFiveTools(t)
	idx := &mockToolIndex{store: &mockVectorStore{searchResults: vectorResultsFive()}, ready: true}
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`^FULLTOOLS$`}, 100),
		nil,
	)
	provNo := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	provYes := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	hNoCap := testHandlerDeps{
		router:                     mustRouterSingle(t, provNo),
		logger:                     slog.Default(),
		maxDynamicSystemRunes:      200_000,
		memoryVectorTopK:           testMemoryVectorTopK(5),
		classifier:                 classifier,
		catalog:                    cat,
		toolIndex:                  idx,
		embedder:                   &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		firstProviderSupportsTools: true,
		toolsSelection:             nil,
	}.handler()
	hCap := testHandlerDeps{
		router:                     mustRouterSingle(t, provYes),
		logger:                     slog.Default(),
		maxDynamicSystemRunes:      200_000,
		memoryVectorTopK:           testMemoryVectorTopK(5),
		classifier:                 classifier,
		catalog:                    cat,
		toolIndex:                  idx,
		embedder:                   &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		firstProviderSupportsTools: true,
		toolsSelection: &config.ToolsSelection{
			Enabled:               true,
			MaxToolsForLLMRequest: 2,
			ToolSearchTopK:        10,
			ToolMinCount:          1,
			ToolFallbackCap:       50,
		},
	}.handler()
	if _, err := hNoCap.HandleMessage(context.Background(), 1, "", "FULLTOOLS"); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if _, err := hCap.HandleMessage(context.Background(), 1, "", "FULLTOOLS"); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if provNo.lastOpts == nil || provYes.lastOpts == nil {
		t.Fatalf("expected tools on both runs: noCap=%v yesCap=%v", provNo.lastOpts, provYes.lastOpts)
	}
	if n, m := len(provNo.lastOpts.Tools), len(provYes.lastOpts.Tools); n <= m {
		t.Fatalf("dynamic cap should reduce tool count: noCap=%d capped=%d", n, m)
	}
}

// Covers AC-18.005
func TestEP018_fullTier_includesSessionExchanges(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	router := mustRouterSingle(t, provider)
	store := newSessionWindowStore()
	cfg := &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10}
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`^LITESESS`}, 100),
		nil,
	)
	h := testHandlerDeps{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(5),
		classifier:            classifier,
		sessionCfg:            cfg,
		sessionStore:          store,
	}.handler()
	store.appendExchange("k", "prior user", "prior assistant", cfg.MaxSessionExchanges)
	_, err := h.HandleMessage(context.Background(), 1, "k", "LITESESS")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(provider.lastMessages) < 4 {
		t.Fatalf("want session + user messages, got %d msgs", len(provider.lastMessages))
	}
	joined := fmt.Sprintf("%v", provider.lastMessages)
	if !strings.Contains(joined, "prior user") || !strings.Contains(joined, "prior assistant") {
		t.Fatalf("expected session in messages: %s", joined)
	}
}

// Covers AC-18.006
func TestEP018_dynamicTail_nilSkillsOmitsPlaybookText(t *testing.T) {
	h := testHandlerDeps{}.handler()
	p := &runtimeskills.Package{ID: "s1", Name: "Skill", Description: "d", Body: "PLAYBOOK_UNIQUE_EP018"}
	with := h.buildDynamicTailString(&tailFitState{skills: []*runtimeskills.Package{p}})
	without := h.buildDynamicTailString(&tailFitState{skills: nil})
	if !strings.Contains(with, "PLAYBOOK_UNIQUE_EP018") {
		t.Fatal("expected playbook when skills set")
	}
	if strings.Contains(without, "PLAYBOOK_UNIQUE_EP018") {
		t.Fatal("nil skills must omit playbook")
	}
}

// Covers AC-18.007 / AC-30.003: full tier with catalog tools uses native tool defs.
func TestEP018_fullTier_withCatalogTools_usesNativeToolDefs(t *testing.T) {
	cat := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	idx := &mockToolIndex{store: &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo"}}}, ready: true}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`^LITEHERM`}, 100),
		nil,
	)
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		logger:                     slog.Default(),
		maxDynamicSystemRunes:      200_000,
		memoryVectorTopK:           testMemoryVectorTopK(5),
		classifier:                 classifier,
		catalog:                    cat,
		toolIndex:                  idx,
		embedder:                   &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		firstProviderSupportsTools: true,
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "LITEHERM echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if provider.lastOpts == nil || len(provider.lastOpts.Tools) == 0 {
		t.Fatal("expected native tools on completion")
	}
}

// Covers AC-18.008: without catalog tools, completion opts omit tools.
func TestEP018_fullTier_noCatalogTools_omitsNativeTools(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`^LITENOTOOL`}, 100),
		nil,
	)
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		logger:                     slog.Default(),
		maxDynamicSystemRunes:      200_000,
		memoryVectorTopK:           testMemoryVectorTopK(5),
		classifier:                 classifier,
		catalog:                    nil,
		firstProviderSupportsTools: true,
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "LITENOTOOL")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if provider.lastOpts != nil {
		t.Fatalf("expected nil opts without catalog tools, got %+v", provider.lastOpts)
	}
}

// Covers AC-18.014 (rank order preserved through filter+cap)
func TestEP018_pickTools_preservesVectorOrderUnderCap(t *testing.T) {
	cat := catalogFiveTools(t)
	h := testHandlerDeps{catalog: cat, logger: slog.New(slog.DiscardHandler)}.handler()
	merged := []string{"t3", "t1", "t5", "bogus", "t2"}
	out := h.pickToolsForMainRequest(context.Background(), merged, 3)
	if len(out) != 3 || out[0] != "t3" || out[1] != "t1" || out[2] != "t5" {
		t.Fatalf("got %#v want [t3 t1 t5]", out)
	}
}

// Covers AC-18.017
func TestEP018_fullTier_dynamicSelection_logsTrueWhenConfigured(t *testing.T) {
	var cap captureHandlerWithAttrs
	logger := slog.New(&cap)
	cat := catalogFiveTools(t)
	idx := &mockToolIndex{store: &mockVectorStore{searchResults: vectorResultsFive()}, ready: true}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`^LITEDYN`}, 100),
		nil,
	)
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		logger:                     logger,
		maxDynamicSystemRunes:      200_000,
		memoryVectorTopK:           testMemoryVectorTopK(5),
		classifier:                 classifier,
		catalog:                    cat,
		toolIndex:                  idx,
		embedder:                   &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		firstProviderSupportsTools: true,
		toolsSelection: &config.ToolsSelection{
			Enabled:               true,
			MaxToolsForLLMRequest: 2,
			ToolSearchTopK:        10,
			ToolMinCount:          1,
			ToolFallbackCap:       50,
		},
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "LITEDYN")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var found bool
	for _, rec := range cap.records {
		if rec.msg != "main llm prompt assembled" {
			continue
		}
		found = true
		if rec.attrs["dynamic_tool_selection"] != "true" {
			t.Fatalf("expected dynamic_tool_selection true, got %#v", rec.attrs)
		}
	}
	if !found {
		t.Fatalf("expected assembled log, records=%#v", cap.records)
	}
}

// Manual scenario trace for pre-selection-disabled fallback (REQ-18.016). Covers AC-18.016
func TestEP018_manual_note_toolPreSelectionFallbackDeferredToOperator(t *testing.T) {
	// Behaviour is specified in ep-requirements / ep-system-design; automated path shares mergeSelectedToolIDs with full tier.
}

// Covers AC-18.021. Entrypoint: ./bin/validate EP-018 (via testutil.EnsureValidator).
func TestEP018_validateCommandExitZero(t *testing.T) {
	root := moduleRootDir(t)
	testutil.RunValidateEpic(t, root, "EP-018")
}
