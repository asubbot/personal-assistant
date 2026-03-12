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
}

// NewStore creates a memory store rooted at rootDir (e.g. cfg.Paths.MemoryDir).
// rootDir must be non-empty. Directories are created on first WriteDay.
func NewStore(rootDir string) (*Store, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("memory: rootDir is required")
	}
	return &Store{rootDir: filepath.Clean(rootDir)}, nil
}

// pathForDay returns the file path for the given calendar day: rootDir/YYYY/MM/DD/full.md.
func (s *Store) pathForDay(t time.Time) string {
	y, m, d := t.UTC().Date()
	return filepath.Join(s.rootDir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d", d), "full.md")
}

// pathForDaySummary returns the path for the day summary: rootDir/YYYY/MM/DD/summary.md.
func (s *Store) pathForDaySummary(day time.Time) string {
	y, m, d := day.UTC().Date()
	return filepath.Join(s.rootDir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d", d), "summary.md")
}

// WriteDay writes content as markdown for the given calendar day (UTC).
// Creates parent directories (year/month) as needed. Overwrites existing file.
func (s *Store) WriteDay(ctx context.Context, day time.Time, content string) error {
	path := s.pathForDay(day)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("memory: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("memory: write %s: %w", path, err)
	}
	return nil
}

// ReadDay reads the markdown content for the given calendar day (UTC).
// Returns empty string and nil error if the file does not exist.
func (s *Store) ReadDay(ctx context.Context, day time.Time) (string, error) {
	path := s.pathForDay(day)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("memory: read %s: %w", path, err)
	}
	return string(data), nil
}

// WriteDaySummary writes the day summary markdown for the given calendar day (UTC).
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

// ReadDaySummary reads the day summary for the given calendar day (UTC).
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
