package embedding

import "context"

// Embedder produces vector embeddings for text (used by the vector store for semantic search).
type Embedder interface {
	// Embed returns the embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
}
