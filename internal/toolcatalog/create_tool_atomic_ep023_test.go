package toolcatalog

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Covers AC-23.004, AC-23.008, Supporting AC-23.010: post-write Load failure restores prior catalog bytes; assertions fail fast.
func TestAppendToolToCatalogFile_postWriteInvalidRestores(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	initial := []byte("tools: []\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	tool := &Tool{ID: "x", IndexText: "i", Template: "echo x", NodeID: "n"}
	testPostMarshalHook = func([]byte) []byte { return []byte("not: valid: yaml: [[[\n") }
	defer func() { testPostMarshalHook = nil }()
	if err := AppendToolToCatalogFile(path, tool); err == nil {
		t.Fatal("expected error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, initial) {
		t.Fatalf("catalog bytes changed\nwas: %q\ngot: %q", initial, got)
	}
}

// Covers AC-23.008: rename failure leaves original catalog file unchanged.
func TestAppendToolToCatalogFile_renameFailsUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	initial := []byte("tools: []\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	tool := &Tool{ID: "a", IndexText: "i", Template: "echo a", NodeID: "n"}
	testRenameHook = func(_, _ string) error { return errors.New("injected rename failure") }
	defer func() { testRenameHook = nil }()
	if err := AppendToolToCatalogFile(path, tool); err == nil {
		t.Fatal("expected error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, initial) {
		t.Fatalf("catalog bytes changed: %q", got)
	}
}

// Covers AC-23.008: truncated body fails validation and restores snapshot.
func TestAppendToolToCatalogFile_shortWriteRestores(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	initial := []byte("tools: []\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	tool := &Tool{ID: "b", IndexText: "i", Template: "echo b", NodeID: "n"}
	testPostMarshalHook = func(b []byte) []byte {
		if len(b) < 12 {
			return b
		}
		return b[:len(b)-10]
	}
	defer func() { testPostMarshalHook = nil }()
	if err := AppendToolToCatalogFile(path, tool); err == nil {
		t.Fatal("expected error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, initial) {
		t.Fatalf("catalog bytes changed: %q", got)
	}
}

// Covers AC-23.002: persistence path syncs temp file data and parent directory.
func TestAppendToolToCatalogFile_invokesDataAndDirSync(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	if err := os.WriteFile(path, []byte("tools: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var nData, nDir int
	testOnTempDataSync = func() { nData++ }
	testOnDirSync = func() { nDir++ }
	defer func() {
		testOnTempDataSync = nil
		testOnDirSync = nil
	}()
	tool := &Tool{ID: "s", IndexText: "i", Template: "echo s", NodeID: "n"}
	if err := AppendToolToCatalogFile(path, tool); err != nil {
		t.Fatal(err)
	}
	if nData != 1 || nDir != 1 {
		t.Fatalf("sync hooks: temp data=%d dir=%d", nData, nDir)
	}
}
