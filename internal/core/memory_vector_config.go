package core

import "pa/internal/config"

func uniformMemoryVectorConfig(topK int) config.MemoryVectorConfig {
	return config.MemoryVectorConfig{
		NotesTopK:     topK,
		SummariesTopK: topK,
		TurnsTopK:     topK,
	}
}
