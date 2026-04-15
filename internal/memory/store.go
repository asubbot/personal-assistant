package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// pathForDayNotes returns rootDir/YYYY/MM/DD/notes.md for the calendar day in pa_timezone (EP-016).
func (s *Store) pathForDayNotes(day time.Time) string {
	y, m, d := s.calendarOf(day)
	return filepath.Join(s.rootDir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d", d), "notes.md")
}

// NotesPathForDay returns the notes.md path for the day (for tools path-prefix checks).
func (s *Store) NotesPathForDay(day time.Time) string {
	if s == nil {
		return ""
	}
	return s.pathForDayNotes(day)
}

// ReadDayNotes reads notes.md for the calendar day. Returns empty string and nil error if missing.
func (s *Store) ReadDayNotes(ctx context.Context, day time.Time) (string, error) {
	if s == nil {
		return "", fmt.Errorf("memory: nil store")
	}
	path := s.pathForDayNotes(day)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("memory: read %s: %w", path, err)
	}
	return string(data), nil
}

func normalizeNoteKind(kind string) (string, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "" {
		return "", nil
	}
	switch kind {
	case "fact", "guideline", "preference", "other":
		return kind, nil
	default:
		return "", fmt.Errorf("memory: invalid kind %q (use fact, guideline, preference, or other)", kind)
	}
}

func buildDayNoteEntry(text, kind string, nowUTC time.Time, maxAppend int) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("memory: empty note text")
	}
	var b strings.Builder
	b.WriteString(nowUTC.UTC().Format(time.RFC3339))
	b.WriteByte('\n')
	if kind != "" {
		b.WriteString("kind=")
		b.WriteString(kind)
		b.WriteByte('\n')
	}
	b.WriteString(strings.TrimRight(text, "\n"))
	b.WriteByte('\n')
	entry := b.String()
	if len(entry) > maxAppend {
		return "", fmt.Errorf("memory: note exceeds max_append_bytes (%d)", maxAppend)
	}
	return entry, nil
}

// AppendDayNote appends one entry to notes.md: first line UTC RFC3339, optional kind= line, then text (EP-016).
// Size is checked after ReadDayNotes then before append; concurrent writers could race (acceptable for a single bot process).
func (s *Store) AppendDayNote(ctx context.Context, day time.Time, text, kind string, nowUTC time.Time, maxAppend, maxFile int) error {
	if s == nil {
		return fmt.Errorf("memory: nil store")
	}
	if maxAppend < 1 || maxFile < 1 {
		return fmt.Errorf("memory: invalid note size limits")
	}
	kind, err := normalizeNoteKind(kind)
	if err != nil {
		return err
	}
	entry, err := buildDayNoteEntry(text, kind, nowUTC, maxAppend)
	if err != nil {
		return err
	}
	existing, err := s.ReadDayNotes(ctx, day)
	if err != nil {
		return err
	}
	sep := ""
	if strings.TrimSpace(existing) != "" {
		sep = "\n\n"
	}
	if len(existing)+len(sep)+len(entry) > maxFile {
		return fmt.Errorf("memory: notes.md would exceed max_file_bytes (%d)", maxFile)
	}
	path := s.pathForDayNotes(day)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("memory: mkdir %s: %w", dir, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("memory: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(sep + entry); err != nil {
		return fmt.Errorf("memory: append %s: %w", path, err)
	}
	return nil
}
