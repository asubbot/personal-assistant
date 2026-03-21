package cmdsafe

import (
	"errors"
)

// CommandRejectKind identifies which ValidateRemoteCommand stage failed.
type CommandRejectKind int

const (
	// CommandRejectRunes is UTF-8 / length / allowed-rune policy (before shell metacharacters).
	CommandRejectRunes CommandRejectKind = iota
	// CommandRejectShellMeta is REQ-04.031 shell metacharacter rejection.
	CommandRejectShellMeta
)

// CommandValidationError wraps a rejection from ValidateRemoteCommand so callers can map outcomes (e.g. noderunner escalation).
type CommandValidationError struct {
	Kind CommandRejectKind
	Err  error
}

func (e *CommandValidationError) Error() string { return e.Err.Error() }
func (e *CommandValidationError) Unwrap() error { return e.Err }

// ValidateRemoteCommand applies RejectDisallowedRunes then RejectShellMetacharacters.
// On failure, returns *CommandValidationError (use RejectKind with errors.As on the returned error).
func ValidateRemoteCommand(cmd string) error {
	if err := RejectDisallowedRunes(cmd); err != nil {
		return &CommandValidationError{Kind: CommandRejectRunes, Err: err}
	}
	if err := RejectShellMetacharacters(cmd); err != nil {
		return &CommandValidationError{Kind: CommandRejectShellMeta, Err: err}
	}
	return nil
}

// RejectKind returns the validation stage for errors from ValidateRemoteCommand.
// If err is not a CommandValidationError (including when wrapped with fmt.Errorf("%w")), ok is false.
func RejectKind(err error) (kind CommandRejectKind, ok bool) {
	var v *CommandValidationError
	if !errors.As(err, &v) {
		return 0, false
	}
	return v.Kind, true
}
