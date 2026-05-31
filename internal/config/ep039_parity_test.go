package config

import (
	"pa/internal/sqlitepragma"
	"reflect"
	"testing"
)

func legacyReliabilityPolicy(journalMode, busyTimeout, synchronous string, foreignKeys bool) sqlitepragma.Policy {
	cfg := &SQLiteStoreReliabilityConfig{
		JournalMode: journalMode,
		BusyTimeout: busyTimeout,
		Synchronous: synchronous,
		ForeignKeys: &foreignKeys,
	}
	return cfg.ToPolicy()
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

// Covers AC-39.004
// Covers AC-39.019
func TestEP039_Parity_VectorSearchTools(t *testing.T) {
	tests := []struct {
		name string
		// want* are pre-EP-039 resolved settings (flat per-tool blocks merged to self).
		wantMemory VectorSearchToolConfig
		wantTool   VectorSearchToolConfig
		wantSkill  VectorSearchToolConfig
		post       *VectorSearchToolsConfig
	}{
		{
			name:       "uniform_legacy_defaults",
			wantMemory: VectorSearchToolConfig{Enabled: true, DefaultTopK: 5, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			wantTool:   VectorSearchToolConfig{Enabled: true, DefaultTopK: 5, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			wantSkill:  VectorSearchToolConfig{Enabled: true, DefaultTopK: 5, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			post: &VectorSearchToolsConfig{
				Defaults: defaultVectorSearchToolConfig(),
			},
		},
		{
			name:       "memory_tool_customized",
			wantMemory: VectorSearchToolConfig{Enabled: true, DefaultTopK: 4, MaxTopK: 9, MaxOutputBytes: 5000, SnippetRunes: 180},
			wantTool:   VectorSearchToolConfig{Enabled: true, DefaultTopK: 5, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			wantSkill:  VectorSearchToolConfig{Enabled: true, DefaultTopK: 5, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			post: &VectorSearchToolsConfig{
				Defaults: defaultVectorSearchToolConfig(),
				SearchVectorMemory: VectorSearchToolOverride{
					Enabled:        boolPtr(true),
					DefaultTopK:    intPtr(4),
					MaxTopK:        intPtr(9),
					MaxOutputBytes: intPtr(5000),
					SnippetRunes:   intPtr(180),
				},
			},
		},
		{
			name:       "skill_disabled_partial_tool_override",
			wantMemory: VectorSearchToolConfig{Enabled: true, DefaultTopK: 5, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			wantTool:   VectorSearchToolConfig{Enabled: true, DefaultTopK: 3, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			wantSkill:  VectorSearchToolConfig{Enabled: false, DefaultTopK: 5, MaxTopK: 10, MaxOutputBytes: 4096, SnippetRunes: 200},
			post: &VectorSearchToolsConfig{
				Defaults: defaultVectorSearchToolConfig(),
				SearchVectorTool: VectorSearchToolOverride{
					DefaultTopK: intPtr(3),
				},
				SearchVectorSkill: VectorSearchToolOverride{
					Enabled: boolPtr(false),
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Tools: &ToolsConfig{VectorSearchTools: tc.post},
			}
			for toolID, want := range map[string]VectorSearchToolConfig{
				"search_vector_memory": tc.wantMemory,
				"search_vector_tool":   tc.wantTool,
				"search_vector_skill":  tc.wantSkill,
			} {
				got := cfg.VectorSearchToolSettings(toolID)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s: got %+v want %+v (pre-legacy baseline)", toolID, got, want)
				}
			}
		})
	}
}

// Covers AC-39.004
// Covers AC-39.019
func TestEP039_Parity_SQLiteReliability(t *testing.T) {
	defaults := &SQLiteStoreDefaultsConfig{
		JournalMode: "WAL",
		BusyTimeout: "5s",
		Synchronous: "NORMAL",
	}
	tests := []struct {
		name       string
		wantVector sqlitepragma.Policy
		wantJobs   sqlitepragma.Policy
		vectorOvr  *SQLiteStoreReliabilityOverride
		jobsOvr    *SQLiteStoreReliabilityOverride
	}{
		{
			name:       "shared_defaults_differing_foreign_keys",
			wantVector: legacyReliabilityPolicy("WAL", "5s", "NORMAL", false),
			wantJobs:   legacyReliabilityPolicy("WAL", "5s", "NORMAL", true),
			vectorOvr:  &SQLiteStoreReliabilityOverride{ForeignKeys: boolPtr(false)},
			jobsOvr:    &SQLiteStoreReliabilityOverride{ForeignKeys: boolPtr(true)},
		},
		{
			name:       "vector_journal_mode_override",
			wantVector: legacyReliabilityPolicy("DELETE", "5s", "NORMAL", false),
			wantJobs:   legacyReliabilityPolicy("WAL", "5s", "NORMAL", true),
			vectorOvr: &SQLiteStoreReliabilityOverride{
				JournalMode: strPtr("DELETE"),
				ForeignKeys: boolPtr(false),
			},
			jobsOvr: &SQLiteStoreReliabilityOverride{ForeignKeys: boolPtr(true)},
		},
		{
			name:       "jobs_busy_timeout_and_synchronous_override",
			wantVector: legacyReliabilityPolicy("WAL", "5s", "NORMAL", false),
			wantJobs:   legacyReliabilityPolicy("WAL", "10s", "FULL", true),
			vectorOvr:  &SQLiteStoreReliabilityOverride{ForeignKeys: boolPtr(false)},
			jobsOvr: &SQLiteStoreReliabilityOverride{
				BusyTimeout: strPtr("10s"),
				Synchronous: strPtr("FULL"),
				ForeignKeys: boolPtr(true),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vectorMerged, err := mergeSQLiteStoreReliability(defaults, tc.vectorOvr)
			if err != nil {
				t.Fatalf("vector merge: %v", err)
			}
			jobsMerged, err := mergeSQLiteStoreReliability(defaults, tc.jobsOvr)
			if err != nil {
				t.Fatalf("jobs merge: %v", err)
			}
			gotVector := vectorMerged.ToPolicy()
			gotJobs := jobsMerged.ToPolicy()
			if !reflect.DeepEqual(gotVector, tc.wantVector) {
				t.Fatalf("vector policy: got %+v want %+v (pre-legacy full block)", gotVector, tc.wantVector)
			}
			if !reflect.DeepEqual(gotJobs, tc.wantJobs) {
				t.Fatalf("jobs policy: got %+v want %+v (pre-legacy full block)", gotJobs, tc.wantJobs)
			}
		})
	}
}

func strPtr(v string) *string { return &v }
