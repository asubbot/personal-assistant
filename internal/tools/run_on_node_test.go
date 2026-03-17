package tools

import (
	"context"
	"errors"
	"testing"
)

// Covers AC-01.022 (US-12): tool invoked with validated input returns result from runner.
func TestRunOnNodeTool_Run_validParams(t *testing.T) {
	var gotNodeID, gotCmd string
	runner := &mockRunOnNodeRunner{
		run: func(ctx context.Context, nodeID, command string) (string, error) {
			gotNodeID = nodeID
			gotCmd = command
			return "stdout", nil
		},
	}
	tool := NewRunOnNode(runner)
	ctx := context.Background()
	out, err := tool.Run(ctx, map[string]any{"node_id": "nas", "command": "uptime"})
	if err != nil {
		t.Fatalf("Run = %v", err)
	}
	if out != "stdout" {
		t.Errorf("Run = %q, want stdout", out)
	}
	if gotNodeID != "nas" || gotCmd != "uptime" {
		t.Errorf("runner called with node_id=%q command=%q, want nas, uptime", gotNodeID, gotCmd)
	}
}

// Covers AC-01.023 (US-12): invalid or out-of-schema input is rejected without executing the tool.
func TestRunOnNodeTool_Run_invalidParams(t *testing.T) {
	runner := &mockRunOnNodeRunner{run: func(context.Context, string, string) (string, error) { return "", nil }}
	tool := NewRunOnNode(runner)
	ctx := context.Background()
	_, err := tool.Run(ctx, map[string]any{})
	if err == nil {
		t.Error("Run(missing params) want error")
	}
	_, err = tool.Run(ctx, map[string]any{"node_id": "nas"})
	if err == nil {
		t.Error("Run(missing command) want error")
	}
	_, err = tool.Run(ctx, map[string]any{"node_id": 123, "command": "uptime"})
	if err == nil {
		t.Error("Run(wrong type node_id) want error")
	}
}

// Covers AC-01.035 (US-12): tool invoked with nil runner returns error to caller.
func TestRunOnNodeTool_Run_nilRunner(t *testing.T) {
	tool := NewRunOnNode(nil)
	_, err := tool.Run(context.Background(), map[string]any{"node_id": "nas", "command": "uptime"})
	if err == nil {
		t.Error("Run(nil runner) want error")
	}
}

// Covers AC-01.008 (US-04), AC-01.035 (US-12): RunOnNode tool propagates runner error (e.g. allowlist denial) to caller.
func TestRunOnNodeTool_Run_runnerError(t *testing.T) {
	wantErr := errors.New("disallowed")
	runner := &mockRunOnNodeRunner{
		run: func(context.Context, string, string) (string, error) { return "", wantErr },
	}
	tool := NewRunOnNode(runner)
	_, err := tool.Run(context.Background(), map[string]any{"node_id": "nas", "command": "rm -rf /"})
	if !errors.Is(err, wantErr) {
		t.Errorf("Run = %v, want %v", err, wantErr)
	}
}

type mockRunOnNodeRunner struct {
	run func(ctx context.Context, nodeID, command string) (string, error)
}

func (m *mockRunOnNodeRunner) RunOnNode(ctx context.Context, nodeID, command string) (string, error) {
	if m.run != nil {
		return m.run(ctx, nodeID, command)
	}
	return "", nil
}
