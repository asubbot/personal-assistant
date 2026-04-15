package core

import (
	"errors"
	"pa/internal/vector"
)

// MemoryVectors holds EP-016 split sqlite-vec tables; fields may be nil.
type MemoryVectors struct {
	Summaries vector.Store
	Turns     vector.Store
	Notes     vector.Store
}

func (m *MemoryVectors) anyNonNil() bool {
	return m != nil && (m.Notes != nil || m.Summaries != nil || m.Turns != nil)
}

// SingleStoreMemoryVectors uses one store for turn indexing/retrieval in tests and tooling.
func SingleStoreMemoryVectors(s vector.Store) *MemoryVectors {
	if s == nil {
		return nil
	}
	return &MemoryVectors{Turns: s}
}

// Close closes each distinct non-nil store pointer once.
func (m *MemoryVectors) Close() error {
	if m == nil {
		return nil
	}
	seen := make(map[vector.Store]struct{})
	var errs []error
	for _, s := range []vector.Store{m.Summaries, m.Turns, m.Notes} {
		if s == nil {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
