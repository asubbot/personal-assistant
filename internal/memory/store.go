package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store is the assistant's single long-term memory store: markdown files under
// a calendar directory structure year/month/day. Not subdivided by interlocutor.
type Store struct {
	rootDir string
	loc     *time.Location // calendar interpretation for day paths; nil means UTC
}

// NewStore creates a memory store rooted at rootDir (e.g. cfg.Paths.MemoryDir).
// rootDir must be non-empty. Directories are created on first write (e.g. WriteDaySummary).
// loc is the assistant calendar timezone (pa_timezone); nil defaults to UTC.
func NewStore(rootDir string, loc *time.Location) (*Store, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("memory: rootDir is required")
	}
	return &Store{rootDir: filepath.Clean(rootDir), loc: loc}, nil
}

// RootDir returns the cleaned memory root directory.
func (s *Store) RootDir() string {
	if s == nil {
		return ""
	}
	return s.rootDir
}

// Location returns the calendar timezone used for day paths, or UTC if unset.
func (s *Store) Location() *time.Location {
	if s == nil || s.loc == nil {
		return time.UTC
	}
	return s.loc
}

func (s *Store) calendarOf(day time.Time) (y int, m time.Month, d int) {
	loc := time.UTC
	if s != nil && s.loc != nil {
		loc = s.loc
	}
	return day.In(loc).Date()
}

// pathForDaySummary returns the path for the day summary: rootDir/YYYY/MM/DD/summary.md.
func (s *Store) pathForDaySummary(day time.Time) string {
	y, m, d := s.calendarOf(day)
	return filepath.Join(s.rootDir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d", d), "summary.md")
}

// WriteDaySummary writes the day summary markdown for the given calendar day in pa_timezone.
// Path: rootDir/YYYY/MM/DD/summary.md. Creates parent directories as needed. Overwrites existing file.
func (s *Store) WriteDaySummary(ctx context.Context, day time.Time, content string) error {
	path := s.pathForDaySummary(day)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("memory: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("memory: write %s: %w", path, err)
	}
	return nil
}

// ReadDaySummary reads the day summary for the given calendar day in pa_timezone.
// Returns empty string and nil error if the file does not exist.
func (s *Store) ReadDaySummary(ctx context.Context, day time.Time) (string, error) {
	path := s.pathForDaySummary(day)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("memory: read %s: %w", path, err)
	}
	return string(data), nil
}

// pathForMonthSummary returns the path for the month summary: rootDir/YYYY/MM/summary.md.
func (s *Store) pathForMonthSummary(year int, month int) string {
	return filepath.Join(s.rootDir, fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", month), "summary.md")
}

// pathForYearSummary returns the path for the year summary: rootDir/YYYY/summary.md.
func (s *Store) pathForYearSummary(year int) string {
	return filepath.Join(s.rootDir, fmt.Sprintf("%04d", year), "summary.md")
}

// WriteMonthSummary writes the month summary to rootDir/YYYY/MM/summary.md. Creates parent directories as needed.
func (s *Store) WriteMonthSummary(ctx context.Context, year int, month int, content string) error {
	path := s.pathForMonthSummary(year, month)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("memory: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("memory: write %s: %w", path, err)
	}
	return nil
}

// ReadMonthSummary reads the month summary. Returns empty string and nil error if the file does not exist.
func (s *Store) ReadMonthSummary(ctx context.Context, year int, month int) (string, error) {
	path := s.pathForMonthSummary(year, month)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("memory: read %s: %w", path, err)
	}
	return string(data), nil
}

// WriteYearSummary writes the year summary to rootDir/YYYY/summary.md. Creates parent directories as needed.
func (s *Store) WriteYearSummary(ctx context.Context, year int, content string) error {
	path := s.pathForYearSummary(year)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("memory: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("memory: write %s: %w", path, err)
	}
	return nil
}

// ReadYearSummary reads the year summary. Returns empty string and nil error if the file does not exist.
func (s *Store) ReadYearSummary(ctx context.Context, year int) (string, error) {
	path := s.pathForYearSummary(year)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("memory: read %s: %w", path, err)
	}
	return string(data), nil
}
