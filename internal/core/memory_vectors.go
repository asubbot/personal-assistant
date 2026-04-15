package core

import (
	"errors"
	"pa/internal/vector"
)

// MemoryVectors holds EP-016 split sqlite-vec tables. Legacy is vec_items for summary-prefix search only; fields may be nil.
type MemoryVectors struct {
	Summaries vector.Store
	Turns     vector.Store
	Notes     vector.Store
	Legacy    vector.Store
}

func (m *MemoryVectors) anyNonNil() bool {
	return m != nil && (m.Notes != nil || m.Summaries != nil || m.Turns != nil || m.Legacy != nil)
}

// LegacyCompatMemoryVectors uses one store as legacy vec_items search plus turn indexing (pre-EP-016 tests and tooling).
func LegacyCompatMemoryVectors(s vector.Store) *MemoryVectors {
	if s == nil {
		return nil
	}
	return &MemoryVectors{Legacy: s, Turns: s}
}

func (m *MemoryVectors) legacySingleStoreCompat() bool {
	return m != nil && m.Legacy != nil && m.Turns == m.Legacy && m.Summaries == nil && m.Notes == nil
}

// Close closes each distinct non-nil store pointer once.
func (m *MemoryVectors) Close() error {
	if m == nil {
		return nil
	}
	seen := make(map[vector.Store]struct{})
	var errs []error
	for _, s := range []vector.Store{m.Summaries, m.Turns, m.Notes, m.Legacy} {
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
