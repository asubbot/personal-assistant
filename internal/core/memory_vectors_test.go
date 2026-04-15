package core

import (
	"context"
	"errors"
	"pa/internal/vector"
	"strings"
	"testing"
)

type closeSpyStore struct {
	closeErr error
	closes   int
}

func (s *closeSpyStore) Add(context.Context, string, []float32, string) error { return nil }
func (s *closeSpyStore) Delete(context.Context, string) error                 { return nil }
func (s *closeSpyStore) Clear(context.Context) error                          { return nil }
func (s *closeSpyStore) Search(context.Context, []float32, int) ([]vector.SearchResult, error) {
	return nil, nil
}
func (s *closeSpyStore) Exists(context.Context, string) (bool, error) { return false, nil }
func (s *closeSpyStore) Close() error {
	s.closes++
	return s.closeErr
}

// Covers AC-16.023: MemoryVectors.Close closes shared pointer once.
func TestMemoryVectorsClose_DeduplicatesSamePointer(t *testing.T) {
	s := &closeSpyStore{}
	mv := &MemoryVectors{
		Summaries: s,
		Turns:     s,
		Notes:     s,
	}
	if err := mv.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if s.closes != 1 {
		t.Fatalf("Close() count = %d, want 1", s.closes)
	}
}

// Covers AC-16.023: MemoryVectors.Close returns joined close errors.
func TestMemoryVectorsClose_ReturnsJoinedErrors(t *testing.T) {
	s1 := &closeSpyStore{closeErr: errors.New("close summaries")}
	s2 := &closeSpyStore{closeErr: errors.New("close turns")}
	mv := &MemoryVectors{Summaries: s1, Turns: s2}
	err := mv.Close()
	if err == nil {
		t.Fatal("Close() error = nil, want joined error")
	}
	if !strings.Contains(err.Error(), "close summaries") || !strings.Contains(err.Error(), "close turns") {
		t.Fatalf("Close() error = %q, want both close errors", err.Error())
	}
}
