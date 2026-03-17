package embedding

import "context"

// Embedder produces vector embeddings for text (used by the vector store for semantic search).
type Embedder interface {
	// Embed returns the embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// BatchEmbedder produces embeddings for multiple texts in one call (used for tool index build, REQ-04.021).
// Implementations must return vectors in the same order as texts; empty texts yields nil, nil without error.
type BatchEmbedder interface {
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
