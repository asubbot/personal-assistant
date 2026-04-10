package core

import (
	"fmt"
	"sync"
	"testing"
)

// Supporting AC-14.012: unit tests for sliding-window store behaviour.
// Covers AC-14.005: concurrent appends for one session key do not lose exchanges.
func TestSessionWindowStore_concurrentSameKey_noLostUpdates(t *testing.T) {
	s := newSessionWindowStore()
	const n = 40
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			s.appendExchange("k", fmt.Sprintf("u%d", i), fmt.Sprintf("a%d", i), 200)
		}(i)
	}
	wg.Wait()
	snap := s.snapshot("k")
	if len(snap) != n {
		t.Errorf("len(snapshot) = %d, want %d", len(snap), n)
	}
}

// Covers AC-14.008 / AC-14.010: cap evicts oldest; order preserved for sequential appends.
func TestSessionWindowStore_capEvictsOldest(t *testing.T) {
	s := newSessionWindowStore()
	s.appendExchange("k", "u1", "a1", 2)
	s.appendExchange("k", "u2", "a2", 2)
	s.appendExchange("k", "u3", "a3", 2)
	snap := s.snapshot("k")
	if len(snap) != 2 {
		t.Fatalf("len = %d, want 2", len(snap))
	}
	if snap[0].user != "u2" || snap[1].user != "u3" {
		t.Errorf("snapshot = %#v", snap)
	}
}
