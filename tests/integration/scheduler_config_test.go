//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"os"
	"pa/internal/allowlist"
	"pa/internal/config"
	"pa/internal/noderunner"
	"pa/internal/scheduler"
	"pa/internal/tools"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestScheduler_loadDifferentTaskFiles verifies AC-024: loading a different task file (e.g. after adding a task and restart) yields the new tasks.
func TestScheduler_loadDifferentTaskFiles(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "tasks_a.json")
	pathB := filepath.Join(dir, "tasks_b.json")
	if err := os.WriteFile(pathA, []byte(`[{"name":"a1","schedule":"0 9 * * *","action":"notify","params":{}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte(`[
		{"name":"b1","schedule":"0 9 * * *","action":"notify","params":{}},
		{"name":"b2","schedule":"@every 1h","action":"run_on_node","params":{"node_id":"nas","command":"uptime"}}
	]`), 0o600); err != nil {
		t.Fatal(err)
	}
	tasksA, err := scheduler.LoadTasks(pathA)
	if err != nil {
		t.Fatalf("LoadTasks(A) = %v", err)
	}
	tasksB, err := scheduler.LoadTasks(pathB)
	if err != nil {
		t.Fatalf("LoadTasks(B) = %v", err)
	}
	if len(tasksA) != 1 {
		t.Errorf("len(tasksA) = %d, want 1", len(tasksA))
	}
	if len(tasksB) != 2 {
		t.Errorf("len(tasksB) = %d, want 2", len(tasksB))
	}
	reg := tools.NewRegistry()
	cfg := scheduler.Config{Registry: reg, Logger: nil}
	_, err = scheduler.New(tasksB, cfg)
	if err != nil {
		t.Fatalf("New(tasksB) = %v", err)
	}
}

// countingTool is a tool that counts Run() calls for integration tests (AC-020).
type countingTool struct {
	name  string
	count *atomic.Int32
}

func (c *countingTool) Name() string                    { return c.name }
func (c *countingTool) Description() string             { return "counts runs" }
func (c *countingTool) ParamsSchema() []tools.ParamSpec { return nil }
func (c *countingTool) Run(ctx context.Context, params map[string]any) (string, error) {
	c.count.Add(1)
	return "ok", nil
}

// TestScheduler_firesAndRunsTool verifies AC-020: when the scheduled time is reached, the scheduler executes the task (invokes the tool).
func TestScheduler_firesAndRunsTool(t *testing.T) {
	var count atomic.Int32
	ct := &countingTool{name: "count_tool", count: &count}
	reg := tools.NewRegistry()
	reg.Register(ct)
	cfg := scheduler.Config{Registry: reg, Logger: nil}
	tasks := []scheduler.Task{
		{Name: "count-every-500ms", Schedule: "@every 500ms", Action: "count_tool", Params: map[string]any{}},
	}
	s, err := scheduler.New(tasks, cfg)
	if err != nil {
		t.Fatalf("New = %v", err)
	}
	s.Start()
	defer func() { <-s.Stop().Done() }()
	time.Sleep(1200 * time.Millisecond)
	if n := count.Load(); n < 1 {
		t.Errorf("tool Run was not called (count=%d), want at least 1", n)
	}
}

// TestScheduler_disallowedCommandNotExecuted verifies AC-021: when a scheduled task would run a command
// not on the node's allowlist, the system does not execute the violating action (runner returns error, executor never called).
func TestScheduler_disallowedCommandNotExecuted(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("uptime\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"n1": {
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
	runner := noderunner.New(cfg, al, slog.Default())
	var execCalls atomic.Int32
	runner.SetExecutor(&mockSchedulerExecutor{
		exec: func(context.Context, string, string) ([]byte, []byte, error) {
			execCalls.Add(1)
			return []byte("ok"), nil, nil
		},
	})
	reg := tools.NewRegistry()
	reg.Register(tools.NewRunOnNode(runner))
	tasks := []scheduler.Task{
		{Name: "disallowed-cmd", Schedule: "@every 500ms", Action: "run_on_node", Params: map[string]any{"node_id": "n1", "command": "rm -rf /"}},
	}
	s, err := scheduler.New(tasks, scheduler.Config{Registry: reg, Logger: slog.Default()})
	if err != nil {
		t.Fatalf("New = %v", err)
	}
	s.Start()
	defer func() { <-s.Stop().Done() }()
	time.Sleep(1200 * time.Millisecond)
	if n := execCalls.Load(); n != 0 {
		t.Errorf("AC-021: disallowed command was executed (exec calls=%d), want 0", n)
	}
}

type mockSchedulerExecutor struct {
	exec func(ctx context.Context, nodeID, command string) (stdout, stderr []byte, err error)
}

func (m *mockSchedulerExecutor) Exec(ctx context.Context, nodeID, command string) ([]byte, []byte, error) {
	if m.exec != nil {
		return m.exec(ctx, nodeID, command)
	}
	return nil, nil, nil
}
