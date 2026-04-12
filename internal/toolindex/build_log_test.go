package toolindex

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
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

// AC-04.021 / REQ-04.025: successful build logs INFO with tool count.
func TestLogBuildOutcome_success_infoWithTools(t *testing.T) {
	cap := &slogCapture{level: slog.LevelInfo}
	LogBuildOutcome(slog.New(cap), 7, nil)
	if len(cap.records) != 1 {
		t.Fatalf("want 1 log record, got %d: %+v", len(cap.records), cap.records)
	}
	r := cap.records[0]
	if r.level != slog.LevelInfo || r.msg != "tool index built" {
		t.Errorf("record = level=%v msg=%q, want INFO tool index built", r.level, r.msg)
	}
	if r.attrs["tools"] != "7" {
		t.Errorf("tools attr = %q, want 7", r.attrs["tools"])
	}
}

// AC-04.021 / REQ-04.025: build failure logs ERROR with reason.
func TestLogBuildOutcome_failure_errorWithReason(t *testing.T) {
	cap := &slogCapture{level: slog.LevelError}
	LogBuildOutcome(slog.New(cap), 0, fmt.Errorf("embedding API unavailable"))
	if len(cap.records) != 1 {
		t.Fatalf("want 1 log record, got %d", len(cap.records))
	}
	r := cap.records[0]
	if r.level != slog.LevelError || r.msg != "tool index build failed" {
		t.Errorf("record = level=%v msg=%q", r.level, r.msg)
	}
	if want := "embedding API unavailable"; r.attrs["error"] != want {
		t.Errorf("error attr = %q, want %q", r.attrs["error"], want)
	}
}

// Covers AC-04.021: traceability for TestLogBuildOutcome_nilLogger_noPanic.
func TestLogBuildOutcome_nilLogger_noPanic(t *testing.T) {
	LogBuildOutcome(nil, 1, nil)
	LogBuildOutcome(nil, 0, fmt.Errorf("x"))
}
