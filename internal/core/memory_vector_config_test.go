package core

import (
	"pa/internal/config"
	"testing"
)

// Supporting AC-01.014: the helper preserves uniform configured top-k across notes, summaries, and turns.
func TestUniformMemoryVectorConfigPopulatesAllLanes(t *testing.T) {
	const topK = 37

	got := uniformMemoryVectorConfig(topK)
	want := config.MemoryVectorConfig{
		NotesTopK:     topK,
		SummariesTopK: topK,
		TurnsTopK:     topK,
	}

	if got != want {
		t.Fatalf("uniformMemoryVectorConfig(%d) = %+v, want %+v", topK, got, want)
	}
}
