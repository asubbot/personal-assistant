package noderunner

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/allowlist"
	"pa/internal/config"
	"pa/internal/ssh"
	"strings"
)

// Executor runs a command on a node (used for testing; production uses SSH).
type Executor interface {
	Exec(ctx context.Context, nodeID, command string) (stdout, stderr []byte, err error)
}

// Runner runs commands on nodes only if they are on the allowlist; uses SSH with config credentials only (REQ-004, REQ-005, REQ-013).
type Runner struct {
	cfg       *config.Config
	allowlist *allowlist.Checker
	logger    *slog.Logger
	executor  Executor // optional; when set (e.g. in tests) used instead of real SSH
}

// New returns a Runner that uses the given config and allowlist. Paths in config are relative to project root (CWD).
func New(cfg *config.Config, al *allowlist.Checker, logger *slog.Logger) *Runner {
	return &Runner{cfg: cfg, allowlist: al, logger: logger}
}

// SetExecutor injects an executor for tests; when set, RunOnNode uses it instead of real SSH.
func (r *Runner) SetExecutor(e Executor) {
	r.executor = e
}

// RunOnNode runs the command on the node only if it is allowlisted (AC-007, AC-008). Uses SSH with the node's dedicated user only (AC-006, AC-009, AC-010).
// On connection or exec failure logs and returns error; no fallback to other users (REQ-013).
func (r *Runner) RunOnNode(ctx context.Context, nodeID, command string) (stdout string, err error) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("noderunner: command is empty")
	}
	if r.allowlist == nil {
		return "", fmt.Errorf("noderunner: allowlist not configured")
	}
	if !r.allowlist.Allow(nodeID, cmd) {
		r.logger.Warn("command not on allowlist", "node_id", nodeID, "command", cmd)
		return "", fmt.Errorf("noderunner: command not allowed for node %q", nodeID)
	}
	var out, stderr []byte
	if r.executor != nil {
		out, stderr, err = r.executor.Exec(ctx, nodeID, cmd)
	} else {
		client, connErr := ssh.NewClient(ctx, r.cfg, nodeID)
		if connErr != nil {
			r.logger.Error("ssh connect", "node_id", nodeID, "error", connErr)
			return "", fmt.Errorf("noderunner: ssh: %w", connErr)
		}
		defer func() {
			if closeErr := client.Close(); closeErr != nil && err == nil {
				r.logger.Error("ssh close", "node_id", nodeID, "error", closeErr)
			}
		}()
		out, stderr, err = client.Exec(ctx, cmd)
	}
	if err != nil {
		r.logger.Error("ssh exec", "node_id", nodeID, "command", cmd, "error", err)
		return "", fmt.Errorf("noderunner: exec: %w", err)
	}
	if len(stderr) > 0 {
		r.logger.Debug("ssh stderr", "node_id", nodeID, "stderr", string(stderr))
	}
	return string(out), nil
}
