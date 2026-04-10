package skillindex

import (
	"context"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/runtimeskills"
	"pa/internal/vector"
	"sync/atomic"
)

// Index holds the skill vector store and ready flag (EP-013).
type Index struct {
	store vector.Store
	ready atomic.Bool
}

// NewIndex returns an index backed by store. Ready is false until Build succeeds.
func NewIndex(store vector.Store) *Index {
	return &Index{store: store}
}

// Store returns the underlying vec_skills store.
func (x *Index) Store() vector.Store {
	return x.store
}

// Ready reports whether Build completed successfully.
func (x *Index) Ready() bool {
	return x.ready.Load()
}

// SetReady sets the ready flag (tests / synchronous build).
func (x *Index) SetReady(v bool) {
	x.ready.Store(v)
}

// Close closes the store.
func (x *Index) Close() error {
	if x.store == nil {
		return nil
	}
	err := x.store.Close()
	x.store = nil
	return err
}

// Build clears the store and embeds all packages.
func Build(ctx context.Context, pkgs []*runtimeskills.Package, embedder embedding.Embedder, store vector.Store) error {
	if len(pkgs) == 0 || embedder == nil || store == nil {
		return nil
	}
	if err := store.Clear(ctx); err != nil {
		return fmt.Errorf("skillindex: clear: %w", err)
	}
	for _, p := range pkgs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		text := p.EmbeddingText()
		emb, err := embedder.Embed(ctx, text)
		if err != nil {
			return err
		}
		if err := store.Add(ctx, p.ID, emb, text); err != nil {
			return err
		}
	}
	return nil
}

// BuildAndSetReady runs Build then sets ready on success.
func (x *Index) BuildAndSetReady(ctx context.Context, pkgs []*runtimeskills.Package, embedder embedding.Embedder) error {
	err := Build(ctx, pkgs, embedder, x.store)
	if err == nil {
		x.ready.Store(true)
	}
	return err
}
