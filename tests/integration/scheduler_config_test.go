//go:build integration

package integration_test

import (
	"context"
	"os"
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
	if err := os.WriteFile(pathA, []byte(`[{"schedule":"0 9 * * *","action":"notify","params":{}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte(`[
		{"schedule":"0 9 * * *","action":"notify","params":{}},
		{"schedule":"@every 1h","action":"run_on_node","params":{"node_id":"nas","command":"uptime"}}
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
		{Schedule: "@every 500ms", Action: "count_tool", Params: map[string]any{}},
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
