package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"pa/internal/llm"
	"testing"
)

type testTimeoutNetErr struct{}

func (testTimeoutNetErr) Error() string   { return "i/o timeout" }
func (testTimeoutNetErr) Timeout() bool   { return true }
func (testTimeoutNetErr) Temporary() bool { return true }

type testNetworkNetErr struct{}

func (testNetworkNetErr) Error() string   { return "connection refused" }
func (testNetworkNetErr) Timeout() bool   { return false }
func (testNetworkNetErr) Temporary() bool { return true }

// Covers AC-01.036: nil classification fallback stays generic and does not crash.
func TestUserFacingErrorMessage_nilGeneric(t *testing.T) {
	got := UserFacingErrorMessage(nil)
	if got != userErrGenericMsg {
		t.Fatalf("UserFacingErrorMessage(nil) = %q, want %q", got, userErrGenericMsg)
	}
}

// Covers AC-01.035: wrapped tool-path error keeps underlying cause and explicit kind.
func TestWrapUserError_preservesUnderlyingAndKind(t *testing.T) {
	base := errors.New("boom")
	wrapped := WrapUserError(UserErrorKindToolResponse, base)
	if !errors.Is(wrapped, base) {
		t.Fatalf("wrapped error must preserve base with errors.Is")
	}
	got := UserFacingErrorMessage(wrapped)
	if got != userErrToolMsg {
		t.Fatalf("UserFacingErrorMessage(wrapped) = %q, want %q", got, userErrToolMsg)
	}
}

// Covers AC-01.036: context deadline exceeded maps to timeout user-facing class.
func TestUserFacingErrorMessage_deadlineExceededTimeout(t *testing.T) {
	got := UserFacingErrorMessage(context.DeadlineExceeded)
	if got != userErrTimeoutMsg {
		t.Fatalf("UserFacingErrorMessage(deadline) = %q, want %q", got, userErrTimeoutMsg)
	}
}

// Covers AC-01.036: wrapped deadline errors still classify as timeout.
func TestUserFacingErrorMessage_wrappedDeadlineExceededTimeout(t *testing.T) {
	err := fmt.Errorf("llm complete: %w", context.DeadlineExceeded)
	got := UserFacingErrorMessage(err)
	if got != userErrTimeoutMsg {
		t.Fatalf("UserFacingErrorMessage(wrapped deadline) = %q, want %q", got, userErrTimeoutMsg)
	}
}

// Covers AC-01.036: net timeout errors classify as timeout.
func TestUserFacingErrorMessage_timeoutNetErrorTimeout(t *testing.T) {
	err := &net.OpError{Err: testTimeoutNetErr{}}
	got := UserFacingErrorMessage(err)
	if got != userErrTimeoutMsg {
		t.Fatalf("UserFacingErrorMessage(timeout net) = %q, want %q", got, userErrTimeoutMsg)
	}
}

// Covers AC-01.036: non-timeout net errors classify as provider unavailable.
func TestUserFacingErrorMessage_nonTimeoutNetErrorProvider(t *testing.T) {
	err := &net.OpError{Err: testNetworkNetErr{}}
	got := UserFacingErrorMessage(err)
	if got != userErrProviderMsg {
		t.Fatalf("UserFacingErrorMessage(network net) = %q, want %q", got, userErrProviderMsg)
	}
}

// Covers AC-01.036: upstream API 5xx classifies as provider unavailable.
func TestUserFacingErrorMessage_apiError5xxProvider(t *testing.T) {
	err := &llm.APIError{StatusCode: 503, Err: errors.New("upstream unavailable")}
	got := UserFacingErrorMessage(err)
	if got != userErrProviderMsg {
		t.Fatalf("UserFacingErrorMessage(api 5xx) = %q, want %q", got, userErrProviderMsg)
	}
}

// Covers AC-01.036: upstream API 429 classifies as provider unavailable.
func TestUserFacingErrorMessage_apiError429Provider(t *testing.T) {
	err := &llm.APIError{StatusCode: 429, Err: errors.New("rate limited")}
	got := UserFacingErrorMessage(err)
	if got != userErrProviderMsg {
		t.Fatalf("UserFacingErrorMessage(api 429) = %q, want %q", got, userErrProviderMsg)
	}
}

// Covers AC-01.036: unknown plain errors fall back to generic class safely.
func TestUserFacingErrorMessage_plainErrorFallsBackToGeneric(t *testing.T) {
	got := UserFacingErrorMessage(errors.New("dial tcp"))
	if got != userErrGenericMsg {
		t.Fatalf("UserFacingErrorMessage(plain string error) = %q, want %q", got, userErrGenericMsg)
	}
}

// Covers AC-01.033: configuration-labeled errors map to configuration user message.
func TestUserFacingErrorMessage_configurationKind(t *testing.T) {
	err := WrapUserError(UserErrorKindConfiguration, errors.New("missing tool id in config"))
	got := UserFacingErrorMessage(err)
	if got != userErrConfigurationMsg {
		t.Fatalf("UserFacingErrorMessage(configuration) = %q, want %q", got, userErrConfigurationMsg)
	}
}

// Covers AC-01.035: outer explicit classification wins on nested wraps.
func TestWrapUserError_doubleWrapOuterKindWins(t *testing.T) {
	base := errors.New("base")
	inner := WrapUserError(UserErrorKindProviderUnavailable, base)
	outer := WrapUserError(UserErrorKindToolResponse, inner)

	got := UserFacingErrorMessage(outer)
	if got != userErrToolMsg {
		t.Fatalf("UserFacingErrorMessage(double wrap) = %q, want %q", got, userErrToolMsg)
	}
}

// Covers AC-01.033: classified errors keep non-empty diagnostic text even with nil inner error.
func TestClassifiedError_errorWithNilInnerIncludesKind(t *testing.T) {
	err := (&ClassifiedError{Kind: UserErrorKindConfiguration}).Error()
	want := "classified error: configuration"
	if err != want {
		t.Fatalf("ClassifiedError.Error() = %q, want %q", err, want)
	}
}
