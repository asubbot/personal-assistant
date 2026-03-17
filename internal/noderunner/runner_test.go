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

// Covers AC-01.008 (US-04): RunOnNode does not execute empty/whitespace command; returns error.
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

// Covers AC-01.007, AC-01.008 (US-04): command not on allowlist is not executed; RunOnNode returns error.
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

// Covers AC-01.010 (US-04): multiple nodes — runner uses correct node ID per call (dedicated user per node from config).
func TestRunOnNode_twoNodes_eachUsesCorrectNodeID(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("uptime\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"node_a": {
				Host:                 "localhost",
				DedicatedUser:        "alice",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key_a")},
				CommandAllowlistPath: allowlistPath,
			},
			"node_b": {
				Host:                 "localhost",
				DedicatedUser:        "bob",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key_b")},
				CommandAllowlistPath: allowlistPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	r := New(cfg, al, slog.Default())

	var calls []struct{ nodeID, command string }
	r.SetExecutor(&mockExecutor{
		execFunc: func(ctx context.Context, nodeID, command string) ([]byte, []byte, error) {
			calls = append(calls, struct{ nodeID, command string }{nodeID, command})
			return []byte("ok"), nil, nil
		},
	})

	_, err = r.RunOnNode(context.Background(), "node_a", "uptime")
	if err != nil {
		t.Fatalf("RunOnNode(node_a): %v", err)
	}
	_, err = r.RunOnNode(context.Background(), "node_b", "uptime")
	if err != nil {
		t.Fatalf("RunOnNode(node_b): %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("executor called %d times, want 2", len(calls))
	}
	if calls[0].nodeID != "node_a" || calls[0].command != "uptime" {
		t.Errorf("first call: nodeID=%q command=%q, want node_a uptime", calls[0].nodeID, calls[0].command)
	}
	if calls[1].nodeID != "node_b" || calls[1].command != "uptime" {
		t.Errorf("second call: nodeID=%q command=%q, want node_b uptime", calls[1].nodeID, calls[1].command)
	}
}

// Covers AC-01.007 (US-04): RunOnNode invokes executor with node ID and command when allowlist allows.
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
