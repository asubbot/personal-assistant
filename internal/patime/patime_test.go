package patime

import (
	"testing"
	"time"
)

// Covers AC-01.020: traceability for TestPreviousCalendarDate.
func TestPreviousCalendarDate(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	// 2026-03-12 10:00 MSK -> previous 2026-03-11
	tm := time.Date(2026, 3, 12, 10, 0, 0, 0, loc)
	y, m, d := PreviousCalendarDate(loc, tm)
	if y != 2026 || m != time.March || d != 11 {
		t.Fatalf("got %d-%02d-%02d, want 2026-03-11", y, m, d)
	}
}

// Covers AC-01.020: traceability for TestPreviousCalendarDate_USDST.
func TestPreviousCalendarDate_USDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	// Spring forward 2026-03-08 — noon anchor avoids ambiguous midnight
	tm := time.Date(2026, 3, 9, 15, 0, 0, 0, loc)
	y, m, d := PreviousCalendarDate(loc, tm)
	if y != 2026 || m != time.March || d != 8 {
		t.Fatalf("got %d-%02d-%02d, want 2026-03-08", y, m, d)
	}
}

// Covers AC-01.020: traceability for TestNextClockAfter.
func TestNextClockAfter(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 4, 10, 0, 30, 0, 0, loc)
	next := NextClockAfter(loc, from, 1, 0)
	want := time.Date(2026, 4, 10, 1, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %v want %v", next, want)
	}
	from2 := time.Date(2026, 4, 10, 2, 0, 0, 0, loc)
	next2 := NextClockAfter(loc, from2, 1, 0)
	want2 := time.Date(2026, 4, 11, 1, 0, 0, 0, loc)
	if !next2.Equal(want2) {
		t.Fatalf("got %v want %v", next2, want2)
	}
}

// Covers AC-01.020: traceability for TestPreviousMonth.
func TestPreviousMonth(t *testing.T) {
	loc := time.UTC
	tm := time.Date(2026, 3, 15, 0, 0, 0, 0, loc)
	y, m := PreviousMonth(loc, tm)
	if y != 2026 || m != time.February {
		t.Fatalf("got %d-%02d", y, m)
	}
}

// Covers AC-01.020: traceability for TestPreviousYear.
func TestPreviousYear(t *testing.T) {
	loc := time.UTC
	tm := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	if PreviousYear(loc, tm) != 2025 {
		t.Fatal("expected 2025")
	}
}
