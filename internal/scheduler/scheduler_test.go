package scheduler

import (
	"context"
	"log/slog"
	"pa/internal/tools"
	"testing"
)

func TestNew_validTasks(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&mockTool{name: "run_on_node"})
	cfg := Config{Registry: reg, Logger: slog.Default()}
	taskList := []Task{
		{Schedule: "0 9 * * *", Action: "notify", Params: map[string]any{}},
		{Schedule: "@every 1h", Action: "run_on_node", Params: map[string]any{"node_id": "n", "command": "uptime"}},
	}
	s, err := New(taskList, cfg)
	if err != nil {
		t.Fatalf("New = %v", err)
	}
	if s == nil {
		t.Fatal("New returned nil scheduler")
	}
	s.Start()
	ctx := s.Stop()
	<-ctx.Done()
}

func TestNew_invalidSchedule(t *testing.T) {
	reg := tools.NewRegistry()
	cfg := Config{Registry: reg, Logger: slog.Default()}
	taskList := []Task{
		{Schedule: "invalid cron!!!", Action: "notify", Params: map[string]any{}},
	}
	_, err := New(taskList, cfg)
	if err == nil {
		t.Error("New(invalid schedule) want error")
	}
}

func TestScheduler_executeTask_unknownAction_skipped(t *testing.T) {
	reg := tools.NewRegistry()
	mt := &mockTool{name: "only_tool"}
	reg.Register(mt)
	cfg := Config{Registry: reg, Logger: slog.Default()}
	s, err := New([]Task{{Schedule: "@every 1s", Action: "unknown_tool", Params: map[string]any{}}}, cfg)
	if err != nil {
		t.Fatalf("New = %v", err)
	}
	s.executeTask(context.Background(), Task{Action: "unknown_tool", Params: map[string]any{}})
	if mt.runCount != 0 {
		t.Errorf("executeTask(unknown) should not call tool, runCount = %d", mt.runCount)
	}
}

func TestScheduler_executeTask_invalidParams_skipped(t *testing.T) {
	reg := tools.NewRegistry()
	mt := &mockTool{name: "run_on_node", schema: []tools.ParamSpec{
		{Name: "node_id", Required: true, Type: "string"},
		{Name: "command", Required: true, Type: "string"},
	}}
	reg.Register(mt)
	cfg := Config{Registry: reg, Logger: slog.Default()}
	s, err := New(nil, cfg)
	if err != nil {
		t.Fatalf("New = %v", err)
	}
	s.executeTask(context.Background(), Task{Action: "run_on_node", Params: map[string]any{}}) // missing params
	if mt.runCount != 0 {
		t.Errorf("executeTask(invalid params) should not call tool, runCount = %d", mt.runCount)
	}
}

type mockTool struct {
	name     string
	schema   []tools.ParamSpec
	runCount int
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "mock" }
func (m *mockTool) ParamsSchema() []tools.ParamSpec {
	if m.schema != nil {
		return m.schema
	}
	return nil
}

func (m *mockTool) Run(ctx context.Context, params map[string]any) (string, error) {
	m.runCount++
	return "ok", nil
}
