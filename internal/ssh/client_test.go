package ssh

import (
	"context"
	"errors"
	"os"
	"pa/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewClient_unknownNode_returnsError validates AC-006, AC-009: client uses only config; unknown node fails.
func TestNewClient_unknownNode_returnsError(t *testing.T) {
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"nas": {
				Host:                 "192.168.1.10",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/key"},
				CommandAllowlistPath: "/allowlist.txt",
			},
		},
	}
	ctx := context.Background()

	_, err := NewClient(ctx, cfg, "nonexistent")
	if err == nil {
		t.Fatal("NewClient(nonexistent): expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("NewClient(nonexistent): error = %v, want 'not found'", err)
	}
}

// Covers AC-006 (US-03): client uses only configured credentials; missing key file returns error.
func TestNewClient_missingKeyFile_returnsError(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "missing_key")},
				CommandAllowlistPath: "/a.txt",
			},
		},
	}
	ctx := context.Background()

	_, err := NewClient(ctx, cfg, "n1")
	if err == nil {
		t.Fatal("NewClient(missing key file): expected error")
	}
	if !strings.Contains(err.Error(), "read private key") && !errors.Is(err, os.ErrNotExist) {
		t.Errorf("NewClient: error = %v", err)
	}
}

// No AC: SSH client parse error for non-PEM key file returns error.
func TestNewClient_invalidKeyFile_returnsError(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "badkey")
	if err := os.WriteFile(keyPath, []byte("not a valid PEM key"), 0o600); err != nil {
		t.Fatalf("write test key: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: keyPath},
				CommandAllowlistPath: "/a.txt",
			},
		},
	}
	ctx := context.Background()

	_, err := NewClient(ctx, cfg, "n1")
	if err == nil {
		t.Fatal("NewClient(invalid key): expected error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("NewClient: error = %v, want parse error", err)
	}
}
