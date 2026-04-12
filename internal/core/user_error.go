package core

import (
	"context"
	"errors"
	"net"
	"pa/internal/llm"
)

// UserErrorKind is a stable user-facing classification for request failures.
type UserErrorKind string

const (
	UserErrorKindGeneric             UserErrorKind = "generic"
	UserErrorKindTimeout             UserErrorKind = "timeout"
	UserErrorKindProviderUnavailable UserErrorKind = "provider_unavailable"
	UserErrorKindToolResponse        UserErrorKind = "tool_response"
	UserErrorKindConfiguration       UserErrorKind = "configuration"
)

const (
	userErrGenericMsg       = "Sorry, an error occurred. Please try again."
	userErrTimeoutMsg       = "Temporary timeout while processing the request. Please try again."
	userErrProviderMsg      = "Temporary connection issue with AI provider. Please try again."
	userErrToolMsg          = "Request processing failed due to invalid tool response. Please rephrase and try again."
	userErrConfigurationMsg = "Service configuration issue detected. Please contact support."
)

// ClassifiedError carries explicit user-facing error classification while preserving the root error.
type ClassifiedError struct {
	Kind UserErrorKind
	Err  error
}

func (e *ClassifiedError) Error() string {
	if e == nil {
		return "classified error"
	}
	if e.Err == nil {
		if e.Kind == "" {
			return "classified error"
		}
		return "classified error: " + string(e.Kind)
	}
	return e.Err.Error()
}

func (e *ClassifiedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// WrapUserError annotates err with explicit user-facing classification.
func WrapUserError(kind UserErrorKind, err error) error {
	if err == nil {
		return nil
	}
	if kind == "" {
		kind = UserErrorKindGeneric
	}
	return &ClassifiedError{Kind: kind, Err: err}
}

// UserFacingErrorMessage maps an internal error to a safe message for chat adapters.
func UserFacingErrorMessage(err error) string {
	switch classifyUserErrorKind(err) {
	case UserErrorKindTimeout:
		return userErrTimeoutMsg
	case UserErrorKindProviderUnavailable:
		return userErrProviderMsg
	case UserErrorKindToolResponse:
		return userErrToolMsg
	case UserErrorKindConfiguration:
		return userErrConfigurationMsg
	default:
		return userErrGenericMsg
	}
}

func classifyUserErrorKind(err error) UserErrorKind {
	if err == nil {
		return UserErrorKindGeneric
	}
	var ce *ClassifiedError
	if errors.As(err, &ce) {
		return ce.Kind
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return UserErrorKindTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return UserErrorKindTimeout
		}
		return UserErrorKindProviderUnavailable
	}
	var apiErr *llm.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 429 || apiErr.StatusCode >= 500 {
			return UserErrorKindProviderUnavailable
		}
	}
	return UserErrorKindGeneric
}
