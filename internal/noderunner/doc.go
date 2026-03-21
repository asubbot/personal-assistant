// Package noderunner runs allowlisted commands on nodes via SSH.
// It applies cmdsafe rune and shell checks, then the per-node allowlist (REQ-01.005), then connects as the dedicated user (REQ-01.013) and executes the command.
package noderunner
