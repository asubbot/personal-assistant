package config

import (
	"slices"
	"testing"
)

// Covers EP-021 AC-21.007 (Trace: REQ-21.007, REQ-21.008): native allowlist includes create_scheduled_job when jobs_db_path is set.
func TestAllowedNativeToolIDs_JobsDBPathIncludesCreateScheduledJob(t *testing.T) {
	c := &Config{
		Paths: Paths{JobsDBPath: "/data/jobs.sqlite"},
	}
	ids := AllowedNativeToolIDs(c)
	if !slices.Contains(ids, "create_scheduled_job") {
		t.Fatalf("AllowedNativeToolIDs = %v, want create_scheduled_job", ids)
	}
	if !NativeToolAllowed(c, "create_scheduled_job") {
		t.Fatal("NativeToolAllowed(create_scheduled_job) = false, want true")
	}
}

// Covers EP-021 AC-21.007: empty or whitespace-only jobs_db_path does not add create_scheduled_job.
func TestAllowedNativeToolIDs_EmptyJobsDBPathOmitsCreateScheduledJob(t *testing.T) {
	for _, path := range []string{"", "   ", "\t"} {
		c := &Config{Paths: Paths{JobsDBPath: path}}
		ids := AllowedNativeToolIDs(c)
		if slices.Contains(ids, "create_scheduled_job") {
			t.Fatalf("JobsDBPath %q: AllowedNativeToolIDs = %v, must not contain create_scheduled_job", path, ids)
		}
		if NativeToolAllowed(c, "create_scheduled_job") {
			t.Fatalf("JobsDBPath %q: NativeToolAllowed should be false", path)
		}
	}
}

// Covers AC-21.008 (Trace: REQ-21.008): nil Config must not expose create_scheduled_job in the native allowlist.
func TestAllowedNativeToolIDs_NilConfigExcludesCreateScheduledJob(t *testing.T) {
	ids := AllowedNativeToolIDs(nil)
	if slices.Contains(ids, "create_scheduled_job") {
		t.Fatalf("AllowedNativeToolIDs(nil) = %v, must not contain create_scheduled_job", ids)
	}
}

// Covers AC-31.008: native allowlist includes search_vector_memory for runtime skills.
func TestAllowedNativeToolIDs_IncludesSearchVectorMemory(t *testing.T) {
	ids := AllowedNativeToolIDs(&Config{})
	if !slices.Contains(ids, "search_vector_memory") {
		t.Fatalf("AllowedNativeToolIDs = %v, want search_vector_memory", ids)
	}
	if !NativeToolAllowed(&Config{}, "search_vector_memory") {
		t.Fatal("NativeToolAllowed(search_vector_memory) = false, want true")
	}
}

// Covers AC-32.014: native allowlist includes search_vector_tool and search_vector_skill.
func TestAllowedNativeToolIDs_IncludesSpecializedVectorKnowledgeTools(t *testing.T) {
	ids := AllowedNativeToolIDs(&Config{})
	for _, id := range []string{"search_vector_tool", "search_vector_skill"} {
		if !slices.Contains(ids, id) {
			t.Fatalf("AllowedNativeToolIDs = %v, want %s", ids, id)
		}
		if !NativeToolAllowed(&Config{}, id) {
			t.Fatalf("NativeToolAllowed(%s) = false, want true", id)
		}
	}
}
