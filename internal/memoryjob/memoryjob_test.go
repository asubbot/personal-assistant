package memoryjob

import (
	"container/heap"
	"context"
	"log/slog"
	"testing"
)

// Supporting AC-02.016: among jobs eligible to run (no user-turn block), lower numeric priority runs first.
func TestRunner_drain_orderLowerPriorityFirst(t *testing.T) {
	r := &Runner{
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	var order []int
	r.Enqueue(10, "bg", func(context.Context) error {
		order = append(order, 10)
		return nil
	})
	r.Enqueue(0, "low_pri", func(context.Context) error {
		order = append(order, 0)
		return nil
	})
	r.drain(context.Background())
	if len(order) != 2 || order[0] != 0 || order[1] != 10 {
		t.Fatalf("run order = %v, want [0 10]", order)
	}
}

// Covers AC-02.016: scheduled summarization (priority 10) is not executed while UserTurnActive; runs after it clears.
func TestRunner_drain_defersScheduledDuringUserTurn(t *testing.T) {
	userTurn := true
	var ran int
	r := &Runner{
		deps: Deps{
			UserTurnActive: func() bool { return userTurn },
			Logger:         slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.Enqueue(PriorityScheduled, "summarize_test", func(context.Context) error {
		ran++
		return nil
	})
	r.drain(context.Background())
	if ran != 0 {
		t.Fatalf("expected 0 runs during user turn, got %d", ran)
	}
	userTurn = false
	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("expected 1 run after user turn, got %d", ran)
	}
}

// Covers AC-02.016: catch-up priority (5) is not executed while UserTurnActive; runs after it clears.
func TestRunner_drain_defersCatchUpDuringUserTurn(t *testing.T) {
	userTurn := true
	var ran int
	r := &Runner{
		deps: Deps{
			UserTurnActive: func() bool { return userTurn },
			Logger:         slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.Enqueue(PriorityCatchUp, "catchup_test", func(context.Context) error {
		ran++
		return nil
	})
	r.drain(context.Background())
	if ran != 0 {
		t.Fatalf("expected 0 runs during user turn, got %d", ran)
	}
	userTurn = false
	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("expected 1 run after user turn, got %d", ran)
	}
}

// Covers AC-02.016 / design trade-off: reconciliation (priority 4) still runs while UserTurnActive.
func TestRunner_drain_reconcileNotDeferredDuringUserTurn(t *testing.T) {
	userTurn := true
	var ran int
	r := &Runner{
		deps: Deps{
			UserTurnActive: func() bool { return userTurn },
			Logger:         slog.New(slog.DiscardHandler),
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		stop: func() {},
	}
	heap.Init(&r.pq)
	r.Enqueue(PriorityReconcile, "recon_test", func(context.Context) error {
		ran++
		return nil
	})
	r.drain(context.Background())
	if ran != 1 {
		t.Fatalf("expected reconcile to run during user turn, got ran=%d", ran)
	}
}
