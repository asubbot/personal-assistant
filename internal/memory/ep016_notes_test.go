package memory

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Covers AC-16.001: notes.md path for a calendar day uses YYYY/MM/DD under memory_dir in pa_timezone.
func TestNotesPathForDay_AmsterdamLayout(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	store, err := NewStore(dir, loc)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 4, 10, 9, 0, 0, 0, loc)
	want := filepath.Join(dir, "2026", "04", "10", "notes.md")
	if got := store.NotesPathForDay(day); got != want {
		t.Fatalf("NotesPathForDay = %q, want %q", got, want)
	}
}

// Covers AC-16.002: WriteDaySummary overwrites summary.md only; notes.md content is preserved.
func TestWriteDaySummary_preservesNotesMd(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	day := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	if err := store.AppendDayNote(ctx, day, "preserve-me", "", time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC), 4096, 1<<20); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteDaySummary(ctx, day, "new automatic summary"); err != nil {
		t.Fatal(err)
	}
	notes, err := store.ReadDayNotes(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "preserve-me") {
		t.Fatalf("notes.md should still contain marker; got %q", notes)
	}
}

// Covers AC-16.003: AppendDayNote entry begins with RFC3339 UTC (Z) and includes submitted text.
func TestAppendDayNote_startsWithRFC3339Z(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	day := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 1, 14, 30, 0, 0, time.UTC)
	if err := store.AppendDayNote(ctx, day, "hello note body", "", now, 4096, 1<<20); err != nil {
		t.Fatal(err)
	}
	out, err := store.ReadDayNotes(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	firstLine := strings.Split(out, "\n")[0]
	if !strings.HasSuffix(firstLine, "Z") || !strings.Contains(firstLine, "T") {
		t.Fatalf("expected RFC3339 UTC first line, got %q", firstLine)
	}
	if !strings.Contains(out, "hello note body") {
		t.Fatalf("expected body text in file: %q", out)
	}
}

// Covers AC-16.004: AppendDayNote rejects when entry exceeds max_append_bytes and message names the limit.
func TestAppendDayNote_rejectsOversizedEntry(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	day := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	err = store.AppendDayNote(ctx, day, "too long", "", time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC), 10, 1<<20)
	if err == nil || !strings.Contains(err.Error(), "max_append_bytes") {
		t.Fatalf("want max_append_bytes error, got %v", err)
	}
}
