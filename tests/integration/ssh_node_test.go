//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"os"
	"pa/internal/allowlist"
	"pa/internal/config"
	"pa/internal/noderunner"
	"path/filepath"
	"strings"
	"testing"
)

// TestNodeRunner_integration_allowlistBlocksDisallowed (integration) validates AC-01.007, AC-01.008:
// when command is not on the allowlist, RunOnNode returns error and does not execute.
func TestNodeRunner_integration_allowlistBlocksDisallowed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("echo *\nsystemctl status *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"node1": {
				Host:                 "192.168.1.1",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowlistPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	r := noderunner.New(cfg, al, slog.Default())

	// Command not in allowlist
	_, err = r.RunOnNode(context.Background(), "node1", "rm -rf /")
	if err == nil {
		t.Fatal("RunOnNode(disallowed): expected error")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("error = %v, want 'not allowed'", err)
	}
}

// TestNodeRunner_integration_allowedCommand_usesConfigNode (integration) validates AC-01.006, AC-01.009:
// when command is allowlisted, RunOnNode uses the node from config (mock executor records nodeID/command).
func TestNodeRunner_integration_allowedCommand_usesConfigNode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"nas": {
				Host:                 "192.168.1.10",
				DedicatedUser:        "pa_nas",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowlistPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	r := noderunner.New(cfg, al, slog.Default())

	var gotNodeID, gotCmd string
	r.SetExecutor(&mockSSHExecutor{
		exec: func(ctx context.Context, nodeID, command string) ([]byte, []byte, error) {
			gotNodeID = nodeID
			gotCmd = command
			return []byte("ok"), nil, nil
		},
	})

	stdout, err := r.RunOnNode(context.Background(), "nas", "echo hello")
	if err != nil {
		t.Fatalf("RunOnNode: %v", err)
	}
	if stdout != "ok" {
		t.Errorf("stdout = %q", stdout)
	}
	if gotNodeID != "nas" || gotCmd != "echo hello" {
		t.Errorf("executor called with nodeID=%q command=%q, want nas / echo hello", gotNodeID, gotCmd)
	}
}

type mockSSHExecutor struct {
	exec func(ctx context.Context, nodeID, command string) (stdout, stderr []byte, err error)
}

func (m *mockSSHExecutor) Exec(ctx context.Context, nodeID, command string) ([]byte, []byte, error) {
	if m.exec != nil {
		return m.exec(ctx, nodeID, command)
	}
	return nil, nil, nil
}
