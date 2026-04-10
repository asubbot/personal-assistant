package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDimensions = 4

// Covers AC-01.013 (US-07): NewWithTable rejects invalid dimensions (vector store interface).
func TestNewWithTable_dimensionsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	_, err := NewWithTable(path, 0, TableMemory)
	if err == nil {
		t.Fatal("expected error for dimensions 0")
	}
	_, err = NewWithTable(path, -1, TableMemory)
	if err == nil {
		t.Fatal("expected error for negative dimensions")
	}
}

// Covers AC-01.013, AC-01.014 (US-07): Add and Search return top-k by similarity (index maintained).
func TestStore_Add_Search_topK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := NewWithTable(path, testDimensions, TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
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

	store, err := NewWithTable(path, testDimensions, TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
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

// Clear removes all rows; memory and tools tables are independent.
func TestStore_Clear_removesAllRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := NewWithTable(path, testDimensions, TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Add(ctx, "a", []float32{1, 0, 0, 0}, "a"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Add(ctx, "b", []float32{0, 1, 0, 0}, "b"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	results, err := store.Search(ctx, []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("after Clear: want 0 results, got %d", len(results))
	}
}

// Covers AC-01.013 (US-07): Add rejects wrong embedding dimensions.
func TestStore_Add_wrongDimensions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	store, err := NewWithTable(path, testDimensions, TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
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

	store, err := NewWithTable(path, testDimensions, TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable: %v", err)
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
	store2, err := NewWithTable(path, testDimensions, TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable (reopen): %v", err)
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

// Covers AC-04.014: same DB file contains both memory table and tool table; search returns top-k tool ids.
func TestStore_TwoTables_SameDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")
	ctx := context.Background()

	storeMem, err := NewWithTable(path, testDimensions, TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable(memory): %v", err)
	}
	defer func() { _ = storeMem.Close() }()

	storeTools, err := NewWithTable(path, testDimensions, TableTools)
	if err != nil {
		t.Fatalf("NewWithTable(tools): %v", err)
	}
	defer func() { _ = storeTools.Close() }()

	twoTablesSameDB_addAndSearchMemory(t, ctx, storeMem)
	twoTablesSameDB_addAndSearchTools(t, ctx, storeTools)
	twoTablesSameDB_assertBothTablesInDB(t, ctx, path)
}

func twoTablesSameDB_addAndSearchMemory(t *testing.T, ctx context.Context, storeMem *Store) {
	t.Helper()
	vecMem := []float32{1, 0, 0, 0}
	if err := storeMem.Add(ctx, "mem-1", vecMem, "memory text"); err != nil {
		t.Fatalf("Add to memory: %v", err)
	}
	resultsMem, err := storeMem.Search(ctx, vecMem, 2)
	if err != nil {
		t.Fatalf("Search memory: %v", err)
	}
	if len(resultsMem) != 1 || resultsMem[0].ID != "mem-1" {
		t.Errorf("Search memory: got %v, want single result mem-1", resultsMem)
	}
}

func twoTablesSameDB_addAndSearchTools(t *testing.T, ctx context.Context, storeTools *Store) {
	t.Helper()
	vecTool := []float32{0, 1, 0, 0}
	if err := storeTools.Add(ctx, "tool-1", vecTool, "tool desc"); err != nil {
		t.Fatalf("Add to tools: %v", err)
	}
	if err := storeTools.Add(ctx, "tool-2", []float32{0, 0.9, 0.1, 0}, "another tool"); err != nil {
		t.Fatalf("Add tool-2: %v", err)
	}
	resultsTools, err := storeTools.Search(ctx, vecTool, 2)
	if err != nil {
		t.Fatalf("Search tools: %v", err)
	}
	if len(resultsTools) != 2 {
		t.Errorf("Search tools: got %d results, want 2", len(resultsTools))
	}
	ids := make(map[string]bool)
	for _, r := range resultsTools {
		ids[r.ID] = true
	}
	if !ids["tool-1"] || !ids["tool-2"] {
		t.Errorf("Search tools: got ids %v, want tool-1 and tool-2", ids)
	}
}

func twoTablesSameDB_assertBothTablesInDB(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open db for table check: %v", err)
	}
	defer func() { _ = db.Close() }()
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name IN (?, ?)", TableMemory, TableTools)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(tables) != 2 {
		t.Errorf("same DB should have 2 tables, got %v", tables)
	}
}

// NewWithTable rejects empty table name and invalid table name.
func TestNewWithTable_validation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vec.db")

	_, err := NewWithTable(path, testDimensions, "")
	if err == nil {
		t.Fatal("NewWithTable(empty table): expected error")
	}
	if !strings.Contains(err.Error(), "table name is required") {
		t.Errorf("NewWithTable(empty table): error = %v", err)
	}

	_, err = NewWithTable(path, testDimensions, "vec-items")
	if err == nil {
		t.Fatal("NewWithTable(invalid table name with hyphen): expected error")
	}
	if !strings.Contains(err.Error(), "alphanumeric") {
		t.Errorf("NewWithTable(invalid table): error = %v", err)
	}

	_, err = NewWithTable(path, 0, TableTools)
	if err == nil {
		t.Fatal("NewWithTable(dimensions 0): expected error")
	}
}
