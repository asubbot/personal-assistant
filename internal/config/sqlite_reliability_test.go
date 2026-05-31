package config

import (
	"testing"
)

// Covers AC-39.008, AC-39.009
func TestMergeSQLiteStoreReliability_InheritsDefaults(t *testing.T) {
	defaults := &SQLiteStoreDefaultsConfig{
		JournalMode: "WAL",
		BusyTimeout: "5s",
		Synchronous: "NORMAL",
	}
	fkFalse := false
	fkTrue := true
	vectorMerged, err := mergeSQLiteStoreReliability(defaults, &SQLiteStoreReliabilityOverride{ForeignKeys: &fkFalse})
	if err != nil {
		t.Fatalf("vector merge: %v", err)
	}
	jobsMerged, err := mergeSQLiteStoreReliability(defaults, &SQLiteStoreReliabilityOverride{ForeignKeys: &fkTrue})
	if err != nil {
		t.Fatalf("jobs merge: %v", err)
	}
	for _, tc := range []struct {
		name string
		got  *SQLiteStoreReliabilityConfig
		fk   bool
	}{
		{"vector", vectorMerged, false},
		{"jobs", jobsMerged, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got.JournalMode != "WAL" || tc.got.BusyTimeout != "5s" || tc.got.Synchronous != "NORMAL" {
				t.Fatalf("merged = %+v, want defaults inherited", tc.got)
			}
			if tc.got.ForeignKeys == nil || *tc.got.ForeignKeys != tc.fk {
				t.Fatalf("foreign_keys = %v, want %v", tc.got.ForeignKeys, tc.fk)
			}
		})
	}
}

// Covers AC-39.009
func TestMergeSQLiteStoreReliability_OverrideFields(t *testing.T) {
	defaults := &SQLiteStoreDefaultsConfig{
		JournalMode: "WAL",
		BusyTimeout: "5s",
		Synchronous: "NORMAL",
	}
	jm := "DELETE"
	bt := "10s"
	sync := "FULL"
	fk := true
	merged, err := mergeSQLiteStoreReliability(defaults, &SQLiteStoreReliabilityOverride{
		JournalMode: &jm,
		BusyTimeout: &bt,
		Synchronous: &sync,
		ForeignKeys: &fk,
	})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if merged.JournalMode != jm || merged.BusyTimeout != bt || merged.Synchronous != sync {
		t.Fatalf("merged = %+v, want overrides applied", merged)
	}
}

// Covers AC-39.014
func TestValidateVectorStoreReliability_MergedToPolicy(t *testing.T) {
	fkFalse := false
	cfg := &Config{
		SQLiteStoreDefaults: &SQLiteStoreDefaultsConfig{
			JournalMode: "WAL",
			BusyTimeout: "5s",
			Synchronous: "NORMAL",
		},
		VectorStoreReliability: &SQLiteStoreReliabilityOverride{ForeignKeys: &fkFalse},
	}
	if err := cfg.ValidateVectorStoreReliability(); err != nil {
		t.Fatalf("ValidateVectorStoreReliability: %v", err)
	}
	merged, err := mergeSQLiteStoreReliability(cfg.SQLiteStoreDefaults, cfg.VectorStoreReliability)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	p := merged.ToPolicy()
	if p.JournalMode != "WAL" || p.Synchronous != "NORMAL" || p.ForeignKeys != false {
		t.Fatalf("policy = %+v", p)
	}
	if p.BusyTimeout.String() != "5s" {
		t.Fatalf("BusyTimeout = %s, want 5s", p.BusyTimeout)
	}
}
