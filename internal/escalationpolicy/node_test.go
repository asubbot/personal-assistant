// Tests for WrapNodeOutcome (noderunner → toolfailure mapping, EP-006).
//
// Covers AC-06.012 (REQ-06.015): escalation qualification uses typed Failure / errors.As (QualifiesForEscalation), not Error() substring rules.
// Supporting AC-06.014 (REQ-06.017): escalation-allowance mapping lives in this package and is unit-tested without handler, Telegram, or real LLM.
package escalationpolicy

import (
	"errors"
	"pa/internal/core/toolfailure"
	"testing"
)

func TestWrapNodeOutcome_nilError(t *testing.T) {
	t.Parallel()
	if got := WrapNodeOutcome(NodeOutcomeRemoteExecFailure, nil); got != nil {
		t.Fatalf("WrapNodeOutcome(..., nil) = %v, want nil", got)
	}
}

func TestWrapNodeOutcome_table(t *testing.T) {
	t.Parallel()
	base := errors.New("underlying")

	tests := []struct {
		name         string
		outcome      NodeOutcome
		wantEscalate bool
	}{
		{"empty_command", NodeOutcomeEmptyCommand, false},
		{"shell_meta", NodeOutcomeShellMetaRejected, false},
		{"allowlist_nil", NodeOutcomeAllowlistNotConfigured, false},
		{"allowlist_denied", NodeOutcomeAllowlistDenied, false},
		{"remote_exec", NodeOutcomeRemoteExecFailure, true},
		{"unknown_outcome_fail_closed", NodeOutcome(999), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wrapped := WrapNodeOutcome(tt.outcome, base)
			if wrapped == nil {
				t.Fatal("expected non-nil error")
			}
			got := toolfailure.QualifiesForEscalation(wrapped)
			if got != tt.wantEscalate {
				t.Errorf("QualifiesForEscalation = %v, want %v", got, tt.wantEscalate)
			}
		})
	}
}
