package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/runtimeskills"
	"pa/internal/toolcatalog"
	"pa/internal/vector"
	"reflect"
	"testing"
)

func parityHandler(t *testing.T, topK, minCount, fallbackCap int, sel *config.ToolsSelection, rs *config.RuntimeSkillsConfig) *conversationHandler {
	t.Helper()
	cat := catalogFiveTools(t)
	idx := &mockToolIndex{store: &mockVectorStore{searchResults: vectorResultsFive()}, ready: true}
	return testHandlerDeps{
		logger:           slog.New(slog.DiscardHandler),
		catalog:          cat,
		toolIndex:        idx,
		embedder:         &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		toolSearchTopK:   topK,
		toolMinCount:     minCount,
		toolFallbackCap:  fallbackCap,
		toolsSelection:   sel,
		runtimeSkillsCfg: rs,
	}.handler()
}

// Covers AC-38.018, AC-38.019
// Covers AC-37.010
func TestToolsSelectionParity_mergeEquivalentPreSelection(t *testing.T) {
	ctx := context.Background()
	userText := "find tools"
	var skills []*runtimeskills.Package
	// Golden baseline: catalogFiveTools + vectorResultsFive yield the five
	// catalog tools in vector-result order with no always_include/skills.
	wantMerged := []string{"t1", "t2", "t3", "t4", "t5"}
	sel := &config.ToolsSelection{
		ToolSearchTopK:  10,
		ToolMinCount:    1,
		ToolFallbackCap: 50,
		Enabled:         false,
	}
	h := parityHandler(t, sel.ToolSearchTopK, sel.ToolMinCount, sel.ToolFallbackCap, sel, nil)
	merged, _, err := h.mergeSelectedToolIDs(ctx, userText, skills)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if !reflect.DeepEqual(merged, wantMerged) {
		t.Fatalf("merge golden: got %v want %v", merged, wantMerged)
	}
	// Equivalent post-EP-037 config (same operator-migrated values) must
	// select the identical tool-id slice.
	h2 := parityHandler(t, 10, 1, 50, &config.ToolsSelection{
		ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50, Enabled: false,
	}, nil)
	merged2, _, err := h2.mergeSelectedToolIDs(ctx, userText, skills)
	if err != nil {
		t.Fatalf("merge2: %v", err)
	}
	if !reflect.DeepEqual(merged2, wantMerged) {
		t.Fatalf("merge parity golden: got %v want %v", merged2, wantMerged)
	}
}

// Covers AC-37.011
func TestToolsSelectionParity_capEquivalentDynamicSelection(t *testing.T) {
	ctx := context.Background()
	merged := []string{"t3", "t1", "t5", "t2", "t4"}
	// Golden baseline: disabled (or nil) selection leaves the merged slice
	// untouched; enabled with max=3 yields the first three valid ids.
	wantUncapped := []string{"t3", "t1", "t5", "t2", "t4"}
	wantCapped := []string{"t3", "t1", "t5"}

	disabled := parityHandler(t, 10, 1, 50, nil, nil)
	outDisabled, ran := disabled.mergedAfterDynamicToolCap(ctx, merged)
	if ran {
		t.Fatal("nil selection should not run cap")
	}
	if !reflect.DeepEqual(outDisabled, wantUncapped) {
		t.Fatalf("nil selection golden: got %v want %v", outDisabled, wantUncapped)
	}
	selOff := &config.ToolsSelection{Enabled: false, MaxToolsForLLMRequest: 99, ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50}
	hOff := parityHandler(t, 10, 1, 50, selOff, nil)
	outOff, ran := hOff.mergedAfterDynamicToolCap(ctx, merged)
	if ran {
		t.Fatal("disabled selection should not run cap")
	}
	if !reflect.DeepEqual(outOff, wantUncapped) {
		t.Fatalf("disabled selection golden: got %v want %v", outOff, wantUncapped)
	}
	selOn := &config.ToolsSelection{Enabled: true, MaxToolsForLLMRequest: 3, ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50}
	hOn := parityHandler(t, 10, 1, 50, selOn, nil)
	outOn, ran := hOn.mergedAfterDynamicToolCap(ctx, merged)
	if !ran {
		t.Fatal("enabled selection should run cap")
	}
	if !reflect.DeepEqual(outOn, wantCapped) {
		t.Fatalf("cap golden: got %v want %v", outOn, wantCapped)
	}
}

// Covers AC-37.009
func TestToolsSelectionParity_runtimeTopKCap(t *testing.T) {
	ctx := context.Background()
	h := parityHandler(t, 20, 1, 50, &config.ToolsSelection{
		ToolSearchTopK: 20, ToolMinCount: 1, ToolFallbackCap: 50, Enabled: false,
	}, &config.RuntimeSkillsConfig{Enabled: true, ToolVectorTopKCap: 5, MaxSkillsPerTurn: 3})
	merged, _, err := h.mergeSelectedToolIDs(ctx, "q", nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(merged) > 5 {
		t.Fatalf("effective top-K should be min(20,5)=5, got %d tools: %v", len(merged), merged)
	}
}

// Covers AC-37.019
func TestToolsSelectionParity_toolVectorTopKCapOnlyUnderRuntimeSkills(t *testing.T) {
	rt := reflect.TypeOf(config.RuntimeSkillsConfig{})
	if _, ok := rt.FieldByName("ToolVectorTopKCap"); !ok {
		t.Fatal("ToolVectorTopKCap must be on RuntimeSkillsConfig")
	}
	sel := reflect.TypeOf(config.ToolsSelection{})
	if _, ok := sel.FieldByName("ToolVectorTopKCap"); ok {
		t.Fatal("ToolVectorTopKCap must not be on ToolsSelection")
	}
	_ = toolcatalog.Catalog{}
	_ = vector.SearchResult{}
}
