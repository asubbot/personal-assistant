package toolindex

import (
	"context"
	"errors"
	"pa/internal/toolcatalog"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"strings"
	"testing"
)

const testDimensions = 4

// mockEmbedder returns a fixed vector for any text (Covers AC-04.014: tool index build and search).
type mockEmbedder struct {
	vec []float32
}

func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return m.vec, nil
}

// Covers AC-04.014: tool index is built from catalog; Search returns expected tool ids.
func TestBuild_populatesStoreAndSearchReturnsToolIds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"tool_a":    {ID: "tool_a", IndexText: "Do A", Template: "echo a", NodeID: "n1", Arguments: nil, Triggers: []string{"a"}},
			"tool_b":    {ID: "tool_b", IndexText: "Do B", Template: "echo b", NodeID: "n1", Arguments: nil, Triggers: nil},
			"node_time": {ID: "node_time", IndexText: "Get node time", Template: "date", NodeID: "nas", Arguments: nil, Triggers: []string{"time"}},
		},
	}
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}

	err = Build(ctx, catalog, emb, store)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Search with same vector; should get our tools back (order may vary by distance)
	results, err := store.Search(ctx, []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Search: got %d results, want 3 tools indexed", len(results))
	}
	ids := make(map[string]bool)
	for _, r := range results {
		ids[r.ID] = true
	}
	for _, id := range []string{"tool_a", "tool_b", "node_time"} {
		if !ids[id] {
			t.Errorf("Search: missing expected tool id %q", id)
		}
	}
}

// Second Build clears vec_tools then repopulates; removed catalog ids no longer appear in Search.
// Covers AC-04.017: traceability for TestBuild_secondBuild_dropsStaleToolIds.
func TestBuild_secondBuild_dropsStaleToolIds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}
	catalog1 := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"stale": {ID: "stale", IndexText: "gone", Template: "echo", NodeID: "n", Arguments: nil},
			"kept":  {ID: "kept", IndexText: "stay", Template: "echo", NodeID: "n", Arguments: nil},
		},
	}
	if err := Build(ctx, catalog1, emb, store); err != nil {
		t.Fatalf("Build first: %v", err)
	}
	catalog2 := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"kept": {ID: "kept", IndexText: "stay", Template: "echo", NodeID: "n", Arguments: nil},
		},
	}
	if err := Build(ctx, catalog2, emb, store); err != nil {
		t.Fatalf("Build second: %v", err)
	}
	results, err := store.Search(ctx, []float32{1, 0, 0, 0}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, r := range results {
		if r.ID == "stale" {
			t.Errorf("Search: stale id %q should not exist after second Build", r.ID)
		}
	}
	if len(results) != 1 || results[0].ID != "kept" {
		t.Errorf("Search: got %+v, want single id kept", results)
	}
}

// Empty catalog after Build leaves vec_tools empty (no stale rows).
// Covers AC-04.017: traceability for TestBuild_emptyCatalog_clearsStore.
func TestBuild_emptyCatalog_clearsStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}
	if err := Build(ctx, &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"x": {ID: "x", IndexText: "X", Template: "echo", NodeID: "n", Arguments: nil},
		},
	}, emb, store); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := Build(ctx, &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}, emb, store); err != nil {
		t.Fatalf("Build empty: %v", err)
	}
	results, err := store.Search(ctx, []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("after empty catalog Build: want 0 results, got %d", len(results))
	}
}

// Covers AC-04.017: when Build completes synchronously, SetReady makes Index ready; when BuildAndSetReady runs, Ready becomes true.
func TestIndex_Ready_afterBuildAndSetReady(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	idx := NewIndex(store)
	if idx.Ready() {
		t.Error("new Index should not be ready")
	}

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"x": {ID: "x", IndexText: "X", Template: "echo x", NodeID: "n", Arguments: nil},
		},
	}
	emb := &mockEmbedder{vec: []float32{0, 0, 0, 1}}

	err = idx.BuildAndSetReady(ctx, catalog, emb)
	if err != nil {
		t.Fatalf("BuildAndSetReady: %v", err)
	}
	if !idx.Ready() {
		t.Error("after BuildAndSetReady: Ready() should be true")
	}

	idx.SetReady(false)
	if idx.Ready() {
		t.Error("after SetReady(false): Ready() should be false")
	}
	idx.SetReady(true)
	if !idx.Ready() {
		t.Error("after SetReady(true): Ready() should be true")
	}
}

// Build with nil catalog/store/embedder is a no-op (no panic).
// Covers AC-04.017: traceability for TestBuild_nilInputs_noOp.
func TestBuild_nilInputs_noOp(t *testing.T) {
	ctx := context.Background()
	err := Build(ctx, nil, &mockEmbedder{vec: []float32{1}}, nil)
	if err != nil {
		t.Errorf("Build(nil catalog, ...): %v", err)
	}
}

// mockBatchEmbedder implements both Embedder and BatchEmbedder (Covers AC-04.017: batch path).
type mockBatchEmbedder struct {
	vec []float32
}

func (m *mockBatchEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return m.vec, nil
}

func (m *mockBatchEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = m.vec
	}
	return out, nil
}

// Covers AC-04.017: Build uses batch path when embedder implements BatchEmbedder.
func TestBuild_usesBatchEmbedder_whenAvailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"a": {ID: "a", IndexText: "A", Template: "echo a", NodeID: "n", Arguments: nil},
			"b": {ID: "b", IndexText: "B", Template: "echo b", NodeID: "n", Arguments: nil},
		},
	}
	emb := &mockBatchEmbedder{vec: []float32{0, 1, 0, 0}}

	err = Build(ctx, catalog, emb, store)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	results, err := store.Search(ctx, []float32{0, 1, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search: got %d results, want 2", len(results))
	}
	ids := make(map[string]bool)
	for _, r := range results {
		ids[r.ID] = true
	}
	if !ids["a"] || !ids["b"] {
		t.Errorf("Search: missing ids, got %v", ids)
	}
}

// failingEmbedder returns an error on Embed (sequential path).
type failingEmbedder struct{}

func (f *failingEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, errors.New("embed failed")
}

// BuildAndSetReady when Build fails must not set ready; Ready() stays false.
// Covers AC-04.017: traceability for TestBuildAndSetReady_whenBuildFails_ReadyStaysFalse.
func TestBuildAndSetReady_whenBuildFails_ReadyStaysFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	idx := NewIndex(store)
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"x": {ID: "x", IndexText: "X", Template: "echo x", NodeID: "n", Arguments: nil},
		},
	}

	err = idx.BuildAndSetReady(ctx, catalog, &failingEmbedder{})
	if err == nil {
		t.Fatal("BuildAndSetReady: expected error, got nil")
	}
	if idx.Ready() {
		t.Error("Ready() should be false when Build failed")
	}
}

// batchEmbedderWrongLength returns fewer vectors than input texts (batch path).
type batchEmbedderWrongLength struct{}

func (b *batchEmbedderWrongLength) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{0}, nil
}

func (b *batchEmbedderWrongLength) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	// Return one vector instead of len(texts).
	if len(texts) == 0 {
		return nil, nil
	}
	return [][]float32{{0, 0, 0, 0}}, nil
}

// Build (batched path) when EmbedBatch returns wrong length returns error with "EmbedBatch result length".
// Covers AC-04.017: traceability for TestBuild_batched_embedBatchLengthMismatch_returnsError.
func TestBuild_batched_embedBatchLengthMismatch_returnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"a": {ID: "a", IndexText: "A", Template: "echo a", NodeID: "n", Arguments: nil},
			"b": {ID: "b", IndexText: "B", Template: "echo b", NodeID: "n", Arguments: nil},
		},
	}

	err = Build(ctx, catalog, &batchEmbedderWrongLength{}, store)
	if err == nil {
		t.Fatal("Build: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "EmbedBatch result length") {
		t.Errorf("Build: error = %v", err)
	}
}

// Build with canceled context returns context error (batched path).
// Covers AC-04.017: traceability for TestBuild_contextCanceled_batched_returnsContextError.
func TestBuild_contextCanceled_batched_returnsContextError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"a": {ID: "a", IndexText: "A", Template: "echo a", NodeID: "n", Arguments: nil},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = Build(ctx, catalog, &mockBatchEmbedder{vec: []float32{0, 0, 0, 0}}, store)
	if err == nil {
		t.Fatal("Build(canceled ctx): expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Build: err = %v, want context.Canceled", err)
	}
}

// Build with canceled context returns context error (sequential path).
// Covers AC-04.017: traceability for TestBuild_contextCanceled_sequential_returnsContextError.
func TestBuild_contextCanceled_sequential_returnsContextError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	store, err := sqlite.NewWithTable(path, testDimensions, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"a": {ID: "a", IndexText: "A", Template: "echo a", NodeID: "n", Arguments: nil},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = Build(ctx, catalog, &mockEmbedder{vec: []float32{0, 0, 0, 0}}, store)
	if err == nil {
		t.Fatal("Build(canceled ctx): expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Build: err = %v, want context.Canceled", err)
	}
}
