package skillindex

import (
	"context"
	"errors"
	"pa/internal/runtimeskills"
	"pa/internal/sqlitepragma"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"strings"
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
	st, err := sqlite.NewWithTable(db, 4, sqlite.TableSkills, sqlitepragma.RecommendedPolicy(false))
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

type failEmb struct{ err error }

func (f failEmb) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, f.err
}

// Covers AC-01.037: skill index build returns embedder errors to the caller as a required startup step.
func TestBuild_embedError_requiredAtStartup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := sqlite.NewWithTable(filepath.Join(dir, "v.db"), 4, sqlite.TableSkills, sqlitepragma.RecommendedPolicy(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	pkgs := []*runtimeskills.Package{
		{ID: "a", Name: "A", Description: "alpha", Body: "x"},
	}
	upstream := errors.New("config.embedding: quota exhausted")
	err = Build(ctx, pkgs, failEmb{err: upstream}, st)
	if err == nil {
		t.Fatal("Build: expected embed error")
	}
	got := err.Error()
	if !strings.Contains(got, "required at process startup") {
		t.Errorf("error %q: want required at process startup", got)
	}
	if !strings.Contains(got, "not a background job") {
		t.Errorf("error %q: want not a background job", got)
	}
	if !strings.Contains(got, upstream.Error()) {
		t.Errorf("error %q: want wrapped %q", got, upstream.Error())
	}
	if !errors.Is(err, upstream) {
		t.Errorf("Build: want errors.Is upstream, got %v", err)
	}
}
