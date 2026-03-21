package escalationpolicy

import (
	"pa/internal/core/toolfailure"
)

// NodeOutcome classifies a noderunner failure for LLM escalation policy (EP-006, REQ-06.004, REQ-06.005, REQ-06.017).
// It is used by noderunner only; this package does not import noderunner.
type NodeOutcome int

const (
	// NodeOutcomeEmptyCommand is a trimmed-empty command before allowlist (policy: no escalation).
	NodeOutcomeEmptyCommand NodeOutcome = iota
	// NodeOutcomeShellMetaRejected is cmdsafe shell-metacharacter rejection (policy: no escalation).
	NodeOutcomeShellMetaRejected
	// NodeOutcomeDisallowedRunes is cmdsafe rune / UTF-8 policy rejection (policy: no escalation).
	NodeOutcomeDisallowedRunes
	// NodeOutcomeAllowlistNotConfigured is nil allowlist checker (policy: no escalation).
	NodeOutcomeAllowlistNotConfigured
	// NodeOutcomeAllowlistDenied is command not on allowlist (policy: no escalation).
	NodeOutcomeAllowlistDenied
	// NodeOutcomeRemoteExecFailure is SSH connect failure, remote exec error, or injected executor error (may escalate).
	NodeOutcomeRemoteExecFailure
)

// WrapNodeOutcome maps a classified node-path outcome to toolfailure.NoEscalate or MayEscalate.
// err must be non-nil. Unknown outcome values fail closed to NoEscalate.
func WrapNodeOutcome(outcome NodeOutcome, err error) error {
	if err == nil {
		return nil
	}
	switch outcome {
	case NodeOutcomeRemoteExecFailure:
		return toolfailure.MayEscalate(err)
	case NodeOutcomeEmptyCommand, NodeOutcomeShellMetaRejected, NodeOutcomeDisallowedRunes, NodeOutcomeAllowlistNotConfigured, NodeOutcomeAllowlistDenied:
		return toolfailure.NoEscalate(err)
	default:
		return toolfailure.NoEscalate(err)
	}
}
