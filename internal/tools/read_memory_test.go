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

// Covers AC-02.011: read_memory rejects range wider than max_span_days.
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
