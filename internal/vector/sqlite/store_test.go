package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const testDimensions = 4

// Covers AC-01.013 (US-07): New rejects invalid dimensions (vector store interface).
func TestNew_dimensionsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	_, err := New(path, 0)
	if err == nil {
		t.Fatal("expected error for dimensions 0")
	}
	_, err = New(path, -1)
	if err == nil {
		t.Fatal("expected error for negative dimensions")
	}
}

// Covers AC-01.013, AC-01.014 (US-07): Add and Search return top-k by similarity (index maintained).
func TestStore_Add_Search_topK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := New(path, testDimensions)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a few vectors; query similar to first
	vec1 := []float32{1, 0, 0, 0}
	vec2 := []float32{0, 1, 0, 0}
	vec3 := []float32{0.9, 0.1, 0, 0}
	if err := store.Add(ctx, "id1", vec1, "text one"); err != nil {
		t.Fatalf("Add id1: %v", err)
	}
	if err := store.Add(ctx, "id2", vec2, "text two"); err != nil {
		t.Fatalf("Add id2: %v", err)
	}
	if err := store.Add(ctx, "id3", vec3, "text three"); err != nil {
		t.Fatalf("Add id3: %v", err)
	}

	// Query with vector close to vec1 and vec3; expect top result to be id1 or id3 (closest)
	query := []float32{1, 0, 0, 0}
	results, err := store.Search(ctx, query, 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search topK=2: got %d results, want 2", len(results))
	}
	// First result should be closest (id1 or id3)
	if results[0].Score > results[1].Score {
		t.Errorf("results should be ordered by distance ascending: %v", results)
	}
	if results[0].Text == "" {
		t.Error("Search result should include stored text")
	}
}

// Supporting AC-01.013 (US-07): Delete removes by id; no-op if id does not exist.
func TestStore_Delete_removesById(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := New(path, testDimensions)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Add(ctx, "summary:day:2026-03-12", []float32{1, 0, 0, 0}, "Day summary."); err != nil {
		t.Fatalf("Add: %v", err)
	}
	results, _ := store.Search(ctx, []float32{1, 0, 0, 0}, 1)
	if len(results) != 1 {
		t.Fatalf("before Delete: expected 1 result, got %d", len(results))
	}
	if err := store.Delete(ctx, "summary:day:2026-03-12"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	results, _ = store.Search(ctx, []float32{1, 0, 0, 0}, 1)
	if len(results) != 0 {
		t.Errorf("after Delete: expected 0 results, got %d", len(results))
	}
	// No-op when id does not exist
	if err := store.Delete(ctx, "nonexistent"); err != nil {
		t.Errorf("Delete(nonexistent): %v", err)
	}
}

// Covers AC-01.013 (US-07): Add rejects wrong embedding dimensions.
func TestStore_Add_wrongDimensions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := New(path, testDimensions)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = store.Close() }()

	err = store.Add(ctx, "x", []float32{1, 2, 3}, "short")
	if err == nil {
		t.Fatal("expected error for embedding length != dimensions")
	}
}

// Covers AC-01.013 (US-07): index persisted to configured path; Search after reopen returns data.
func TestStore_Search_persisted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pa_vectors.sqlite")
	ctx := context.Background()

	store, err := New(path, testDimensions)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := store.Add(ctx, "a", []float32{1, 1, 1, 1}, "content a"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("database file not created at %s", path)
	}

	// Reopen and search
	store2, err := New(path, testDimensions)
	if err != nil {
		t.Fatalf("New (reopen): %v", err)
	}
	defer func() { _ = store2.Close() }()
	results, err := store2.Search(ctx, []float32{1, 1, 1, 1}, 1)
	if err != nil {
		t.Fatalf("Search after reopen: %v", err)
	}
	if len(results) != 1 || results[0].ID != "a" || results[0].Text != "content a" {
		t.Errorf("Search after reopen: got %v", results)
	}
}
