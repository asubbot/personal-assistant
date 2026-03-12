package noderunner

import (
	"context"
	"log/slog"
	"os"
	"pa/internal/allowlist"
	"pa/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-008 (US-04): RunOnNode does not execute empty/whitespace command; returns error.
func TestRunOnNode_emptyCommand_returnsError(t *testing.T) {
	dir := t.TempDir()
	ap := filepath.Join(dir, "allowlist.txt")
	_ = os.WriteFile(ap, []byte("echo *\n"), 0o600)
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {Host: "h", DedicatedUser: "u", Auth: config.NodeAuth{PrivateKeyPath: "/k"}, CommandAllowlistPath: ap},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	r := New(cfg, al, slog.Default())

	_, err = r.RunOnNode(context.Background(), "n1", "   ")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error = %v", err)
	}
}

// Covers AC-007, AC-008 (US-04): command not on allowlist is not executed; RunOnNode returns error.
func TestRunOnNode_allowlistDenies_returnsError(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
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
	r := New(cfg, al, slog.Default())

	// Command not in allowlist (we only allow "echo *")
	_, err = r.RunOnNode(context.Background(), "n1", "rm -rf /")
	if err == nil {
		t.Fatal("expected error when command not allowed")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("error = %v", err)
	}
}

// Covers AC-007 (US-04): RunOnNode invokes executor with node ID and command when allowlist allows.
func TestRunOnNode_allowedCommand_usesExecutorWhenSet(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
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
	r := New(cfg, al, slog.Default())

	var gotNodeID, gotCmd string
	r.SetExecutor(&mockExecutor{
		execFunc: func(ctx context.Context, nodeID, command string) ([]byte, []byte, error) {
			gotNodeID = nodeID
			gotCmd = command
			return []byte("hello"), nil, nil
		},
	})

	stdout, err := r.RunOnNode(context.Background(), "n1", "echo hello")
	if err != nil {
		t.Fatalf("RunOnNode: %v", err)
	}
	if stdout != "hello" {
		t.Errorf("stdout = %q, want hello", stdout)
	}
	if gotNodeID != "n1" || gotCmd != "echo hello" {
		t.Errorf("executor called with nodeID=%q command=%q, want n1 echo hello", gotNodeID, gotCmd)
	}
}

type mockExecutor struct {
	execFunc func(ctx context.Context, nodeID, command string) (stdout, stderr []byte, err error)
}

func (m *mockExecutor) Exec(ctx context.Context, nodeID, command string) ([]byte, []byte, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, nodeID, command)
	}
	return nil, nil, nil
}
