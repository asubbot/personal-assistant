package llmrouter

import (
	"context"
	"errors"
	"net"
	"pa/internal/llm"
)

// ClassifyCompleteError returns a stable class for provider Complete errors.
func ClassifyCompleteError(err error) FailureClass {
	if errors.Is(err, context.Canceled) {
		return FailureClassCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return FailureClassTransportTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return FailureClassTransportNetwork
	}
	var apiErr *llm.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
		return FailureClassTransport5xx
	}
	return FailureClassOther
}

// IsTransportRetryable returns true when error class warrants trying next provider.
func IsTransportRetryable(class FailureClass) bool {
	switch class {
	case FailureClassTransportTimeout, FailureClassTransportNetwork, FailureClassTransport5xx:
		return true
	default:
		return false
	}
}
