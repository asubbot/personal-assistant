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

const tableName = "vec_items"

var autoOnce sync.Once

func initAuto() {
	autoOnce.Do(func() { vec.Auto() })
}

// Store implements vector.Store using SQLite and the sqlite-vec extension.
type Store struct {
	db         *sql.DB
	dimensions int
}

// New opens or creates a SQLite database at dbPath and returns a vector store.
// dimensions is the fixed size of embedding vectors (e.g. 384, 1536).
func New(dbPath string, dimensions int) (*Store, error) {
	if dimensions <= 0 {
		return nil, fmt.Errorf("vector/sqlite: dimensions must be positive, got %d", dimensions)
	}
	initAuto()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("vector/sqlite: open db: %w", err)
	}
	s := &Store{db: db, dimensions: dimensions}
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
		tableName, s.dimensions,
	)
	_, err := s.db.Exec(query)
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
		fmt.Sprintf("INSERT INTO %s(embedding, id, content) VALUES (?, ?, ?)", tableName),
		blob, id, text,
	)
	if err != nil {
		return fmt.Errorf("vector/sqlite: insert: %w", err)
	}
	return nil
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
	query := fmt.Sprintf(
		"SELECT rowid, id, content, distance FROM %s WHERE embedding MATCH ? AND k = ?",
		tableName,
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
