package lifecyclelog

import (
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// Covers AC-29.005: lifecycle logs include subsystem, phase, and duration_ms.
func TestInfo_IncludesLifecycleFields(t *testing.T) {
	var buf strings.Builder
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)
	Info(logger, "memory_job", "job_complete", 5*time.Millisecond, "lifecycle", "job", "reconciliation_scan")
	if !strings.Contains(buf.String(), "lifecycle_event=true") {
		t.Fatalf("log: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "subsystem=memory_job") || !strings.Contains(buf.String(), "lifecycle_phase=job_complete") {
		t.Fatalf("log: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "duration_ms=5") {
		t.Fatalf("log: %q", buf.String())
	}
}

// Covers AC-29.005: error lifecycle path.
func TestError_IncludesError(t *testing.T) {
	var buf strings.Builder
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	logger := slog.New(h)
	err := errors.New("boom")
	Error(logger, "jobs_runtime", "init", 2*time.Millisecond, err, "lifecycle", "jobs_db_path", "x.sqlite")
	s := buf.String()
	if !strings.Contains(s, "error=boom") {
		t.Fatalf("log: %q", s)
	}
}

// Covers AC-29.005: nil logger is a no-op.
func TestInfo_nilLogger(t *testing.T) {
	Info(nil, "x", "y", 0, "lifecycle")
	Error(nil, "x", "y", 0, errors.New("e"), "lifecycle")
}
