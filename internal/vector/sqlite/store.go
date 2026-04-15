// Package sqlite provides the default vector store implementation using SQLite + sqlite-vec (CGO).
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"pa/internal/vector"
	"sync"

	vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

// Table names for the vector DB. Same DB file can hold multiple tables (e.g. memory and tool index).
const (
	TableTools  = "vec_tools"
	TableSkills = "vec_skills"
	// EP-016: split memory vectors (same DB file as tools/skills).
	TableSummaries = "vec_summaries"
	TableTurns     = "vec_turns"
	TableNotes     = "vec_notes"
)

var autoOnce sync.Once

func initAuto() {
	autoOnce.Do(func() { vec.Auto() })
}

// Store implements vector.Store using SQLite and the sqlite-vec extension.
type Store struct {
	db         *sql.DB
	dimensions int
	table      string
}

// validateTableName returns an error if table is empty or contains invalid characters (only alphanumeric and underscore allowed).
func validateTableName(table string) error {
	if table == "" {
		return fmt.Errorf("vector/sqlite: table name is required")
	}
	for _, r := range table {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return fmt.Errorf("vector/sqlite: table name must be alphanumeric or underscore, got %q", table)
	}
	return nil
}

// NewWithTable opens or creates a SQLite database at dbPath and returns a vector store for the given table.
// Use TableSummaries/TableTurns/TableNotes for memory stores and TableTools for tool index.
// Same dbPath allows multiple tables in one DB file.
func NewWithTable(dbPath string, dimensions int, table string) (*Store, error) {
	if dimensions <= 0 {
		return nil, fmt.Errorf("vector/sqlite: dimensions must be positive, got %d", dimensions)
	}
	if err := validateTableName(table); err != nil {
		return nil, err
	}
	initAuto()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("vector/sqlite: open db: %w", err)
	}
	s := &Store{db: db, dimensions: dimensions, table: table}
	if err := s.createTable(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) createTable() error {
	// vec0 virtual table: embedding vector + auxiliary columns for id and content
	query := fmt.Sprintf(
		`CREATE VIRTUAL TABLE IF NOT EXISTS %s USING vec0(
			embedding float[%d],
			+id text,
			+content text
		)`,
		s.table, s.dimensions,
	)
	_, err := s.db.ExecContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("vector/sqlite: create table: %w", err)
	}
	return nil
}

// Add implements vector.Store.
func (s *Store) Add(ctx context.Context, id string, embedding []float32, text string) error {
	if len(embedding) != s.dimensions {
		return fmt.Errorf("vector/sqlite: embedding length %d != dimensions %d", len(embedding), s.dimensions)
	}
	blob, err := vec.SerializeFloat32(embedding)
	if err != nil {
		return fmt.Errorf("vector/sqlite: serialize: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s(embedding, id, content) VALUES (?, ?, ?)", s.table),
		blob, id, text,
	)
	if err != nil {
		return fmt.Errorf("vector/sqlite: insert: %w", err)
	}
	return nil
}

// Delete implements vector.Store. Removes the row with the given id. No-op if id does not exist.
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.table),
		id,
	)
	if err != nil {
		return fmt.Errorf("vector/sqlite: delete: %w", err)
	}
	return nil
}

// Clear implements vector.Store. Removes all rows from this table.
func (s *Store) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", s.table))
	if err != nil {
		return fmt.Errorf("vector/sqlite: clear: %w", err)
	}
	return nil
}

// Exists implements vector.Store.
func (s *Store) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	var one int
	//nolint:gosec // G201: table identifier validated in NewWithTable
	q := fmt.Sprintf("SELECT 1 FROM %s WHERE id = ? LIMIT 1", s.table)
	err := s.db.QueryRowContext(ctx, q, id).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("vector/sqlite: exists: %w", err)
	}
	return true, nil
}

// Search implements vector.Store.
func (s *Store) Search(ctx context.Context, queryEmbedding []float32, topK int) ([]vector.SearchResult, error) {
	if len(queryEmbedding) != s.dimensions {
		return nil, fmt.Errorf("vector/sqlite: query length %d != dimensions %d", len(queryEmbedding), s.dimensions)
	}
	if topK <= 0 {
		return nil, fmt.Errorf("vector/sqlite: topK must be positive, got %d", topK)
	}
	blob, err := vec.SerializeFloat32(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("vector/sqlite: serialize query: %w", err)
	}
	// Table name is validated in NewWithTable (alphanumeric + underscore only); not user input.
	//nolint:gosec // G201: table identifier is validated, not interpolated from user input
	query := fmt.Sprintf(
		"SELECT rowid, id, content, distance FROM %s WHERE embedding MATCH ? AND k = ?",
		s.table,
	)
	rows, err := s.db.QueryContext(ctx, query, blob, topK)
	if err != nil {
		return nil, fmt.Errorf("vector/sqlite: search: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var results []vector.SearchResult
	for rows.Next() {
		var rowid int64
		var id, content string
		var distance float64
		if err := rows.Scan(&rowid, &id, &content, &distance); err != nil {
			return nil, fmt.Errorf("vector/sqlite: scan: %w", err)
		}
		results = append(results, vector.SearchResult{ID: id, Text: content, Score: distance})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vector/sqlite: rows: %w", err)
	}
	return results, nil
}

// Close implements vector.Store.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
