package vector

import "context"

// Store is the pluggable vector store interface (REQ-01.007).
// Implementations: SQLite+sqlite-vec (default), vecgo, chromem-go.
type Store interface {
	// Add inserts a document with the given id, embedding vector, and text.
	// The text is stored for retrieval in Search results.
	Add(ctx context.Context, id string, embedding []float32, text string) error
	// Delete removes the document with the given id. No-op if the id does not exist.
	Delete(ctx context.Context, id string) error
	// Search returns the top-k nearest neighbors for the query embedding.
	// Score is distance (lower is closer). Order is by distance ascending.
	Search(ctx context.Context, queryEmbedding []float32, topK int) ([]SearchResult, error)
	Close() error
}

// SearchResult is one hit from a vector search.
type SearchResult struct {
	ID    string  // document id
	Text  string  // stored text
	Score float64 // distance (lower = closer)
}
