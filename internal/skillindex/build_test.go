package skillindex

import (
	"context"
	"pa/internal/runtimeskills"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"testing"
)

type fakeEmb struct{}

func (fakeEmb) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return []float32{1, 0, 0, 0}, nil
}

func TestBuildAndSearch(t *testing.T) {
	// Covers AC-13.010
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "v.db")
	st, err := sqlite.NewWithTable(db, 4, sqlite.TableSkills)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	pkgs := []*runtimeskills.Package{
		{ID: "a", Name: "A", Description: "alpha", Body: "x"},
	}
	if err := Build(ctx, pkgs, fakeEmb{}, st); err != nil {
		t.Fatal(err)
	}
	ids, err := SearchSkillIDs(ctx, fakeEmb{}, st, "alpha", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "a" {
		t.Fatalf("got %v", ids)
	}
}
