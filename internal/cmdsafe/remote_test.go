package cmdsafe

import (
	"errors"
	"fmt"
	"testing"
)

// Covers AC-04.029: traceability for TestValidateRemoteCommand_ok.
func TestValidateRemoteCommand_ok(t *testing.T) {
	if err := ValidateRemoteCommand("uptime"); err != nil {
		t.Fatal(err)
	}
}

// Covers AC-04.029: traceability for TestCommandValidationError_Unwrap.
func TestCommandValidationError_Unwrap(t *testing.T) {
	inner := errors.New("underlying")
	e := &CommandValidationError{Kind: CommandRejectRunes, Err: inner}
	if !errors.Is(e, inner) {
		t.Fatal("errors.Is should follow Unwrap to inner error")
	}
}

// Covers AC-04.029: traceability for TestValidateRemoteCommand_rejectKind_runes.
func TestValidateRemoteCommand_rejectKind_runes(t *testing.T) {
	err := ValidateRemoteCommand("a~b")
	if err == nil {
		t.Fatal("want error")
	}
	kind, ok := RejectKind(err)
	if !ok || kind != CommandRejectRunes {
		t.Fatalf("RejectKind = %v, %v, want CommandRejectRunes, true", kind, ok)
	}
	var v *CommandValidationError
	if !errors.As(err, &v) || v.Kind != CommandRejectRunes {
		t.Fatalf("errors.As CommandValidationError: %v", err)
	}
}

// Semicolon is rejected by the rune policy before shell-meta; RejectKind still classifies the first failure.
// Covers AC-04.029: traceability for TestValidateRemoteCommand_semicolonRejectedAsRunes.
func TestValidateRemoteCommand_semicolonRejectedAsRunes(t *testing.T) {
	err := ValidateRemoteCommand("echo;rm")
	if err == nil {
		t.Fatal("want error")
	}
	kind, ok := RejectKind(err)
	if !ok || kind != CommandRejectRunes {
		t.Fatalf("RejectKind = %v, %v, want CommandRejectRunes (semicolon not in allowed set)", kind, ok)
	}
}

// Covers AC-04.029: traceability for TestRejectKind_findsWrappedCommandValidationError.
func TestRejectKind_findsWrappedCommandValidationError(t *testing.T) {
	inner := &CommandValidationError{Kind: CommandRejectShellMeta, Err: errors.New("forbidden shell sequence")}
	wrapped := fmt.Errorf("noderunner: %w", inner)
	kind, ok := RejectKind(wrapped)
	if !ok || kind != CommandRejectShellMeta {
		t.Fatalf("RejectKind = %v, %v, want CommandRejectShellMeta, true", kind, ok)
	}
}
