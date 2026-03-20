package noderunner

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/allowlist"
	"pa/internal/cmdsafe"
	"pa/internal/config"
	"pa/internal/escalationpolicy"
	"pa/internal/ssh"
	"strings"
	"unicode/utf8"
)

// maxRemoteStreamBytes caps stdout/stderr fragments embedded in errors and logs (per stream).
const maxRemoteStreamBytes = 4096

const remoteStreamTruncatedSuffix = "...[truncated]"

// truncateRemoteStream returns a display-safe fragment of remote command output for errors and logs.
// Empty input yields empty string. Long output is cut at maxRemoteStreamBytes on a UTF-8 boundary.
func truncateRemoteStream(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) <= maxRemoteStreamBytes {
		return string(b)
	}
	n := maxRemoteStreamBytes
	for n > 0 && !utf8.RuneStart(b[n]) {
		n--
	}
	if n == 0 {
		return remoteStreamTruncatedSuffix
	}
	return string(b[:n]) + remoteStreamTruncatedSuffix
}

// remoteStreamsSuffix appends labeled stdout/stderr fragments to an exec error message (non-empty parts only).
func remoteStreamsSuffix(stdoutTrunc, stderrTrunc string) string {
	var sb strings.Builder
	if stdoutTrunc != "" {
		sb.WriteString("\nstdout: ")
		sb.WriteString(stdoutTrunc)
	}
	if stderrTrunc != "" {
		sb.WriteString("\nstderr: ")
		sb.WriteString(stderrTrunc)
	}
	return sb.String()
}

func appendRemoteStreamAttrs(attrs []any, stdoutTrunc, stderrTrunc string) []any {
	if stdoutTrunc != "" {
		attrs = append(attrs, "remote_stdout", stdoutTrunc)
	}
	if stderrTrunc != "" {
		attrs = append(attrs, "remote_stderr", stderrTrunc)
	}
	return attrs
}

// Executor runs a command on a node (used for testing; production uses SSH).
type Executor interface {
	Exec(ctx context.Context, nodeID, command string) (stdout, stderr []byte, err error)
}

// Runner runs commands on nodes only if they are on the allowlist; uses SSH with config credentials only (REQ-01.004, REQ-01.005, REQ-01.013).
type Runner struct {
	cfg       *config.Config
	allowlist *allowlist.Checker
	logger    *slog.Logger
	executor  Executor            // optional; when set (e.g. in tests) used instead of real SSH
	logRedact func(string) string // optional; remote stdout/stderr fragments in app logs only (not returned errors)
}

// New returns a Runner that uses the given config and allowlist. Paths in config are relative to project root (CWD).
func New(cfg *config.Config, al *allowlist.Checker, logger *slog.Logger) *Runner {
	return &Runner{cfg: cfg, allowlist: al, logger: logger}
}

// SetExecutor injects an executor for tests; when set, RunOnNode uses it instead of real SSH.
func (r *Runner) SetExecutor(e Executor) {
	r.executor = e
}

// SetLogRedactor sets optional redaction for remote stdout/stderr fragments in Error/Debug logs. Errors returned from RunOnNode are not redacted so tool/LLM diagnostics stay intact.
func (r *Runner) SetLogRedactor(fn func(string) string) {
	r.logRedact = fn
}

func (r *Runner) redactLogString(s string) string {
	if r == nil || r.logRedact == nil {
		return s
	}
	return r.logRedact(s)
}

// RunOnNode runs the command on the node only if it is allowlisted (AC-01.007, AC-01.008). Uses SSH with the node's dedicated user only (AC-01.006, AC-01.009, AC-01.010).
// On connection or exec failure logs and returns error; no fallback to other users (REQ-01.013).
func (r *Runner) RunOnNode(ctx context.Context, nodeID, command string) (stdout string, err error) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", escalationpolicy.WrapNodeOutcome(escalationpolicy.NodeOutcomeEmptyCommand, fmt.Errorf("noderunner: command is empty"))
	}
	if err := cmdsafe.RejectShellMetacharacters(cmd); err != nil {
		return "", escalationpolicy.WrapNodeOutcome(escalationpolicy.NodeOutcomeShellMetaRejected, fmt.Errorf("noderunner: %w", err))
	}
	if r.allowlist == nil {
		return "", escalationpolicy.WrapNodeOutcome(escalationpolicy.NodeOutcomeAllowlistNotConfigured, fmt.Errorf("noderunner: allowlist not configured"))
	}
	if !r.allowlist.Allow(nodeID, cmd) {
		r.logger.Warn("command not on allowlist", "node_id", nodeID, "command", cmd)
		return "", escalationpolicy.WrapNodeOutcome(escalationpolicy.NodeOutcomeAllowlistDenied, fmt.Errorf("noderunner: command not allowed for node %q", nodeID))
	}
	var out, stderr []byte
	if r.executor != nil {
		out, stderr, err = r.executor.Exec(ctx, nodeID, cmd)
	} else {
		client, connErr := ssh.NewClient(ctx, r.cfg, nodeID)
		if connErr != nil {
			r.logger.Error("ssh connect", "node_id", nodeID, "error", connErr)
			return "", escalationpolicy.WrapNodeOutcome(escalationpolicy.NodeOutcomeRemoteExecFailure, fmt.Errorf("noderunner: ssh: %w", connErr))
		}
		defer func() {
			if closeErr := client.Close(); closeErr != nil && err == nil {
				r.logger.Error("ssh close", "node_id", nodeID, "error", closeErr)
			}
		}()
		out, stderr, err = client.Exec(ctx, cmd)
	}
	return r.finishRemoteExec(nodeID, cmd, out, stderr, err)
}

// finishRemoteExec logs and maps remote stdout/stderr into errors or DEBUG output after Exec returns.
func (r *Runner) finishRemoteExec(nodeID, cmd string, out, stderr []byte, execErr error) (string, error) {
	if execErr != nil {
		outTrunc := truncateRemoteStream(out)
		errTrunc := truncateRemoteStream(stderr)
		logAttrs := appendRemoteStreamAttrs([]any{"node_id", nodeID, "command", cmd, "error", execErr}, r.redactLogString(outTrunc), r.redactLogString(errTrunc))
		r.logger.Error("ssh exec", logAttrs...)
		detail := remoteStreamsSuffix(outTrunc, errTrunc)
		return "", escalationpolicy.WrapNodeOutcome(escalationpolicy.NodeOutcomeRemoteExecFailure, fmt.Errorf("noderunner: exec: %w%s", execErr, detail))
	}
	if len(stderr) > 0 || len(out) > 0 {
		outTrunc := truncateRemoteStream(out)
		errTrunc := truncateRemoteStream(stderr)
		if outTrunc != "" || errTrunc != "" {
			dbgAttrs := appendRemoteStreamAttrs([]any{"node_id", nodeID, "command", cmd}, r.redactLogString(outTrunc), r.redactLogString(errTrunc))
			r.logger.Debug("ssh exec output", dbgAttrs...)
		}
	}
	return string(out), nil
}
