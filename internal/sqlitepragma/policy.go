// Package sqlitepragma centralizes the SQLite PRAGMA policy applied to every
// connection opened by the Local SQLite Stores (vector store and jobs store).
//
// The policy is encoded as DSN query parameters supported by the
// mattn/go-sqlite3 driver, so the driver applies the per-connection PRAGMAs
// (busy_timeout, synchronous, foreign_keys) on every new connection the
// database/sql pool hands out. WAL journal_mode is a database-level setting
// that is persisted on first application.
//
// A fresh-connection self-check (VerifyOnOpen) reads PRAGMAs back to fail
// fast when the driver rejects a value or the DSN is malformed.
package sqlitepragma

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Policy is the explicit set of PRAGMA values applied on every new connection
// for a single Local SQLite Store. Every field is required; the config loader
// rejects empty/zero values. There are no hidden defaults at config load; see
// RecommendedPolicy for reference values used in documentation and tests.
type Policy struct {
	JournalMode string        // e.g. "WAL"; required, non-empty
	BusyTimeout time.Duration // > 0
	Synchronous string        // e.g. "NORMAL"; required, non-empty
	ForeignKeys bool          // jobs store must be true; vector store must be false
}

// Validate returns an error if any field is unset or invalid.
func (p Policy) Validate() error {
	if strings.TrimSpace(p.JournalMode) == "" {
		return errors.New("sqlitepragma: journal_mode is required")
	}
	if p.BusyTimeout <= 0 {
		return fmt.Errorf("sqlitepragma: busy_timeout must be > 0, got %s", p.BusyTimeout)
	}
	if strings.TrimSpace(p.Synchronous) == "" {
		return errors.New("sqlitepragma: synchronous is required")
	}
	return nil
}

// RecommendedPolicy returns reference values matching the operator
// documentation. Production code must not rely on this: the config loader
// fails fast when PRAGMA fields are missing. It is exposed for tests and for
// documentation cross-references only.
func RecommendedPolicy(foreignKeys bool) Policy {
	return Policy{
		JournalMode: "WAL",
		BusyTimeout: 5 * time.Second,
		Synchronous: "NORMAL",
		ForeignKeys: foreignKeys,
	}
}

// BuildDSN returns a mattn/go-sqlite3 DSN of the form
// file:<path>?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on
// so the driver re-applies per-connection PRAGMAs on every new pool
// connection. path must be a filesystem path; the function escapes it.
func BuildDSN(path string, p Policy) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("sqlitepragma: path is required")
	}
	if err := p.Validate(); err != nil {
		return "", err
	}
	q := url.Values{}
	q.Set("_journal_mode", p.JournalMode)
	q.Set("_busy_timeout", fmt.Sprintf("%d", p.BusyTimeout.Milliseconds()))
	q.Set("_synchronous", p.Synchronous)
	// Encode ForeignKeys explicitly in both directions. The vector store must
	// stay FK=off because vec0 virtual tables reject FK enforcement; writing
	// "off" makes the policy observable in logs and VerifyOnOpen symmetric
	// for both stores.
	if p.ForeignKeys {
		q.Set("_foreign_keys", "on")
	} else {
		q.Set("_foreign_keys", "off")
	}
	return "file:" + path + "?" + q.Encode(), nil
}

// VerifyOnOpen opens a fresh connection and asserts that every PRAGMA in p is
// reflected by the driver. Returns an error naming the mismatched PRAGMA when
// the driver silently rejected a value or the DSN was malformed. The caller
// should close the *sql.DB on a non-nil error.
func VerifyOnOpen(ctx context.Context, db *sql.DB, p Policy) error {
	if db == nil {
		return errors.New("sqlitepragma: nil db")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("sqlitepragma: acquire connection: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := verifyJournalMode(ctx, conn, p.JournalMode); err != nil {
		return err
	}
	if err := verifyBusyTimeout(ctx, conn, p.BusyTimeout); err != nil {
		return err
	}
	if err := verifySynchronous(ctx, conn, p.Synchronous); err != nil {
		return err
	}
	return verifyForeignKeys(ctx, conn, p.ForeignKeys)
}

func verifyJournalMode(ctx context.Context, conn *sql.Conn, want string) error {
	var got string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&got); err != nil {
		return fmt.Errorf("sqlitepragma: read journal_mode: %w", err)
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("sqlitepragma: journal_mode mismatch: got %q want %q", got, want)
	}
	return nil
}

func verifyBusyTimeout(ctx context.Context, conn *sql.Conn, want time.Duration) error {
	var got int64
	if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&got); err != nil {
		return fmt.Errorf("sqlitepragma: read busy_timeout: %w", err)
	}
	if w := want.Milliseconds(); got != w {
		return fmt.Errorf("sqlitepragma: busy_timeout mismatch: got %dms want %dms", got, w)
	}
	return nil
}

func verifySynchronous(ctx context.Context, conn *sql.Conn, want string) error {
	var got string
	if err := conn.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&got); err != nil {
		return fmt.Errorf("sqlitepragma: read synchronous: %w", err)
	}
	if !synchronousEquals(got, want) {
		return fmt.Errorf("sqlitepragma: synchronous mismatch: got %q want %q", got, want)
	}
	return nil
}

func verifyForeignKeys(ctx context.Context, conn *sql.Conn, want bool) error {
	var got int
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&got); err != nil {
		return fmt.Errorf("sqlitepragma: read foreign_keys: %w", err)
	}
	w := 0
	if want {
		w = 1
	}
	if got != w {
		return fmt.Errorf("sqlitepragma: foreign_keys mismatch: got %d want %d", got, w)
	}
	return nil
}

// synchronousEquals compares a PRAGMA synchronous value returned as either a
// symbolic name ("NORMAL") or numeric string ("1") against the configured
// symbolic name, case-insensitively.
func synchronousEquals(got, want string) bool {
	if strings.EqualFold(got, want) {
		return true
	}
	names := map[string]string{
		"0": "OFF",
		"1": "NORMAL",
		"2": "FULL",
		"3": "EXTRA",
	}
	if sym, ok := names[strings.TrimSpace(got)]; ok {
		return strings.EqualFold(sym, want)
	}
	return false
}
