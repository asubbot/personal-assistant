package llmlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"pa/internal/llm"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type errNotDir struct{ path string }

func (e *errNotDir) Error() string { return fmt.Sprintf("llm log path is not a directory: %s", e.path) }

// Entry is one JSONL record: request metadata and response for a single LLM call.
type Entry struct {
	RequestID       string        `json:"request_id"`
	Messages        []llm.Message `json:"messages"`
	Model           string        `json:"model,omitempty"`
	ResponseContent string        `json:"response_content"`
	Usage           llm.Usage     `json:"usage"`
	DurationMs      int64         `json:"duration_ms"`
}

// Writer appends LLM log entries as JSON Lines. Safe for concurrent use.
// Log never returns an error; on write failure the error is logged and the entry is skipped (see package doc).
type Writer interface {
	Log(entry *Entry)
}

// Redactor is a function that redacts secret values from a string before writing to the log (REQ-01.026).
type Redactor func(string) string

// fileWriter appends entries to a daily file in the given directory.
type fileWriter struct {
	dir      string
	date     string
	f        *os.File
	mu       sync.Mutex
	log      *slog.Logger
	redactor Redactor // applied to message contents and response_content before marshal
}

// NewWriter creates a Writer that writes one file per day under dir (llm-YYYY-MM-DD.jsonl).
// If dir is empty, NewWriter returns (nil, nil); callers should skip logging.
// redactor is applied to all string fields that may contain secrets (message contents, response_content) before writing; may be nil to skip redaction.
// Logger may be nil; when non-nil, write errors are logged to it. See package doc for startup and write-time behaviour.
func NewWriter(dir string, logger *slog.Logger, redactor Redactor) (Writer, error) {
	if dir == "" {
		return nil, nil
	}
	dir = filepath.Clean(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &errNotDir{path: dir}
	}
	if err := checkWritable(dir); err != nil {
		return nil, err
	}
	w := &fileWriter{dir: dir, log: logger, redactor: redactor}
	return w, nil
}

func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".llmlog-probe-")
	if err != nil {
		return err
	}
	_ = f.Close()
	return os.Remove(f.Name())
}

func (w *fileWriter) Log(entry *Entry) {
	w.mu.Lock()
	defer w.mu.Unlock()
	today := time.Now().UTC().Format("2006-01-02")
	if w.f == nil || w.date != today {
		if w.f != nil {
			_ = w.f.Close()
			w.f = nil
		}
		name := "llm-" + today + ".jsonl"
		f, err := os.OpenFile(filepath.Join(w.dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			w.logWriteError("open log file", err)
			return
		}
		w.f = f
		w.date = today
	}
	toWrite := w.redactEntry(entry)
	line, err := json.Marshal(toWrite)
	if err != nil {
		w.logWriteError("marshal log entry", err)
		return
	}
	line = append(line, '\n')
	if _, err := w.f.Write(line); err != nil {
		w.logWriteError("write log entry", err)
		return
	}
}

// redactEntry returns a copy of the entry with string fields redacted so logs never contain secrets (REQ-01.026).
func (w *fileWriter) redactEntry(entry *Entry) *Entry {
	if w.redactor == nil {
		return entry
	}
	out := &Entry{
		RequestID:       entry.RequestID,
		Model:           entry.Model,
		ResponseContent: w.redactor(entry.ResponseContent),
		Usage:           entry.Usage,
		DurationMs:      entry.DurationMs,
	}
	out.Messages = make([]llm.Message, len(entry.Messages))
	for i, m := range entry.Messages {
		out.Messages[i] = llm.Message{Role: m.Role, Content: w.redactor(m.Content)}
	}
	return out
}

func (w *fileWriter) logWriteError(op string, err error) {
	if w.log != nil {
		w.log.Warn("llm log write failed", "op", op, "error", err)
	}
}

// ReadEntriesForDay reads all LLM log entries for the given calendar day (UTC).
// File name is llm-YYYY-MM-DD.jsonl under dir. Returns empty slice and nil error if the file does not exist.
func ReadEntriesForDay(dir string, day time.Time) ([]Entry, error) {
	dir = filepath.Clean(dir)
	dateStr := day.UTC().Format("2006-01-02")
	name := "llm-" + dateStr + ".jsonl"
	path := filepath.Join(dir, name)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("llmlog: read %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("llmlog: parse %s: %w", path, err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("llmlog: read %s: %w", path, err)
	}
	return entries, nil
}

const (
	llmLogFilenamePrefix = "llm-"
	llmLogFilenameSuffix = ".jsonl"
	llmLogFilenameLen    = 20 // "llm-YYYY-MM-DD.jsonl"
)

// parseLLMLogFilename returns the date from a filename "llm-YYYY-MM-DD.jsonl", or (zero, false) if not matching.
func parseLLMLogFilename(name string) (time.Time, bool) {
	if len(name) != llmLogFilenameLen || !strings.HasPrefix(name, llmLogFilenamePrefix) || !strings.HasSuffix(name, llmLogFilenameSuffix) {
		return time.Time{}, false
	}
	date, err := time.Parse("2006-01-02", name[4:14])
	if err != nil {
		return time.Time{}, false
	}
	return date, true
}

// PruneRetention deletes llm-YYYY-MM-DD.jsonl files in dir older than retentionDays (UTC).
// If retentionDays <= 0, no files are removed and nil is returned.
// Only regular files with names exactly "llm-YYYY-MM-DD.jsonl" are considered; parsing errors are skipped.
// Returns the first error from ReadDir or Remove.
func PruneRetention(dir string, retentionDays int, logger *slog.Logger) error {
	return pruneRetentionWithNow(dir, retentionDays, logger, time.Time{})
}

// PruneRetentionWithNow is for tests: nowUTC is the "current" time in UTC; if zero, time.Now().UTC() is used.
func PruneRetentionWithNow(dir string, retentionDays int, logger *slog.Logger, nowUTC time.Time) error {
	return pruneRetentionWithNow(dir, retentionDays, logger, nowUTC)
}

func pruneRetentionWithNow(dir string, retentionDays int, logger *slog.Logger, nowUTC time.Time) error {
	if retentionDays <= 0 {
		return nil
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	cutoff := nowUTC.Truncate(24*time.Hour).AddDate(0, 0, -retentionDays)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		fileDate, ok := parseLLMLogFilename(name)
		if !ok || !fileDate.Before(cutoff) {
			continue
		}
		path := filepath.Join(dir, name)
		if err := os.Remove(path); err != nil {
			return err
		}
		if logger != nil {
			logger.Info("llm log pruned", "file", name)
		}
	}
	return nil
}
