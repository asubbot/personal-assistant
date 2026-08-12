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

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}
