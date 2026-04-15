package summarize

import "testing"

// Covers AC-02.009: vector id prefixes map to retrieval chunk labels consumed by retrieval assembly.
func TestVectorChunkLabel(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"notes:2024-01-01:x", "notes"},
		{"turn:2024-01-01:abc", "turn"},
		{VectorIDPrefixDay + "2024-01-01", "summary:day"},
		{VectorIDPrefixMonth + "2024-01", "summary:month"},
		{VectorIDPrefixYear + "2024", "summary:year"},
		{"weird:prefix:unknown", "unknown"},
		{"", "unknown"},
	}
	for _, tc := range tests {
		if got := VectorChunkLabel(tc.id); got != tc.want {
			t.Errorf("VectorChunkLabel(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
