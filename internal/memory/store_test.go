package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewStore_emptyRootDir
// Validates: AC-011, AC-012 (REQ-006, REQ-018, REQ-019 — memory in designated dir, single store, calendar structure)
func TestNewStore_emptyRootDir(t *testing.T) {
	_, err := NewStore("")
	if err == nil {
		t.Fatal("expected error for empty rootDir")
	}
	if err.Error() != "memory: rootDir is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestWriteDay_createsCalendarPath
// Validates: AC-011 (REQ-006, REQ-019 — files created in calendar structure year/month/day)
func TestWriteDay_createsCalendarPath(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	content := "# Day note\n\nSome content."
	if err := store.WriteDay(ctx, day, content); err != nil {
		t.Fatalf("WriteDay: %v", err)
	}

	wantPath := filepath.Join(dir, "2026", "03", "09", "full.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected file at %s: %v", wantPath, err)
	}
}

// TestWriteDay_ReadDay_roundtrip
// Validates: AC-011, AC-012 (REQ-006 — read/write markdown in designated structure)
func TestWriteDay_ReadDay_roundtrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	content := "Memory content for the day."
	if err := store.WriteDay(ctx, day, content); err != nil {
		t.Fatalf("WriteDay: %v", err)
	}

	got, err := store.ReadDay(ctx, day)
	if err != nil {
		t.Fatalf("ReadDay: %v", err)
	}
	if got != content {
		t.Errorf("ReadDay: got %q, want %q", got, content)
	}
}

// TestReadDay_missingFile_returnsEmpty
// Validates: AC-012 (REQ-006 — read from structure; missing file returns empty)
func TestReadDay_missingFile_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := store.ReadDay(ctx, day)
	if err != nil {
		t.Fatalf("ReadDay: %v", err)
	}
	if got != "" {
		t.Errorf("ReadDay (missing): got %q, want empty", got)
	}
}

// TestStore_singleStore_noPerUserPaths
// Validates: AC-011, AC-012 (REQ-018 — single store, not subdivided by interlocutor; paths are date-only)
func TestStore_singleStore_noPerUserPaths(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	path := store.pathForDay(day)
	// Path must be rootDir/YYYY/MM/DD/full.md — no user id or interlocutor segment
	if filepath.Base(path) != "full.md" {
		t.Errorf("file must be full.md: got %s", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != "01" || filepath.Base(filepath.Dir(filepath.Dir(path))) != "04" {
		t.Errorf("path must be calendar-only (year/month/day): got %s", path)
	}

	_ = store.WriteDay(ctx, day, "content")
	got, _ := store.ReadDay(ctx, day)
	if got != "content" {
		t.Errorf("ReadDay: got %q", got)
	}
}

// TestWriteDaySummary_ReadDaySummary_roundtrip — day summary path layout and overwrite.
func TestWriteDaySummary_ReadDaySummary_roundtrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
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

// TestReadDaySummary_missingFile_returnsEmpty — missing summary file returns empty string.
func TestReadDaySummary_missingFile_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
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

// Covers AC-011, AC-012 (US-06): month summary written and read from calendar path rootDir/YYYY/MM/summary.md.
func TestWriteMonthSummary_ReadMonthSummary_roundtrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
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

// Supporting AC-012 (US-06): ReadMonthSummary returns empty when file does not exist.
func TestReadMonthSummary_missing_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
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

// Covers AC-011, AC-012 (US-06): year summary written and read from calendar path rootDir/YYYY/summary.md.
func TestWriteYearSummary_ReadYearSummary_roundtrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
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

// Supporting AC-012 (US-06): ReadYearSummary returns empty when file does not exist.
func TestReadYearSummary_missing_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := NewStore(dir)
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
