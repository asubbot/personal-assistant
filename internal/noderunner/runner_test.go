package noderunner

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"pa/internal/allowlist"
	"pa/internal/config"
	"pa/internal/core/toolfailure"
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

// Covers AC-04.029 (REQ-04.031): command with shell metacharacters rejected before allowlist/exec.
func TestRunOnNode_shellMetacharacters_rejected(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("*\n"), 0o600); err != nil {
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
	var execCalls int
	r.SetExecutor(&mockExecutor{
		execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
			execCalls++
			return nil, nil, nil
		},
	})
	for _, cmd := range []string{"uptime;id", "foo &", "a | b", "x\ny", "echo $(id)", "echo `x`"} {
		_, err := r.RunOnNode(context.Background(), "n1", cmd)
		if err == nil {
			t.Fatalf("RunOnNode(%q): want error", cmd)
		}
	}
	if execCalls != 0 {
		t.Errorf("executor must not run for rejected commands; calls=%d", execCalls)
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

// Smoke: RunOnNode errors still carry EP-006 escalation typing via escalationpolicy (full matrix in escalationpolicy tests).
func TestRunOnNode_escalationPolicy_smoke(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	allowPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	t.Run("empty_NoEscalate", func(t *testing.T) {
		t.Parallel()
		r := New(cfg, al, slog.Default())
		r.SetExecutor(&mockExecutor{execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
			t.Fatal("executor must not run")
			return nil, nil, nil
		}})
		_, runErr := r.RunOnNode(context.Background(), "n1", "   ")
		if runErr == nil {
			t.Fatal("expected error")
		}
		if toolfailure.QualifiesForEscalation(runErr) {
			t.Fatal("expected NoEscalate for empty command")
		}
	})

	t.Run("executor_error_MayEscalate", func(t *testing.T) {
		t.Parallel()
		r := New(cfg, al, slog.Default())
		r.SetExecutor(&mockExecutor{execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
			return nil, nil, errors.New("remote exec failed")
		}})
		_, runErr := r.RunOnNode(context.Background(), "n1", "echo hello")
		if runErr == nil {
			t.Fatal("expected error")
		}
		if !toolfailure.QualifiesForEscalation(runErr) {
			t.Fatal("expected MayEscalate for remote exec failure")
		}
	})
}

// Remote stderr/stdout from the node are appended to the exec error (and logs) so tools and operators see script output.
func TestRunOnNode_execError_includesRemoteStderr(t *testing.T) {
	dir := t.TempDir()
	allowPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	r := New(cfg, al, slog.Default())
	stderrLine := "Error: Speaker name '' is ambiguous within {'Bedroom'}"
	r.SetExecutor(&mockExecutor{execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
		return nil, []byte(stderrLine), errors.New("ssh: exec: Process exited with status 1")
	}})
	_, runErr := r.RunOnNode(context.Background(), "n1", "echo hello")
	if runErr == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(runErr.Error(), stderrLine) {
		t.Errorf("error should include remote stderr; got %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "stderr:") {
		t.Errorf("error should label stderr; got %v", runErr)
	}
	if !toolfailure.QualifiesForEscalation(runErr) {
		t.Fatal("expected MayEscalate for remote exec failure")
	}
}

func TestRunOnNode_execError_includesStdoutAndStderr(t *testing.T) {
	dir := t.TempDir()
	allowPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	r := New(cfg, al, slog.Default())
	r.SetExecutor(&mockExecutor{execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
		return []byte("partial out\n"), []byte("partial err\n"), errors.New("remote failed")
	}})
	_, runErr := r.RunOnNode(context.Background(), "n1", "echo hello")
	if runErr == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(runErr.Error(), "stdout:") || !strings.Contains(runErr.Error(), "partial out") {
		t.Errorf("error should include stdout; got %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "stderr:") || !strings.Contains(runErr.Error(), "partial err") {
		t.Errorf("error should include stderr; got %v", runErr)
	}
}

func TestRunOnNode_execError_truncatesLongRemoteStreams(t *testing.T) {
	dir := t.TempDir()
	allowPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	long := strings.Repeat("x", maxRemoteStreamBytes+100)
	r := New(cfg, al, slog.Default())
	r.SetExecutor(&mockExecutor{execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
		return nil, []byte(long), errors.New("exit 1")
	}})
	_, runErr := r.RunOnNode(context.Background(), "n1", "echo hello")
	if runErr == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(runErr.Error(), remoteStreamTruncatedSuffix) {
		t.Errorf("long stderr should be truncated in error: %v", runErr)
	}
	if strings.Count(runErr.Error(), "x") > maxRemoteStreamBytes+50 {
		t.Errorf("error should not include full long stderr")
	}
}

// REQ-01.026: DEBUG ssh exec output redacts remote stream fragments when SetLogRedactor is set.
func TestRunOnNode_debugLog_redactsRemoteStreams(t *testing.T) {
	dir := t.TempDir()
	allowPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := New(cfg, al, logger)
	r.SetLogRedactor(func(s string) string { return strings.ReplaceAll(s, "SECRETPAYLOAD", "[REDACTED]") })
	r.SetExecutor(&mockExecutor{execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
		return []byte("ok"), []byte("stderr line SECRETPAYLOAD\n"), nil
	}})
	_, err = r.RunOnNode(context.Background(), "n1", "echo hello")
	if err != nil {
		t.Fatalf("RunOnNode: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "[REDACTED]") || strings.Contains(out, "SECRETPAYLOAD") {
		t.Errorf("DEBUG log should redact stream attrs; got:\n%s", out)
	}
}

// Returned exec error keeps unredacted remote detail for tool/LLM; Error log attrs are redacted.
func TestRunOnNode_errorLog_redactsStreams_returnUnredacted(t *testing.T) {
	dir := t.TempDir()
	allowPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowPath, []byte("echo *\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
				Host:                 "localhost",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: filepath.Join(dir, "key")},
				CommandAllowlistPath: allowPath,
			},
		},
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	r := New(cfg, al, logger)
	r.SetLogRedactor(func(s string) string { return strings.ReplaceAll(s, "SECRETPAYLOAD", "[REDACTED]") })
	r.SetExecutor(&mockExecutor{execFunc: func(context.Context, string, string) ([]byte, []byte, error) {
		return nil, []byte("SECRETPAYLOAD"), errors.New("exit 1")
	}})
	_, runErr := r.RunOnNode(context.Background(), "n1", "echo hello")
	if runErr == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(runErr.Error(), "SECRETPAYLOAD") {
		t.Errorf("returned error should keep raw stderr for diagnostics; got %v", runErr)
	}
	logOut := buf.String()
	if !strings.Contains(logOut, "[REDACTED]") || strings.Contains(logOut, "SECRETPAYLOAD") {
		t.Errorf("Error log should redact stream attrs; got:\n%s", logOut)
	}
}
