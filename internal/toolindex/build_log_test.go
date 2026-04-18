package toolindex

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"
)

// slogCapture records slog records for AC-04.021 assertions.
type slogCapture struct {
	level   slog.Level
	records []struct {
		level slog.Level
		msg   string
		attrs map[string]string
	}
}

func (c *slogCapture) Enabled(_ context.Context, level slog.Level) bool {
	return level >= c.level
}

func (c *slogCapture) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	c.records = append(c.records, struct {
		level slog.Level
		msg   string
		attrs map[string]string
	}{r.Level, r.Message, attrs})
	return nil
}

func (c *slogCapture) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *slogCapture) WithGroup(string) slog.Handler      { return c }

// AC-04.021 / REQ-04.025: successful build logs INFO with tool count (EP-029 lifecycle schema).
func TestLogBuildOutcome_success_infoWithTools(t *testing.T) {
	cap := &slogCapture{level: slog.LevelInfo}
	LogBuildOutcome(slog.New(cap), 7, 12*time.Millisecond, nil)
	if len(cap.records) != 1 {
		t.Fatalf("want 1 log record, got %d: %+v", len(cap.records), cap.records)
	}
	r := cap.records[0]
	if r.level != slog.LevelInfo || r.msg != "lifecycle" {
		t.Errorf("record = level=%v msg=%q, want INFO lifecycle", r.level, r.msg)
	}
	if r.attrs["tool_count"] != "7" {
		t.Errorf("tool_count attr = %q, want 7", r.attrs["tool_count"])
	}
	if r.attrs["lifecycle_event"] != "true" || r.attrs["subsystem"] != "tool_index" {
		t.Errorf("attrs = %#v", r.attrs)
	}
}

// AC-04.021 / REQ-04.025: build failure logs ERROR with reason.
func TestLogBuildOutcome_failure_errorWithReason(t *testing.T) {
	cap := &slogCapture{level: slog.LevelError}
	LogBuildOutcome(slog.New(cap), 0, 3*time.Millisecond, fmt.Errorf("embedding API unavailable"))
	if len(cap.records) != 1 {
		t.Fatalf("want 1 log record, got %d", len(cap.records))
	}
	r := cap.records[0]
	if r.level != slog.LevelError || r.msg != "lifecycle" {
		t.Errorf("record = level=%v msg=%q", r.level, r.msg)
	}
	if want := "embedding API unavailable"; r.attrs["error"] != want {
		t.Errorf("error attr = %q, want %q", r.attrs["error"], want)
	}
}

// Covers AC-04.021: traceability for TestLogBuildOutcome_nilLogger_noPanic.
func TestLogBuildOutcome_nilLogger_noPanic(t *testing.T) {
	LogBuildOutcome(nil, 1, 0, nil)
	LogBuildOutcome(nil, 0, 0, fmt.Errorf("x"))
}
