package tools

import (
	"context"
	"pa/internal/memory"
	"strings"
	"testing"
	"time"
)

// Covers AC-02.010: read_memory returns day summary for valid ISO date; reads only resolved paths under memory_dir.
func TestReadMemoryTool_singleDate(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	if err := store.WriteDaySummary(context.Background(), day, "hello memory"); err != nil {
		t.Fatal(err)
	}
	tool := NewReadMemoryTool(store, 31, 10240)
	out, err := tool.Run(context.Background(), map[string]any{"date": "2026-04-01"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello memory") {
		t.Fatalf("got %q", out)
	}
}

// Covers AC-02.011, AC-16.008: read_memory rejects range wider than max_span_days (span limit error).
func TestReadMemoryTool_rangeTooLarge(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	tool := NewReadMemoryTool(store, 2, 10240)
	_, err = tool.Run(context.Background(), map[string]any{"from": "2026-04-01", "to": "2026-04-10"})
	if err == nil || !strings.Contains(err.Error(), "range spans") {
		t.Fatalf("expected range error, got %v", err)
	}
}

// Covers AC-02.011: read_memory rejects when assembled output would exceed max_output_bytes.
func TestReadMemoryTool_outputExceedsMaxBytes(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	big := strings.Repeat("x", 500)
	if err := store.WriteDaySummary(context.Background(), day, big); err != nil {
		t.Fatal(err)
	}
	tool := NewReadMemoryTool(store, 31, 80)
	_, err = tool.Run(context.Background(), map[string]any{"date": "2026-04-01"})
	if err == nil || !strings.Contains(err.Error(), "max_output_bytes") {
		t.Fatalf("expected max_output_bytes error, got %v", err)
	}
}

// Covers AC-02.010: read_memory rejects invalid ISO dates, path-like strings, and conflicting parameters (no path escape).
func TestReadMemoryTool_rejectsInvalidISOAndConflictingParams(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	tool := NewReadMemoryTool(store, 31, 10240)
	ctx := context.Background()

	t.Run("invalid_iso", func(t *testing.T) {
		for _, date := range []string{
			"2026-13-01",
			"2026-04-31",
			"2026-4-01",
			"06-04-01",
			"2026/04/01",
			"../../../etc/passwd",
			"..\\..\\windows",
		} {
			t.Run(date, func(t *testing.T) {
				_, err := tool.Run(ctx, map[string]any{"date": date})
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), "invalid ISO") && !strings.Contains(err.Error(), "empty date") {
					t.Fatalf("date %q: want invalid ISO or empty date, got %v", date, err)
				}
			})
		}
	})

	t.Run("whitespace_date_rejected", func(t *testing.T) {
		_, err := tool.Run(ctx, map[string]any{"date": "   "})
		if err == nil || !strings.Contains(err.Error(), "provide date or both from and to") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("date_and_range", func(t *testing.T) {
		_, err := tool.Run(ctx, map[string]any{"date": "2026-04-01", "from": "2026-04-01", "to": "2026-04-02"})
		if err == nil || !strings.Contains(err.Error(), "either date or from/to") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("range_invalid_from", func(t *testing.T) {
		_, err := tool.Run(ctx, map[string]any{"from": "../x", "to": "2026-04-02"})
		if err == nil || !strings.Contains(err.Error(), "from:") {
			t.Fatalf("got %v", err)
		}
	})
}
