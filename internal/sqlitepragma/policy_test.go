package sqlitepragma_test

import (
	"context"
	"database/sql"
	"pa/internal/sqlitepragma"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Covers AC-22.001, AC-22.002, AC-22.003: Policy self-validation before DSN build.
func TestPolicyValidate(t *testing.T) {
	cases := []struct {
		name    string
		p       sqlitepragma.Policy
		wantErr string
	}{
		{"ok", sqlitepragma.Policy{JournalMode: "WAL", BusyTimeout: time.Second, Synchronous: "NORMAL"}, ""},
		{"empty journal", sqlitepragma.Policy{BusyTimeout: time.Second, Synchronous: "NORMAL"}, "journal_mode"},
		{"zero busy", sqlitepragma.Policy{JournalMode: "WAL", Synchronous: "NORMAL"}, "busy_timeout"},
		{"empty sync", sqlitepragma.Policy{JournalMode: "WAL", BusyTimeout: time.Second}, "synchronous"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want err containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// Covers AC-22.001, AC-22.002, AC-22.003: DSN encodes PRAGMA policy.
func TestBuildDSN(t *testing.T) {
	p := sqlitepragma.Policy{JournalMode: "WAL", BusyTimeout: 5 * time.Second, Synchronous: "NORMAL", ForeignKeys: true}
	dsn, err := sqlitepragma.BuildDSN("/tmp/a.sqlite", p)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	for _, want := range []string{"file:/tmp/a.sqlite?", "_journal_mode=WAL", "_busy_timeout=5000", "_synchronous=NORMAL", "_foreign_keys=on"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn %q missing %q", dsn, want)
		}
	}
	if _, err := sqlitepragma.BuildDSN("", p); err == nil {
		t.Fatalf("empty path should fail")
	}
	if _, err := sqlitepragma.BuildDSN("/tmp/a.sqlite", sqlitepragma.Policy{}); err == nil {
		t.Fatalf("invalid policy should fail")
	}
}

// Covers AC-22.001, AC-22.002: VerifyOnOpen accepts matching PRAGMAs on a fresh connection.
func TestVerifyOnOpen_Matches(t *testing.T) {
	dir := t.TempDir()
	p := sqlitepragma.Policy{JournalMode: "WAL", BusyTimeout: 3 * time.Second, Synchronous: "NORMAL", ForeignKeys: true}
	dsn, err := sqlitepragma.BuildDSN(filepath.Join(dir, "x.sqlite"), p)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := sqlitepragma.VerifyOnOpen(context.Background(), db, p); err != nil {
		t.Fatalf("VerifyOnOpen: %v", err)
	}
}

// Covers AC-22.001, AC-22.002: VerifyOnOpen rejects mismatched PRAGMAs.
func TestVerifyOnOpen_Mismatch(t *testing.T) {
	dir := t.TempDir()
	built := sqlitepragma.Policy{JournalMode: "WAL", BusyTimeout: 3 * time.Second, Synchronous: "NORMAL", ForeignKeys: false}
	dsn, err := sqlitepragma.BuildDSN(filepath.Join(dir, "y.sqlite"), built)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	want := built
	want.BusyTimeout = 9 * time.Second
	if err := sqlitepragma.VerifyOnOpen(context.Background(), db, want); err == nil {
		t.Fatalf("expected mismatch error")
	}
}

// Covers AC-22.001, AC-22.002, AC-22.003: per-connection PRAGMAs are re-applied on a fresh pool connection.
func TestVerifyOnOpen_FreshConnectionHasPragmas(t *testing.T) {
	// Regression guard for REQ-22.001: pool gives a new connection, and it
	// must still carry the PRAGMAs (per-connection ones are re-applied via DSN).
	dir := t.TempDir()
	p := sqlitepragma.Policy{JournalMode: "WAL", BusyTimeout: 7 * time.Second, Synchronous: "NORMAL", ForeignKeys: true}
	dsn, err := sqlitepragma.BuildDSN(filepath.Join(dir, "z.sqlite"), p)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	// Force a second pooled connection by holding the first one.
	first, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("first conn: %v", err)
	}
	defer func() { _ = first.Close() }()

	second, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("second conn: %v", err)
	}
	defer func() { _ = second.Close() }()

	var busy int64
	if err := second.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if want := p.BusyTimeout.Milliseconds(); busy != want {
		t.Fatalf("busy_timeout on fresh connection: got %d want %d", busy, want)
	}
	var fk int
	if err := second.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("read foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys on fresh connection: got %d want 1", fk)
	}
}
