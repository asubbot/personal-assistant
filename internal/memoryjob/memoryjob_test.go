package memoryjob

import (
	"container/heap"
	"context"
	"testing"
)

// Covers AC-02.016: lower numeric priority runs before background summarization priority when dequeuing jobs.
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
	r.Enqueue(0, "interactive", func(context.Context) error {
		order = append(order, 0)
		return nil
	})
	r.drain(context.Background())
	if len(order) != 2 || order[0] != 0 || order[1] != 10 {
		t.Fatalf("run order = %v, want [0 10]", order)
	}
}
