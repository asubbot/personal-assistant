package tools

import (
	"context"
	"fmt"
	"strings"
)

// RunOnNodeRunner runs an allowlisted command on a node (e.g. via SSH). Implemented by core.NodeRunner.
type RunOnNodeRunner interface {
	RunOnNode(ctx context.Context, nodeID, command string) (stdout string, err error)
}

// RunOnNodeTool runs an allowlisted command on a configured node (REQ-01.004, REQ-01.005, REQ-01.013).
type RunOnNodeTool struct {
	runner RunOnNodeRunner
}

// NewRunOnNode returns a tool that runs commands on nodes via the given runner.
func NewRunOnNode(runner RunOnNodeRunner) *RunOnNodeTool {
	return &RunOnNodeTool{runner: runner}
}

// Name returns the tool name.
func (t *RunOnNodeTool) Name() string { return "run_on_node" }

// Description returns a short description.
func (t *RunOnNodeTool) Description() string {
	return "Run an allowlisted command on a configured node (e.g. SSH)."
}

// ParamsSchema returns the required params: node_id (string), command (string).
func (t *RunOnNodeTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{
		{Name: "node_id", Required: true, Type: "string"},
		{Name: "command", Required: true, Type: "string"},
	}
}

// Run executes the command on the node. Runner may be nil (no nodes configured); then returns error.
func (t *RunOnNodeTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if t.runner == nil {
		return "", fmt.Errorf("tools: run_on_node: no node runner configured")
	}
	if err := ValidateParams(t.ParamsSchema(), params); err != nil {
		return "", err
	}
	nodeID, _ := params["node_id"].(string)
	command, _ := params["command"].(string)
	nodeID = strings.TrimSpace(nodeID)
	command = strings.TrimSpace(command)
	if nodeID == "" || command == "" {
		return "", fmt.Errorf("tools: run_on_node: node_id and command are required")
	}
	return t.runner.RunOnNode(ctx, nodeID, command)
}
