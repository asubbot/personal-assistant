package memoryjob

import (
	"container/heap"
	"testing"
	"time"
)

// Covers AC-02.001 (partial): daily summarization job is enqueued only in local 01:xx window; same calendar fire key deduped.
func TestMaybeEnqueueDaily_windowAndDedup(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{deps: Deps{Loc: loc}}
	heap.Init(&r.pq)

	at0030 := time.Date(2026, 4, 10, 0, 30, 0, 0, loc)
	r.maybeEnqueueDaily(at0030, loc)
	if r.pq.Len() != 0 {
		t.Fatalf("off window: pq.Len=%d want 0", r.pq.Len())
	}

	at0105 := time.Date(2026, 4, 10, 1, 5, 0, 0, loc)
	r.maybeEnqueueDaily(at0105, loc)
	if r.pq.Len() != 1 {
		t.Fatalf("in window: pq.Len=%d want 1", r.pq.Len())
	}

	r.maybeEnqueueDaily(at0105, loc)
	if r.pq.Len() != 1 {
		t.Fatalf("same fire key: pq.Len=%d want 1", r.pq.Len())
	}

	nextDay := time.Date(2026, 4, 11, 1, 0, 0, 0, loc)
	r.maybeEnqueueDaily(nextDay, loc)
	if r.pq.Len() != 2 {
		t.Fatalf("next calendar day: pq.Len=%d want 2", r.pq.Len())
	}
}

// Covers AC-02.002 (partial): month rollup job enqueued only on first local day at 01:xx; deduped per month key.
func TestMaybeEnqueueMonthRollup_windowAndDedup(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{deps: Deps{Loc: loc}}
	heap.Init(&r.pq)

	mar2 := time.Date(2026, 3, 2, 1, 0, 0, 0, loc)
	r.maybeEnqueueMonthRollup(mar2, loc)
	if r.pq.Len() != 0 {
		t.Fatalf("not first of month: pq.Len=%d want 0", r.pq.Len())
	}

	mar1 := time.Date(2026, 3, 1, 1, 10, 0, 0, loc)
	r.maybeEnqueueMonthRollup(mar1, loc)
	if r.pq.Len() != 1 {
		t.Fatalf("first of month 01:xx: pq.Len=%d want 1", r.pq.Len())
	}

	r.maybeEnqueueMonthRollup(mar1, loc)
	if r.pq.Len() != 1 {
		t.Fatalf("same month key: pq.Len=%d want 1", r.pq.Len())
	}
}

// Covers AC-02.003 (partial): year rollup job enqueued only on Jan 1 local at 01:xx; deduped per year key.
func TestMaybeEnqueueYearRollup_windowAndDedup(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{deps: Deps{Loc: loc}}
	heap.Init(&r.pq)

	jan2 := time.Date(2027, 1, 2, 1, 0, 0, 0, loc)
	r.maybeEnqueueYearRollup(jan2, loc)
	if r.pq.Len() != 0 {
		t.Fatalf("not Jan 1: pq.Len=%d want 0", r.pq.Len())
	}

	jan1 := time.Date(2027, 1, 1, 1, 0, 0, 0, loc)
	r.maybeEnqueueYearRollup(jan1, loc)
	if r.pq.Len() != 1 {
		t.Fatalf("Jan 1 01:00: pq.Len=%d want 1", r.pq.Len())
	}

	r.maybeEnqueueYearRollup(jan1, loc)
	if r.pq.Len() != 1 {
		t.Fatalf("same year key: pq.Len=%d want 1", r.pq.Len())
	}
}
