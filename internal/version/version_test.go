package version

import "testing"

// Supporting AC-24.007: build identity embedded at link time for operator Docker/Makefile builds.
func TestString_defaults(t *testing.T) {
	oldCommit, oldTime := Commit, BuildTime
	t.Cleanup(func() {
		Commit, BuildTime = oldCommit, oldTime
	})
	Commit = "unknown"
	BuildTime = "unknown"
	if got := String(); got != "commit=unknown built=unknown" {
		t.Fatalf("String() = %q, want commit=unknown built=unknown", got)
	}
}

// Supporting AC-24.007: build identity string format for -version and startup logs.
func TestString_injectedValues(t *testing.T) {
	oldCommit, oldTime := Commit, BuildTime
	t.Cleanup(func() {
		Commit, BuildTime = oldCommit, oldTime
	})
	Commit = "abc1234"
	BuildTime = "2026-07-09T13:19:00Z"
	if got := String(); got != "commit=abc1234 built=2026-07-09T13:19:00Z" {
		t.Fatalf("String() = %q, want commit=abc1234 built=2026-07-09T13:19:00Z", got)
	}
}
