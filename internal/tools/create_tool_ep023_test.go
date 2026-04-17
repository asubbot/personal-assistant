package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"pa/internal/config"
	"pa/internal/toolcatalog"
	"pa/internal/toolindex"
	"pa/internal/vector"
	"path/filepath"
	"sync"
	"testing"
)

// errEmbedder always fails (Covers AC-23.007).
type errEmbedder struct{}

func (errEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("injected embed failure")
}

// failAddStore implements vector.Store; Add fails after Delete succeeds (Upsert path).
type failAddStore struct {
	deleted bool
}

func (failAddStore) Clear(context.Context) error { return nil }
func (failAddStore) Search(context.Context, []float32, int) ([]vector.SearchResult, error) {
	return nil, nil
}
func (failAddStore) Exists(context.Context, string) (bool, error) { return false, nil }
func (failAddStore) Close() error                                 { return nil }

func (f *failAddStore) Delete(_ context.Context, _ string) error {
	f.deleted = true
	return nil
}

func (f *failAddStore) Add(context.Context, string, []float32, string) error {
	return errors.New("injected add failure")
}

// Covers AC-23.007: embedding upsert failure rolls back catalog file and in-memory catalog.
func TestCreateToolTool_Run_embedFailureRollsBack(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	initial := []byte("tools: []\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n": {Host: "h", DedicatedUser: "u", Auth: config.NodeAuth{PrivateKeyPath: "/k"}, CommandAllowlistPath: "/a"},
		},
	}
	st := &failAddStore{}
	idx := toolindex.NewIndex(st)
	var mu sync.Mutex
	ct := NewCreateTool(&mu, cat, path, cfg, errEmbedder{}, idx, nil)
	good := `docker run --rm --network bridge alpine:latest timeout 30s echo hi`
	_, err := ct.Run(context.Background(), map[string]any{
		"id": "fail_embed", "index_text": "x", "template": good, "node_id": "n",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if cat.Tools["fail_embed"] != nil {
		t.Fatal("in-memory catalog should not retain tool after embed failure")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, initial) {
		t.Fatalf("catalog file not restored: %q", got)
	}
}

// Covers AC-23.006: vector store Add runs only after catalog file contains the new tool (persist before index).
func TestCreateToolTool_Run_embedSuccessAfterPersist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	if err := os.WriteFile(path, []byte("tools: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n": {Host: "h", DedicatedUser: "u", Auth: config.NodeAuth{PrivateKeyPath: "/k"}, CommandAllowlistPath: "/a"},
		},
	}
	st := &recordingAddStore{catalogPath: path}
	idx := toolindex.NewIndex(st)
	var mu sync.Mutex
	emb := &fixedEmbedder{vec: []float32{1, 0, 0, 0}}
	ct := NewCreateTool(&mu, cat, path, cfg, emb, idx, nil)
	good := `docker run --rm --network bridge alpine:latest timeout 30s echo hi`
	_, err := ct.Run(context.Background(), map[string]any{
		"id": "ok_order", "index_text": "x", "template": good, "node_id": "n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !st.sawAdd {
		t.Fatal("expected Add on vector store after successful create")
	}
}

type fixedEmbedder struct{ vec []float32 }

func (f *fixedEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return f.vec, nil
}

type recordingAddStore struct {
	catalogPath string
	sawAdd      bool
}

func (recordingAddStore) Delete(context.Context, string) error { return nil }
func (recordingAddStore) Clear(context.Context) error          { return nil }
func (recordingAddStore) Search(context.Context, []float32, int) ([]vector.SearchResult, error) {
	return nil, nil
}
func (recordingAddStore) Exists(context.Context, string) (bool, error) { return false, nil }
func (recordingAddStore) Close() error                                 { return nil }

func (r *recordingAddStore) Add(_ context.Context, id string, _ []float32, _ string) error {
	if r.catalogPath == "" {
		return errors.New("recordingAddStore: catalogPath not set")
	}
	cat, err := toolcatalog.Load(r.catalogPath)
	if err != nil {
		return fmt.Errorf("load catalog during Add: %w", err)
	}
	if cat.Tools[id] == nil {
		return fmt.Errorf("parsed catalog missing tool %q before vector Add", id)
	}
	r.sawAdd = true
	return nil
}
