package skillindex

import (
	"context"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/vector"
)

// SearchSkillIDs returns up to topK skill package ids by semantic similarity.
func SearchSkillIDs(ctx context.Context, embedder embedding.Embedder, store vector.Store, query string, topK int) ([]string, error) {
	if store == nil || embedder == nil || topK < 1 {
		return nil, nil
	}
	q, err := embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	res, err := store.Search(ctx, q, topK)
	if err != nil {
		return nil, fmt.Errorf("skillindex search: %w", err)
	}
	ids := make([]string, 0, len(res))
	for _, r := range res {
		ids = append(ids, r.ID)
	}
	return ids, nil
}
