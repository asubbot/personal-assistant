package llmlog

import (
	"encoding/json"
	"fmt"
	"os"
	"pa/internal/llm"
	"pa/internal/logredact"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Covers AC-017, AC-018 (US-09, US-10): Log writes parseable JSONL with required fields (request_id, messages, response_content, usage, duration_ms).
func TestLog_writesParseableJSONLWithRequiredFields(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir, nil, nil)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if w == nil {
		t.Fatal("NewWriter returned nil writer")
	}

	requestID := "req-123"
	messages := []llm.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}
	model := "gpt-4"
	responseContent := "Response text"
	usage := llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	durationMs := int64(250)

	entry := &Entry{
		RequestID:       requestID,
		Messages:        messages,
		Model:           model,
		ResponseContent: responseContent,
		Usage:           usage,
		DurationMs:      durationMs,
	}
	w.Log(entry)

	today := time.Now().UTC().Format("2006-01-02")
	logPath := filepath.Join(dir, "llm-"+today+".jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", logPath, err)
	}

	parsed, lines := parseJSONLLines(t, data)
	if lines != 1 {
		t.Errorf("expected exactly one log line, got %d", lines)
	}
	assertLogEntryFields(t, parsed, requestID, responseContent, durationMs)
}

// parseJSONLLines parses non-empty lines as JSON; returns the last parsed object and line count.
func parseJSONLLines(t *testing.T, data []byte) (map[string]interface{}, int) {
	t.Helper()
	var parsed map[string]interface{}
	lines := 0
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		lines++
		parsed = make(map[string]interface{})
		if err := json.Unmarshal(line, &parsed); err != nil {
			t.Fatalf("line %d: invalid JSON: %v\nraw: %s", lines, err, line)
		}
	}
	return parsed, lines
}

// assertLogEntryFields checks AC-017 required fields in a parsed log entry.
func assertLogEntryFields(t *testing.T, parsed map[string]interface{}, requestID, responseContent string, durationMs int64) {
	t.Helper()
	if got, ok := parsed["request_id"].(string); !ok || got != requestID {
		t.Errorf("request_id = %v (type %T), want %q", parsed["request_id"], parsed["request_id"], requestID)
	}
	if _, ok := parsed["messages"]; !ok {
		t.Error("missing messages field")
	}
	if got, ok := parsed["response_content"].(string); !ok || got != responseContent {
		t.Errorf("response_content = %v, want %q", got, responseContent)
	}
	if _, ok := parsed["usage"]; !ok {
		t.Error("missing usage field")
	}
	switch v := parsed["duration_ms"].(type) {
	case float64:
		if int64(v) != durationMs {
			t.Errorf("duration_ms = %v, want %d", v, durationMs)
		}
	case nil:
		t.Error("missing duration_ms field")
	default:
		t.Errorf("duration_ms = %v (type %T), want %d", v, v, durationMs)
	}
}

// splitLines returns non-empty lines from data (split by newline).
func splitLines(data []byte) [][]byte {
	var out [][]byte
	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			if i > start {
				out = append(out, data[start:i])
			}
			start = i + 1
		}
	}
	return out
}

// Covers AC-019 (US-10): NewWriter rejects path that exists and is a file (not a directory).
func TestNewWriter_rejectsPathThatIsFile(t *testing.T) {
	f, err := os.CreateTemp("", "llmlog-test-*.tmp")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	defer func() { _ = os.Remove(path) }()

	_, err = NewWriter(path, nil, nil)
	if err == nil {
		t.Fatal("NewWriter with file path: expected error, got nil")
	}
	// When path is a file, MkdirAll may fail with PathError before Stat; either way we require an error.
}

// Covers AC-019 (US-10): NewWriter rejects read-only directory (not writable).
func TestNewWriter_rejectsReadOnlyDirectory(t *testing.T) {
	dir := t.TempDir()
	// Remove write permission for the directory so checkWritable (CreateTemp inside dir) fails.
	if err := os.Chmod(dir, 0o444); err != nil {
		t.Skipf("Chmod read-only not supported: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }() // restore so TempDir cleanup can remove it

	_, err := NewWriter(dir, nil, nil)
	if err == nil {
		t.Fatal("NewWriter with read-only dir: expected error, got nil")
	}
}

// Supporting AC-030 (US-16): when a redactor is supplied, secret content is redacted in the written log file.
func TestLog_redactsSecretInWrittenFile(t *testing.T) {
	dir := t.TempDir()
	redactor := logredact.NewRedactor(nil)
	w, err := NewWriter(dir, nil, Redactor(redactor))
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Built-in pattern sk-[a-zA-Z0-9]{20,} → [REDACTED]
	secret := "sk-abc123def456ghi789jkl012"
	entry := &Entry{
		RequestID:       "req-redact",
		Messages:        []llm.Message{{Role: "user", Content: "key is " + secret}},
		ResponseContent: "reply with " + secret,
		Usage:           llm.Usage{},
		DurationMs:      0,
	}
	w.Log(entry)

	today := time.Now().UTC().Format("2006-01-02")
	logPath := filepath.Join(dir, "llm-"+today+".jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if strings.Contains(content, secret) {
		t.Errorf("log file must not contain raw secret %q\ncontent: %s", secret, content)
	}
	if !strings.Contains(content, "[REDACTED]") {
		t.Errorf("log file must contain redaction replacement [REDACTED]\ncontent: %s", content)
	}
}

// Supporting AC-011 (US-06): ReadEntriesForDay returns nil slice when log file is missing (summarize input).
func TestReadEntriesForDay_missingFile_returnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	entries, err := ReadEntriesForDay(dir, day)
	if err != nil {
		t.Fatalf("ReadEntriesForDay: %v", err)
	}
	if entries != nil {
		t.Errorf("ReadEntriesForDay(missing file): got %d entries, want nil", len(entries))
	}
}

// Supporting AC-011 (US-06): ReadEntriesForDay parses one JSONL line as one Entry (summarize input).
func TestReadEntriesForDay_oneEntry_parsed(t *testing.T) {
	dir := t.TempDir()
	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	path := filepath.Join(dir, "llm-2026-03-12.jsonl")
	entry := &Entry{
		RequestID:       "r1",
		Messages:        []llm.Message{{Role: "user", Content: "hi"}},
		ResponseContent: "hello",
		Usage:           llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
		DurationMs:      100,
	}
	line, _ := json.Marshal(entry)
	if err := os.WriteFile(path, append(line, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entries, err := ReadEntriesForDay(dir, day)
	if err != nil {
		t.Fatalf("ReadEntriesForDay: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadEntriesForDay: got %d entries, want 1", len(entries))
	}
	if entries[0].RequestID != "r1" || entries[0].ResponseContent != "hello" {
		t.Errorf("entry: request_id=%q response_content=%q", entries[0].RequestID, entries[0].ResponseContent)
	}
}

// Supporting AC-011 (US-06): ReadEntriesForDay parses multiple JSONL lines (summarize input).
func TestReadEntriesForDay_twoEntries_parsed(t *testing.T) {
	dir := t.TempDir()
	day := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	path := filepath.Join(dir, "llm-2026-03-15.jsonl")
	var lines []byte
	for i, content := range []string{"first", "second"} {
		e := &Entry{
			RequestID:       fmt.Sprintf("req-%d", i),
			Messages:        []llm.Message{{Role: "user", Content: content}},
			ResponseContent: "reply-" + content,
			Usage:           llm.Usage{},
			DurationMs:      int64(i),
		}
		line, _ := json.Marshal(e)
		lines = append(lines, line...)
		lines = append(lines, '\n')
	}
	if err := os.WriteFile(path, lines, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entries, err := ReadEntriesForDay(dir, day)
	if err != nil {
		t.Fatalf("ReadEntriesForDay: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ReadEntriesForDay: got %d entries, want 2", len(entries))
	}
	if entries[0].ResponseContent != "reply-first" || entries[1].ResponseContent != "reply-second" {
		t.Errorf("entries: got %q, %q", entries[0].ResponseContent, entries[1].ResponseContent)
	}
}

// PruneRetention: retentionDays <= 0 does not delete any files.
func TestPruneRetention_zeroOrNegative_doesNotDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "llm-2020-01-01.jsonl")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	if err := PruneRetentionWithNow(dir, 0, nil, now); err != nil {
		t.Fatalf("PruneRetention(0): %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file was removed: %v", err)
	}
	if err := PruneRetentionWithNow(dir, -1, nil, now); err != nil {
		t.Fatalf("PruneRetention(-1): %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file was removed: %v", err)
	}
}

// PruneRetention: with retention 7 and "today" 2026-03-18, files older than 7 days are deleted, recent kept.
func TestPruneRetention_removesOldKeepsRecent(t *testing.T) {
	dir := t.TempDir()
	old1 := filepath.Join(dir, "llm-2020-01-01.jsonl")
	old2 := filepath.Join(dir, "llm-2026-03-10.jsonl")
	recent := filepath.Join(dir, "llm-2026-03-12.jsonl")
	for _, p := range []string{old1, old2, recent} {
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatalf("setup %s: %v", p, err)
		}
	}
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	if err := PruneRetentionWithNow(dir, 7, nil, now); err != nil {
		t.Fatalf("PruneRetention: %v", err)
	}
	for _, p := range []string{old1, old2} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("expected %s to be removed", p)
		}
	}
	if _, err := os.Stat(recent); err != nil {
		t.Errorf("expected %s to remain: %v", recent, err)
	}
}

// PruneRetention: files not matching llm-YYYY-MM-DD.jsonl are not deleted.
func TestPruneRetention_ignoresNonMatchingNames(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "llm-2026-03-12.jsonl.bak")
	if err := os.WriteFile(keep, []byte("{}"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	if err := PruneRetentionWithNow(dir, 7, nil, now); err != nil {
		t.Fatalf("PruneRetention: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("non-matching file was removed: %v", err)
	}
}

// PruneRetention: returns error when directory does not exist (ReadDir fails).
func TestPruneRetention_nonexistentDir_returnsError(t *testing.T) {
	dir := t.TempDir()
	nonexistent := filepath.Join(dir, "nonexistent_subdir")
	err := PruneRetention(nonexistent, 7, nil)
	if err == nil {
		t.Fatal("PruneRetention(nonexistent dir): expected error, got nil")
	}
}
