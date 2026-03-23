package toolindex

import (
	"context"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/toolcatalog"
	"time"
)

// UpsertToolEmbedding embeds one tool and stores it in the tool vector index (best-effort after create_tool; REQ-09.012).
// Deletes any existing row for the tool id before insert. Returns nil when idx or embedder is nil.
func UpsertToolEmbedding(ctx context.Context, idx *Index, embedder embedding.Embedder, t *toolcatalog.Tool) error {
	if idx == nil || embedder == nil || t == nil {
		return nil
	}
	st := idx.Store()
	if st == nil {
		return nil
	}
	text := toolIndexText(t)
	emb, err := embedWithRetry(ctx, embedder, text)
	if err != nil {
		return fmt.Errorf("toolindex: embed tool %q: %w", t.ID, err)
	}
	if err := st.Delete(ctx, t.ID); err != nil {
		return fmt.Errorf("toolindex: delete old embedding %q: %w", t.ID, err)
	}
	if err := st.Add(ctx, t.ID, emb, text); err != nil {
		return fmt.Errorf("toolindex: add embedding %q: %w", t.ID, err)
	}
	return nil
}

func embedWithRetry(ctx context.Context, embedder embedding.Embedder, text string) ([]float32, error) {
	var emb []float32
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(150 * time.Millisecond):
			}
		}
		emb, err = embedder.Embed(ctx, text)
		if err == nil {
			return emb, nil
		}
	}
	return nil, err
}
