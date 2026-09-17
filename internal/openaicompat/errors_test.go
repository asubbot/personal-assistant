package openaicompat

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// Covers AC-01.036, AC-01.037: shared provider error parsing returns a safe message without consuming ownership of the response body.
func TestDecodeErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "valid error message",
			body: `{"error":{"message":"rate limit exceeded"}}`,
			want: "rate limit exceeded",
		},
		{
			name: "empty error message",
			body: `{"error":{"message":""}}`,
			want: "429 Too Many Requests",
		},
		{
			name: "invalid JSON",
			body: `not json`,
			want: "429 Too Many Requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackingReadCloser{Reader: strings.NewReader(tt.body)}
			resp := &http.Response{
				Status:     "429 Too Many Requests",
				StatusCode: http.StatusTooManyRequests,
				Body:       body,
			}

			if got := DecodeErrorMessage(resp); got != tt.want {
				t.Fatalf("DecodeErrorMessage() = %q, want %q", got, tt.want)
			}
			if body.closed {
				t.Fatal("DecodeErrorMessage() closed response body")
			}
		})
	}
}

// Covers AC-01.036, AC-01.037: structured error.code/error.type decode; null code is empty.
func TestDecodeError_codeTypeAndNullCode(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader(`{"error":{"message":"You have no credits remaining.","type":"insufficient_quota","code":"credit_balance_exhausted"}}`)}
	resp := &http.Response{Status: "429 Too Many Requests", StatusCode: http.StatusTooManyRequests, Body: body}
	got := DecodeError(resp)
	if got.Message != "You have no credits remaining." || got.Type != "insufficient_quota" || got.Code != "credit_balance_exhausted" {
		t.Fatalf("DecodeError() = %+v", got)
	}
	if body.closed {
		t.Fatal("DecodeError() closed response body")
	}

	nullBody := &trackingReadCloser{Reader: strings.NewReader(`{"error":{"message":"x","type":"insufficient_quota","code":null}}`)}
	got = DecodeError(&http.Response{StatusCode: http.StatusTooManyRequests, Body: nullBody})
	if got.Code != "" || got.Type != "insufficient_quota" || got.Message != "x" {
		t.Fatalf("DecodeError(null code) = %+v", got)
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}
