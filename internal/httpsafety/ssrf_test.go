package httpsafety

import (
	"context"
	"errors"
	"testing"
)

// Covers AC-11.008, AC-11.014 — non-https scheme rejected (structured error type).
func TestValidateFetchURL_HTTPRejected(t *testing.T) {
	ctx := context.Background()
	err := ValidateFetchURL(ctx, "http://example.com/", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var du *ErrDisallowedURL
	if !errors.As(err, &du) || du.Reason == "" {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.008 — https with literal public IP allowed (1.1.1.1).
func TestValidateFetchURL_PublicIPAllowed(t *testing.T) {
	ctx := context.Background()
	err := ValidateFetchURL(ctx, "https://1.1.1.1/", nil)
	if err != nil {
		t.Fatal(err)
	}
}

// Covers AC-11.009 — loopback literal rejected.
func TestValidateFetchURL_LoopbackRejected(t *testing.T) {
	ctx := context.Background()
	for _, u := range []string{"https://127.0.0.1/", "https://[::1]/"} {
		err := ValidateFetchURL(ctx, u, nil)
		if err == nil {
			t.Fatalf("want error for %s", u)
		}
	}
}

// Covers AC-11.009, AC-11.014 — private IP literal rejected.
func TestValidateFetchURL_PrivateIPRejected(t *testing.T) {
	ctx := context.Background()
	err := ValidateFetchURL(ctx, "https://10.0.0.1/", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Covers AC-11.009, AC-11.014 — metadata hostname rejected.
func TestValidateFetchURL_MetadataHostRejected(t *testing.T) {
	ctx := context.Background()
	err := ValidateFetchURL(ctx, "https://metadata.google.internal/path", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
