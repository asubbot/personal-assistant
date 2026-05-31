package config

import (
	"fmt"
	"pa/internal/sqlitepragma"
	"strings"
	"time"
)

func validateSQLiteStoreDefaults(c *SQLiteStoreDefaultsConfig) error {
	if c == nil {
		return fmt.Errorf("sqlite_store_defaults: block is required")
	}
	if strings.TrimSpace(c.JournalMode) == "" {
		return fmt.Errorf("sqlite_store_defaults.journal_mode: required")
	}
	if strings.TrimSpace(c.BusyTimeout) == "" {
		return fmt.Errorf("sqlite_store_defaults.busy_timeout: required")
	}
	if _, err := parsePositiveDuration("sqlite_store_defaults.busy_timeout", c.BusyTimeout); err != nil {
		return err
	}
	if strings.TrimSpace(c.Synchronous) == "" {
		return fmt.Errorf("sqlite_store_defaults.synchronous: required")
	}
	return nil
}

func mergeSQLiteStoreReliability(defaults *SQLiteStoreDefaultsConfig, override *SQLiteStoreReliabilityOverride) (*SQLiteStoreReliabilityConfig, error) {
	if defaults == nil {
		return nil, fmt.Errorf("sqlite_store_defaults is required")
	}
	if override == nil {
		return nil, fmt.Errorf("override block is required")
	}
	if override.ForeignKeys == nil {
		return nil, fmt.Errorf("foreign_keys: required (true or false)")
	}
	merged := &SQLiteStoreReliabilityConfig{
		JournalMode: defaults.JournalMode,
		BusyTimeout: defaults.BusyTimeout,
		Synchronous: defaults.Synchronous,
		ForeignKeys: override.ForeignKeys,
	}
	if override.JournalMode != nil {
		merged.JournalMode = *override.JournalMode
	}
	if override.BusyTimeout != nil {
		merged.BusyTimeout = *override.BusyTimeout
	}
	if override.Synchronous != nil {
		merged.Synchronous = *override.Synchronous
	}
	return merged, nil
}

func parsePositiveDuration(field, raw string) (time.Duration, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("config: %s is required (Go duration, e.g. \"5s\")", field)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("config: %s invalid duration %q: %w", field, raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("config: %s must be > 0, got %s", field, d)
	}
	return d, nil
}

// VectorStoreReliabilityPolicy returns the effective vector-store PRAGMA policy (EP-039).
func (c *Config) VectorStoreReliabilityPolicy() sqlitepragma.Policy {
	if c == nil {
		panic("config: Config is nil")
	}
	merged, err := mergeSQLiteStoreReliability(c.SQLiteStoreDefaults, c.VectorStoreReliability)
	if err != nil {
		panic(fmt.Sprintf("config: vector store reliability merge failed (Load should have rejected this): %v", err))
	}
	return merged.ToPolicy()
}

// JobsStoreReliabilityPolicy returns the effective jobs-store PRAGMA policy (EP-039).
func (c *Config) JobsStoreReliabilityPolicy() sqlitepragma.Policy {
	if c == nil {
		panic("config: Config is nil")
	}
	merged, err := mergeSQLiteStoreReliability(c.SQLiteStoreDefaults, c.JobsStoreReliability)
	if err != nil {
		panic(fmt.Sprintf("config: jobs store reliability merge failed (Load should have rejected this): %v", err))
	}
	return merged.ToPolicy()
}
