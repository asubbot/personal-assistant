package tools

import (
	"context"
	"os"
	"pa/internal/config"
	"pa/internal/toolcatalog"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"
)

// Covers AC-09.008–013, AC-09.017: create_tool persists and updates runtime catalog.
func TestCreateToolTool_Run_success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	if err := os.WriteFile(path, []byte("tools: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n": {
				Host:                 "h",
				DedicatedUser:        "u",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/k"},
				CommandAllowlistPath: "/a",
			},
		},
	}
	var mu sync.Mutex
	ct := NewCreateTool(&mu, cat, path, cfg, nil, nil, nil)
	good := `docker run --rm --network bridge --memory="256m" --cpus="0.5" timeout 30s alpine:latest echo hi`
	out, err := ct.Run(context.Background(), map[string]any{
		"id":         "t_ep009",
		"index_text": "test tool",
		"template":   good,
		"node_id":    "n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("empty result")
	}
	if cat.Tools["t_ep009"] == nil {
		t.Fatal("runtime catalog not updated")
	}
}

// Covers AC-09.010: duplicate id rejected.
func TestCreateToolTool_Run_duplicateID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	if err := os.WriteFile(path, []byte("tools: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{
		"dup": {ID: "dup", IndexText: "x", Template: "x", NodeID: "n"},
	}}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n": {Host: "h", DedicatedUser: "u", Auth: config.NodeAuth{PrivateKeyPath: "/k"}, CommandAllowlistPath: "/a"},
		},
	}
	var mu sync.Mutex
	ct := NewCreateTool(&mu, cat, path, cfg, nil, nil, nil)
	good := `docker run --rm --network bridge --memory="256m" --cpus="0.5" timeout 30s alpine:latest echo hi`
	_, err := ct.Run(context.Background(), map[string]any{
		"id": "dup", "index_text": "y", "template": good, "node_id": "n",
	})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

// Covers AC-09.017: secret pattern match rejects persist.
func TestCreateToolTool_Run_secretRejected(t *testing.T) {
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
		CreateToolSecretRegex: []*regexp.Regexp{regexp.MustCompile(`api[_-]?key`)},
	}
	var mu sync.Mutex
	ct := NewCreateTool(&mu, cat, path, cfg, nil, nil, nil)
	good := `docker run --rm --network bridge --memory="256m" --cpus="0.5" timeout 30s alpine:latest echo hi`
	_, err := ct.Run(context.Background(), map[string]any{
		"id": "t1", "index_text": "has api_key: secret", "template": good, "node_id": "n",
	})
	if err == nil {
		t.Fatal("expected secret rejection")
	}
}

// Supporting AC-09.015: create_tool path stays bounded (sanity timing).
func TestCreateToolTool_Run_durationBudget(t *testing.T) {
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
	var mu sync.Mutex
	ct := NewCreateTool(&mu, cat, path, cfg, nil, nil, nil)
	good := `docker run --rm --network bridge --memory="256m" --cpus="0.5" timeout 30s alpine:latest echo hi`
	start := time.Now()
	_, err := ct.Run(context.Background(), map[string]any{
		"id": "bench_id", "index_text": "x", "template": good, "node_id": "n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("create_tool took %v, expected <1s on fast FS", time.Since(start))
	}
}
