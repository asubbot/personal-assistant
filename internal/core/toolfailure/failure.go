// Package toolfailure types tool-path outcomes for LLM escalation policy (EP-006, REQ-06.015).
package toolfailure

import "errors"

// Failure attaches an explicit escalation policy to a tool-path error. Classification for
// escalation SHALL use this type (errors.As); it SHALL NOT rely solely on matching substrings in Error().
type Failure struct {
	Escalate bool
	err      error
}

func (e *Failure) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap returns the underlying error for errors.Is / errors.As.
func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// NoEscalate wraps err as a failure that does not qualify for provider escalation (policy, unknown tool, etc.).
func NoEscalate(err error) error {
	if err == nil {
		return nil
	}
	return &Failure{Escalate: false, err: err}
}

// MayEscalate wraps err as a failure that may qualify for escalation (e.g. remote exec, SSH, tool parse).
func MayEscalate(err error) error {
	if err == nil {
		return nil
	}
	return &Failure{Escalate: true, err: err}
}

// QualifiesForEscalation reports whether err is typed as Failure with Escalate true (REQ-06.015).
// Untyped errors do not qualify (fail closed).
func QualifiesForEscalation(err error) bool {
	if err == nil {
		return false
	}
	var f *Failure
	if errors.As(err, &f) {
		return f.Escalate
	}
	return false
}
