package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Supporting AC-24.007: -version prints embedded commit and build time without loading config.
func TestCLI_versionFlag(t *testing.T) {
	root := repoRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ldflags := "-X pa/internal/version.Commit=testcommit -X pa/internal/version.BuildTime=2026-07-09T12:00:00Z"
	cmd := exec.CommandContext(ctx, "go", "run", "-ldflags", ldflags, "./cmd/pa", "-version")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run -version: %v\n%s", err, out)
	}
	text := strings.TrimSpace(string(out))
	want := "pa commit=testcommit built=2026-07-09T12:00:00Z"
	if text != want {
		t.Fatalf("stdout = %q, want %q", text, want)
	}
}
