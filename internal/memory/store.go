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

// pathForDay returns the file path for the given calendar day: rootDir/YYYY/MM/DD.md.
func (s *Store) pathForDay(t time.Time) string {
	y, m, d := t.UTC().Date()
	return filepath.Join(s.rootDir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d.md", d))
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
