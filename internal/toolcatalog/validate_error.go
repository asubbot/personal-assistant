// ValidateError and ValidateKind classify failures from ValidateToolCall for policy and tests.
package toolcatalog

// ValidateKind identifies a catalog validation failure for policy mapping and tests.
type ValidateKind int

const (
	ValidateKindUnknownTool ValidateKind = iota + 1
	ValidateKindInvalidArgsJSON
	ValidateKindMissingRequiredArg
	ValidateKindAllowedValuesMismatch
	ValidateKindPatternInvalid
	ValidateKindPatternMismatch
	ValidateKindMinMaxNonNumber
	ValidateKindMinViolated
	ValidateKindMaxViolated
	ValidateKindArgTypeString
	ValidateKindArgTypeInteger
	ValidateKindArgTypeNumber
	ValidateKindArgTypeBoolean
)

// ValidateError wraps a validation failure with a stable kind inspectable via errors.As.
type ValidateError struct {
	Kind ValidateKind
	err  error
}

// Error implements error.
func (e *ValidateError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap returns the underlying error for errors.Is / errors.Unwrap.
func (e *ValidateError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func validateErr(kind ValidateKind, err error) error {
	if err == nil {
		return nil
	}
	return &ValidateError{Kind: kind, err: err}
}
