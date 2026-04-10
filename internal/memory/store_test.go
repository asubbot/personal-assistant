package memory

import (
	"context"
	"testing"
	"time"
)

// Covers AC-01.011, AC-01.012 (US-06): NewStore rejects empty rootDir (memory in designated dir, single store, calendar structure).
func TestNewStore_emptyRootDir(t *testing.T) {
	_, err := NewStore("", nil)
	if err == nil {
		t.Fatal("expected error for empty rootDir")
	}
	if err.Error() != "memory: rootDir is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): day summary path layout and WriteDaySummary/ReadDaySummary roundtrip.
func TestWriteDaySummary_ReadDaySummary_roundtrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	content := "Day summary: discussed X and Y."
	if err := store.WriteDaySummary(ctx, day, content); err != nil {
		t.Fatalf("WriteDaySummary: %v", err)
	}

	got, err := store.ReadDaySummary(ctx, day)
	if err != nil {
		t.Fatalf("ReadDaySummary: %v", err)
	}
	if got != content {
		t.Errorf("ReadDaySummary: got %q, want %q", got, content)
	}

	// Overwrite
	content2 := "Updated summary."
	if err := store.WriteDaySummary(ctx, day, content2); err != nil {
		t.Fatalf("WriteDaySummary overwrite: %v", err)
	}
	got, _ = store.ReadDaySummary(ctx, day)
	if got != content2 {
		t.Errorf("ReadDaySummary after overwrite: got %q, want %q", got, content2)
	}
}

// Supporting AC-01.012 (US-06): ReadDaySummary returns empty when summary file is missing.
func TestReadDaySummary_missingFile_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	got, err := store.ReadDaySummary(ctx, day)
	if err != nil {
		t.Fatalf("ReadDaySummary: %v", err)
	}
	if got != "" {
		t.Errorf("ReadDaySummary (missing): got %q, want empty", got)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): month summary written and read from calendar path rootDir/YYYY/MM/summary.md.
func TestWriteMonthSummary_ReadMonthSummary_roundtrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	content := "Month summary: key themes and tasks."
	if err := store.WriteMonthSummary(ctx, 2026, 3, content); err != nil {
		t.Fatalf("WriteMonthSummary: %v", err)
	}

	got, err := store.ReadMonthSummary(ctx, 2026, 3)
	if err != nil {
		t.Fatalf("ReadMonthSummary: %v", err)
	}
	if got != content {
		t.Errorf("ReadMonthSummary: got %q, want %q", got, content)
	}

	content2 := "Updated month summary."
	if err := store.WriteMonthSummary(ctx, 2026, 3, content2); err != nil {
		t.Fatalf("WriteMonthSummary overwrite: %v", err)
	}
	got, _ = store.ReadMonthSummary(ctx, 2026, 3)
	if got != content2 {
		t.Errorf("ReadMonthSummary after overwrite: got %q, want %q", got, content2)
	}
}

// Supporting AC-01.012 (US-06): ReadMonthSummary returns empty when file does not exist.
func TestReadMonthSummary_missing_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	got, err := store.ReadMonthSummary(ctx, 2025, 7)
	if err != nil {
		t.Fatalf("ReadMonthSummary: %v", err)
	}
	if got != "" {
		t.Errorf("ReadMonthSummary (missing): got %q, want empty", got)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): year summary written and read from calendar path rootDir/YYYY/summary.md.
func TestWriteYearSummary_ReadYearSummary_roundtrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	content := "Year summary: main achievements and themes."
	if err := store.WriteYearSummary(ctx, 2026, content); err != nil {
		t.Fatalf("WriteYearSummary: %v", err)
	}

	got, err := store.ReadYearSummary(ctx, 2026)
	if err != nil {
		t.Fatalf("ReadYearSummary: %v", err)
	}
	if got != content {
		t.Errorf("ReadYearSummary: got %q, want %q", got, content)
	}

	content2 := "Updated year summary."
	if err := store.WriteYearSummary(ctx, 2026, content2); err != nil {
		t.Fatalf("WriteYearSummary overwrite: %v", err)
	}
	got, _ = store.ReadYearSummary(ctx, 2026)
	if got != content2 {
		t.Errorf("ReadYearSummary after overwrite: got %q, want %q", got, content2)
	}
}

// Supporting AC-01.012 (US-06): ReadYearSummary returns empty when file does not exist.
func TestReadYearSummary_missing_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	got, err := store.ReadYearSummary(ctx, 2024)
	if err != nil {
		t.Fatalf("ReadYearSummary: %v", err)
	}
	if got != "" {
		t.Errorf("ReadYearSummary (missing): got %q, want empty", got)
	}
}
