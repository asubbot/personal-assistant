package toolindex

import (
	"bytes"
	"context"
	"log/slog"
	"pa/internal/toolcatalog"
	"pa/internal/vector"
	"strings"
	"testing"
)

// mockSearchStore implements vector.Store and returns predefined Search results (for SelectToolIDs tests).
type mockSearchStore struct {
	searchResults []vector.SearchResult
	searchCalled  bool
}

func (m *mockSearchStore) Add(_ context.Context, _ string, _ []float32, _ string) error { return nil }
func (m *mockSearchStore) Delete(_ context.Context, _ string) error                     { return nil }
func (m *mockSearchStore) Clear(_ context.Context) error                                { return nil }
func (m *mockSearchStore) Close() error                                                 { return nil }

func (m *mockSearchStore) Search(_ context.Context, _ []float32, topK int) ([]vector.SearchResult, error) {
	m.searchCalled = true
	if len(m.searchResults) <= topK {
		return m.searchResults, nil
	}
	return m.searchResults[:topK], nil
}

func catalogWithIDs(ids ...string) *toolcatalog.Catalog {
	tools := make(map[string]*toolcatalog.Tool)
	for _, id := range ids {
		tools[id] = &toolcatalog.Tool{ID: id, IndexText: id, Template: "echo " + id, NodeID: "n", Arguments: nil}
	}
	return &toolcatalog.Catalog{Tools: tools}
}

// Covers AC-04.015: search returns top-k tool ids from vector store.
func TestSelectToolIDs_searchReturnsTopK(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{
		searchResults: []vector.SearchResult{
			{ID: "tool_a", Text: "a", Score: 0.1},
			{ID: "tool_b", Text: "b", Score: 0.2},
			{ID: "tool_c", Text: "c", Score: 0.3},
		},
	}
	catalog := catalogWithIDs("tool_a", "tool_b", "tool_c", "tool_d")
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}

	ids, err := SelectToolIDs(ctx, emb, store, true, catalog, "query", 2, 1, 50, nil)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("got %d ids, want 2 (topK=2)", len(ids))
	}
	if !store.searchCalled {
		t.Error("Search was not called")
	}
	// Result order is as returned by store
	if ids[0] != "tool_a" || ids[1] != "tool_b" {
		t.Errorf("got ids %v", ids)
	}
}

// Covers AC-04.016: fallback when store returns 0 results yields bounded non-empty list.
func TestSelectToolIDs_fallbackWhenStoreReturnsZero(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{searchResults: nil}
	catalog := catalogWithIDs("a", "b", "c")
	emb := &mockEmbedder{vec: []float32{0, 0, 0, 0}}

	ids, err := SelectToolIDs(ctx, emb, store, true, catalog, "query", 10, 1, 50, nil)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("fallback: got %d ids, want 3 (all catalog)", len(ids))
	}
	// Fallback returns sorted catalog IDs
	if ids[0] != "a" || ids[1] != "b" || ids[2] != "c" {
		t.Errorf("fallback ids: %v", ids)
	}
}

// Covers AC-04.016: fallback when len(ids) < minTools.
func TestSelectToolIDs_fallbackWhenBelowMinTools(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{
		searchResults: []vector.SearchResult{{ID: "only_one", Text: "x", Score: 0}},
	}
	catalog := catalogWithIDs("only_one", "other_a", "other_b")
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}

	ids, err := SelectToolIDs(ctx, emb, store, true, catalog, "query", 10, 3, 50, nil)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	// Fallback: sorted catalog, cap 50 → 3 ids
	if len(ids) != 3 {
		t.Errorf("got %d ids, want 3 (fallback sorted)", len(ids))
	}
	if ids[0] != "only_one" || ids[1] != "other_a" || ids[2] != "other_b" {
		t.Errorf("fallback ids: %v", ids)
	}
}

// Covers AC-04.015, AC-04.016: index not ready → fallback, no embed/store Search call.
func TestSelectToolIDs_indexNotReady_usesFallbackNoSearch(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{searchResults: []vector.SearchResult{{ID: "x", Text: "x", Score: 0}}}
	catalog := catalogWithIDs("c", "a", "b")
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}

	ids, err := SelectToolIDs(ctx, emb, store, false, catalog, "query", 10, 1, 50, nil)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	if store.searchCalled {
		t.Error("Search must not be called when index not ready")
	}
	// Fallback: sorted catalog [a, b, c]
	if len(ids) != 3 {
		t.Errorf("got %d ids, want 3", len(ids))
	}
	if ids[0] != "a" || ids[1] != "b" || ids[2] != "c" {
		t.Errorf("fallback ids: %v", ids)
	}
}

// Covers AC-04.019: empty catalog → empty result.
func TestSelectToolIDs_emptyCatalog_returnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{searchResults: nil}
	catalog := catalogWithIDs() // empty
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}

	ids, err := SelectToolIDs(ctx, emb, store, true, catalog, "query", 10, 1, 50, nil)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	if ids != nil {
		t.Errorf("empty catalog: got %v, want nil", ids)
	}
}

// Covers AC-04.019: nil catalog → empty result.
func TestSelectToolIDs_nilCatalog_returnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{searchResults: nil}
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}

	ids, err := SelectToolIDs(ctx, emb, store, true, nil, "query", 10, 1, 50, nil)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	if ids != nil {
		t.Errorf("nil catalog: got %v, want nil", ids)
	}
}

// Covers AC-04.015, AC-04.016: fallback cap respected (catalog 100 tools, cap 5).
func TestSelectToolIDs_fallbackCapRespected(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{searchResults: nil}
	ids100 := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids100[i] = string(rune('z'-i%26)) + string(rune('0'+i/26))
	}
	catalog := catalogWithIDs(ids100...)
	emb := &mockEmbedder{vec: []float32{0, 0, 0, 0}}

	ids, err := SelectToolIDs(ctx, emb, store, true, catalog, "query", 10, 1, 5, nil)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	if len(ids) != 5 {
		t.Errorf("fallback cap 5: got %d ids, want 5", len(ids))
	}
}

// Covers AC-04.019: DEBUG log emitted when fallback and logger with DEBUG enabled.
func TestSelectToolIDs_fallback_emitsDebugLog(t *testing.T) {
	ctx := context.Background()
	store := &mockSearchStore{searchResults: nil}
	catalog := catalogWithIDs("a", "b")
	emb := &mockEmbedder{vec: []float32{0, 0, 0, 0}}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	_, err := SelectToolIDs(ctx, emb, store, true, catalog, "query", 10, 1, 50, logger)
	if err != nil {
		t.Fatalf("SelectToolIDs: %v", err)
	}
	logOut := buf.String()
	if logOut == "" {
		t.Error("expected DEBUG log line when fallback with DEBUG logger")
	}
	if !strings.Contains(logOut, "tool pre-selection: using fallback") {
		t.Errorf("log should contain fallback message: %s", logOut)
	}
	if !strings.Contains(logOut, "empty result") {
		t.Errorf("log should contain reason attribute (empty result): %s", logOut)
	}
}
