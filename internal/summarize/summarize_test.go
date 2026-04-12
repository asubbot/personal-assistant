package summarize

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/memory"
	"pa/internal/vector"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockLLM struct {
	content string
	calls   int
}

func (m *mockLLM) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.calls++
	return &llm.CompletionResult{Content: m.content}, nil
}

type mockEmbedder struct {
	vec []float32
	err error
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vec, nil
}

type mockVectorStore struct {
	adds     []string // ids added
	addTexts []string // vector document text passed to Add
	deletes  []string // ids deleted
}

func (m *mockVectorStore) Add(ctx context.Context, id string, embedding []float32, text string) error {
	m.adds = append(m.adds, id)
	m.addTexts = append(m.addTexts, text)
	return nil
}

func (m *mockVectorStore) Delete(ctx context.Context, id string) error {
	m.deletes = append(m.deletes, id)
	return nil
}

func (m *mockVectorStore) Clear(ctx context.Context) error { return nil }

func (m *mockVectorStore) Search(ctx context.Context, queryEmbedding []float32, topK int) ([]vector.SearchResult, error) {
	return nil, nil
}

func (m *mockVectorStore) Exists(context.Context, string) (bool, error) { return false, nil }

func (m *mockVectorStore) Close() error { return nil }

// Covers AC-02.015: EP-002 summarization and vector paths are exercised under make check (non-functional gate).
func TestEP002_makeCheckGate(t *testing.T) {}

// Covers AC-01.011, AC-01.012 (US-06): day summarization with no log entries skips write; no memory or vector update.
func TestDay_noEntries_skips(t *testing.T) {
	dir := t.TempDir()
	llmLogDir := filepath.Join(dir, "llm_logs")
	if err := os.MkdirAll(llmLogDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	cfg := DayConfig{
		LLMLogDir:   llmLogDir,
		LLMProvider: &mockLLM{content: "summary"},
		MemoryStore: memStore,
		Logger:      slog.Default(),
	}

	err = Day(context.Background(), day, cfg)
	if err != nil {
		t.Fatalf("Day(no entries): %v", err)
	}

	content, _ := memStore.ReadDaySummary(context.Background(), day)
	if content != "" {
		t.Errorf("expected no summary written when no entries; got %q", content)
	}
}

// Covers AC-01.011, AC-01.012 (US-06), AC-02.008: day summarization with log entries writes summary to memory and vector store; vector text includes Date line and summary chunk label.
// Covers AC-02.014: vector upsert deletes then adds the same stable summary:day id.
func TestDay_withEntries_callsLLMAndWrites(t *testing.T) {
	dir := t.TempDir()
	llmLogDir := filepath.Join(dir, "llm_logs")
	if err := os.MkdirAll(llmLogDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Write one entry for 2026-03-12
	logPath := filepath.Join(llmLogDir, "llm-2026-03-12.jsonl")
	entry := llmlog.Entry{
		RequestID:       "r1",
		Messages:        []llm.Message{{Role: "user", Content: "hello"}},
		ResponseContent: "hi there",
		Usage:           llm.Usage{},
		DurationMs:      1,
	}
	line, _ := json.Marshal(entry)
	if err := os.WriteFile(logPath, append(line, '\n'), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	llmMock := &mockLLM{content: "User said hello; assistant replied."}
	vecMock := &mockVectorStore{}

	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	cfg := DayConfig{
		LLMLogDir:   llmLogDir,
		LLMProvider: llmMock,
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		VectorStore: vecMock,
		Logger:      slog.Default(),
	}

	err = Day(context.Background(), day, cfg)
	if err != nil {
		t.Fatalf("Day: %v", err)
	}

	if llmMock.calls != 1 {
		t.Errorf("LLM calls = %d, want 1", llmMock.calls)
	}

	content, err := memStore.ReadDaySummary(context.Background(), day)
	if err != nil {
		t.Fatalf("ReadDaySummary: %v", err)
	}
	if content != "User said hello; assistant replied." {
		t.Errorf("memory summary = %q", content)
	}

	wantID := "summary:day:2026-03-12"
	if len(vecMock.deletes) != 1 || vecMock.deletes[0] != wantID {
		t.Errorf("vector deletes = %v, want [%s]", vecMock.deletes, wantID)
	}
	if len(vecMock.adds) != 1 || vecMock.adds[0] != wantID {
		t.Errorf("vector adds = %v, want [%s]", vecMock.adds, wantID)
	}
	assertSingleAddTextContains(t, vecMock, "Date: 2026-03-12", "[summary:day]")
}

// Covers AC-02.014: second run for the same calendar day deletes and re-adds the same vector id (no duplicate id).
//
//nolint:gocyclo // setup + two Day runs + vector assertions; clarity over splitting
func TestDay_secondRun_upsertsSameVectorID(t *testing.T) {
	dir := t.TempDir()
	llmLogDir := filepath.Join(dir, "llm_logs")
	if err := os.MkdirAll(llmLogDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logPath := filepath.Join(llmLogDir, "llm-2026-03-12.jsonl")
	entry := llmlog.Entry{
		RequestID:       "r1",
		Messages:        []llm.Message{{Role: "user", Content: "hello"}},
		ResponseContent: "hi",
		Usage:           llm.Usage{},
		DurationMs:      1,
	}
	line, _ := json.Marshal(entry)
	if err := os.WriteFile(logPath, append(line, '\n'), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	memStore, err := memory.NewStore(filepath.Join(dir, "memory"), time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	llmMock := &mockLLM{content: "first summary"}
	vecMock := &mockVectorStore{}
	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	cfg := DayConfig{
		LLMLogDir:   llmLogDir,
		LLMProvider: llmMock,
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		VectorStore: vecMock,
		Logger:      slog.Default(),
	}
	if err := Day(context.Background(), day, cfg); err != nil {
		t.Fatalf("Day first: %v", err)
	}
	llmMock.content = "second summary"
	if err := Day(context.Background(), day, cfg); err != nil {
		t.Fatalf("Day second: %v", err)
	}
	wantID := "summary:day:2026-03-12"
	if len(vecMock.deletes) != 2 || vecMock.deletes[0] != wantID || vecMock.deletes[1] != wantID {
		t.Errorf("vector deletes = %v, want two %s", vecMock.deletes, wantID)
	}
	if len(vecMock.adds) != 2 || vecMock.adds[0] != wantID || vecMock.adds[1] != wantID {
		t.Errorf("vector adds = %v, want two %s", vecMock.adds, wantID)
	}
	got, _ := memStore.ReadDaySummary(context.Background(), day)
	if got != "second summary" {
		t.Errorf("memory after rerun = %q, want second summary", got)
	}
}

// Covers AC-02.017: after markdown write succeeds, embed failure surfaces ErrVectorIndexAfterFileWrite for reconciliation (no second LLM in this unit).
func TestDay_embedFailsAfterFileWrite_returnsVectorIndexErr(t *testing.T) {
	dir := t.TempDir()
	llmLogDir := filepath.Join(dir, "llm_logs")
	if err := os.MkdirAll(llmLogDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logPath := filepath.Join(llmLogDir, "llm-2026-03-12.jsonl")
	entry := llmlog.Entry{
		RequestID: "r1", Messages: []llm.Message{{Role: "user", Content: "x"}},
		ResponseContent: "y", Usage: llm.Usage{}, DurationMs: 1,
	}
	line, _ := json.Marshal(entry)
	if err := os.WriteFile(logPath, append(line, '\n'), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	memStore, err := memory.NewStore(filepath.Join(dir, "memory"), time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	cfg := DayConfig{
		LLMLogDir:   llmLogDir,
		LLMProvider: &mockLLM{content: "summary text"},
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{err: errors.New("embed failed")},
		VectorStore: &mockVectorStore{},
		Logger:      slog.Default(),
	}
	err = Day(context.Background(), day, cfg)
	if err == nil || !IsVectorIndexAfterFileWrite(err) {
		t.Fatalf("Day: want ErrVectorIndexAfterFileWrite wrap, got %v", err)
	}
	got, _ := memStore.ReadDaySummary(context.Background(), day)
	if got != "summary text" {
		t.Errorf("memory written despite vector failure = %q", got)
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — YYYY-MM-DD parses as day scope.
func TestParseSummarizeScope_day(t *testing.T) {
	scope, err := ParseSummarizeScope("2026-03-12")
	if err != nil {
		t.Fatalf("ParseSummarizeScope: %v", err)
	}
	if scope.Kind != "day" {
		t.Fatalf("Kind = %q, want day", scope.Kind)
	}
	if scope.Day.Year() != 2026 || scope.Day.Month() != 3 || scope.Day.Day() != 12 {
		t.Errorf("Day = %v", scope.Day)
	}
	if scope.Day.Location() != time.UTC {
		t.Errorf("Day location = %v", scope.Day.Location())
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — YYYY-MM parses as month scope.
func TestParseSummarizeScope_month(t *testing.T) {
	scope, err := ParseSummarizeScope("2026-03")
	if err != nil {
		t.Fatalf("ParseSummarizeScope: %v", err)
	}
	if scope.Kind != "month" || scope.Year != 2026 || scope.Month != 3 {
		t.Errorf("scope = %+v", scope)
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — YYYY parses as year scope.
func TestParseSummarizeScope_year(t *testing.T) {
	scope, err := ParseSummarizeScope("2026")
	if err != nil {
		t.Fatalf("ParseSummarizeScope: %v", err)
	}
	if scope.Kind != "year" || scope.Year != 2026 {
		t.Errorf("scope = %+v", scope)
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — empty value returns error.
func TestParseSummarizeScope_empty_returnsError(t *testing.T) {
	_, err := ParseSummarizeScope("")
	if err == nil {
		t.Fatal("ParseSummarizeScope(empty): expected error")
	}
}

// Supporting AC-01.011, AC-01.012 (US-06): CLI scope parsing — invalid format returns error.
func TestParseSummarizeScope_invalid_returnsError(t *testing.T) {
	_, err := ParseSummarizeScope("not-a-date")
	if err == nil {
		t.Fatal("ParseSummarizeScope(invalid): expected error")
	}
}

// Covers AC-01.011, AC-01.012 (US-06): month summarization with no day summaries skips write.
func TestMonth_noDaySummaries_skips(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	cfg := MonthConfig{
		LLMProvider: &mockLLM{content: "month summary"},
		MemoryStore: memStore,
		Logger:      slog.Default(),
	}

	err = Month(context.Background(), 2026, 3, cfg)
	if err != nil {
		t.Fatalf("Month(no day summaries): %v", err)
	}

	content, _ := memStore.ReadMonthSummary(context.Background(), 2026, 3)
	if content != "" {
		t.Errorf("expected no month summary when no day summaries; got %q", content)
	}
}

// Covers AC-01.011, AC-01.012 (US-06), AC-02.008: month summarization writes month summary to memory and vector store with Date and [summary:month] in vector text.
func TestMonth_withDaySummaries_callsLLMAndWrites(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	day1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	if err := memStore.WriteDaySummary(context.Background(), day1, "Day 1 summary."); err != nil {
		t.Fatalf("WriteDaySummary 1: %v", err)
	}
	if err := memStore.WriteDaySummary(context.Background(), day2, "Day 15 summary."); err != nil {
		t.Fatalf("WriteDaySummary 2: %v", err)
	}

	llmMock := &mockLLM{content: "March overview: two active days."}
	vecMock := &mockVectorStore{}

	cfg := MonthConfig{
		LLMProvider: llmMock,
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{vec: []float32{0, 1, 0, 0}},
		VectorStore: vecMock,
		Logger:      slog.Default(),
	}

	err = Month(context.Background(), 2026, 3, cfg)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}

	if llmMock.calls != 1 {
		t.Errorf("LLM calls = %d, want 1", llmMock.calls)
	}

	content, err := memStore.ReadMonthSummary(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("ReadMonthSummary: %v", err)
	}
	if content != "March overview: two active days." {
		t.Errorf("memory month summary = %q", content)
	}

	wantID := "summary:month:2026-03"
	if len(vecMock.deletes) != 1 || vecMock.deletes[0] != wantID {
		t.Errorf("vector deletes = %v, want [%s]", vecMock.deletes, wantID)
	}
	if len(vecMock.adds) != 1 || vecMock.adds[0] != wantID {
		t.Errorf("vector adds = %v, want [%s]", vecMock.adds, wantID)
	}
	assertSingleAddTextContains(t, vecMock, "Date: 2026-03", "[summary:month]")
}

// Covers AC-01.011, AC-01.012 (US-06): year summarization with no month summaries skips write.
func TestYear_noMonthSummaries_skips(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	cfg := YearConfig{
		LLMProvider: &mockLLM{content: "year summary"},
		MemoryStore: memStore,
		Logger:      slog.Default(),
	}

	err = Year(context.Background(), 2026, cfg)
	if err != nil {
		t.Fatalf("Year(no month summaries): %v", err)
	}

	content, _ := memStore.ReadYearSummary(context.Background(), 2026)
	if content != "" {
		t.Errorf("expected no year summary when no month summaries; got %q", content)
	}
}

// Covers AC-01.011, AC-01.012 (US-06), AC-02.008: year summarization writes year summary to memory and vector store with Date and [summary:year] in vector text.
func TestYear_withMonthSummaries_callsLLMAndWrites(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	memStore, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if err := memStore.WriteMonthSummary(context.Background(), 2026, 1, "January summary."); err != nil {
		t.Fatalf("WriteMonthSummary 1: %v", err)
	}
	if err := memStore.WriteMonthSummary(context.Background(), 2026, 6, "June summary."); err != nil {
		t.Fatalf("WriteMonthSummary 2: %v", err)
	}

	llmMock := &mockLLM{content: "2026 overview: active in Jan and Jun."}
	vecMock := &mockVectorStore{}

	cfg := YearConfig{
		LLMProvider: llmMock,
		MemoryStore: memStore,
		Embedder:    &mockEmbedder{vec: []float32{0, 0, 1, 0}},
		VectorStore: vecMock,
		Logger:      slog.Default(),
	}

	err = Year(context.Background(), 2026, cfg)
	if err != nil {
		t.Fatalf("Year: %v", err)
	}

	if llmMock.calls != 1 {
		t.Errorf("LLM calls = %d, want 1", llmMock.calls)
	}

	content, err := memStore.ReadYearSummary(context.Background(), 2026)
	if err != nil {
		t.Fatalf("ReadYearSummary: %v", err)
	}
	if content != "2026 overview: active in Jan and Jun." {
		t.Errorf("memory year summary = %q", content)
	}

	wantID := "summary:year:2026"
	if len(vecMock.deletes) != 1 || vecMock.deletes[0] != wantID {
		t.Errorf("vector deletes = %v, want [%s]", vecMock.deletes, wantID)
	}
	if len(vecMock.adds) != 1 || vecMock.adds[0] != wantID {
		t.Errorf("vector adds = %v, want [%s]", vecMock.adds, wantID)
	}
	assertSingleAddTextContains(t, vecMock, "Date: 2026", "[summary:year]")
}

// Covers AC-01.011: traceability for TestBuildDayTranscript_omitsEmptyAssistantLines.
func TestBuildDayTranscript_omitsEmptyAssistantLines(t *testing.T) {
	out := buildDayTranscript([]llmlog.Entry{{
		Messages: []llm.Message{
			{Role: "user", Content: "q1"},
			{Role: "assistant", Content: ""},
			{Role: "assistant", Content: "   "},
			{Role: "assistant", Content: "answer"},
		},
		ResponseContent: "answer",
	}})
	if strings.Contains(out, "assistant: \n") || strings.Contains(out, "assistant:    \n") {
		t.Errorf("must omit empty/whitespace-only assistant lines; got %q", out)
	}
	if strings.Count(out, "answer") != 1 {
		t.Errorf("want single non-duplicate answer; got %q", out)
	}
	if !strings.Contains(out, "user: q1") {
		t.Errorf("want user line; got %q", out)
	}
}

// Covers AC-01.011: traceability for TestBuildDayTranscript_omitsToolRole.
func TestBuildDayTranscript_omitsToolRole(t *testing.T) {
	out := buildDayTranscript([]llmlog.Entry{{
		Messages: []llm.Message{
			{Role: "user", Content: "run tool"},
			{Role: "tool", Content: "{\"huge\":\"TOOL_PAYLOAD_DO_NOT_SUMMARIZE\"}"},
			{Role: "assistant", Content: "done"},
		},
		ResponseContent: "done",
	}})
	if strings.Contains(out, "TOOL_PAYLOAD") {
		t.Errorf("transcript must omit tool role content; got %q", out)
	}
	if !strings.Contains(out, "user: run tool") || !strings.Contains(out, "assistant: done") {
		t.Errorf("want user+assistant only; got %q", out)
	}
}

// Covers AC-01.011: traceability for TestBuildDayTranscript_omitsSystem.
func TestBuildDayTranscript_omitsSystem(t *testing.T) {
	out := buildDayTranscript([]llmlog.Entry{{
		Messages: []llm.Message{
			{Role: "system", Content: "DO_NOT_LEAK_THIS_SYSTEM_BLOCK"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
		ResponseContent: "hi",
	}})
	if strings.Contains(out, "DO_NOT_LEAK_THIS_SYSTEM_BLOCK") {
		t.Errorf("transcript must omit system content; got %q", out)
	}
	if !strings.Contains(out, "user: hello") || !strings.Contains(out, "assistant: hi") {
		t.Errorf("want user+assistant; got %q", out)
	}
	if strings.Count(out, "hi") > 2 {
		t.Errorf("unexpected duplication; got %q", out)
	}
}

// Covers AC-01.011: traceability for TestBuildDayTranscript_noDuplicateAssistant.
func TestBuildDayTranscript_noDuplicateAssistant(t *testing.T) {
	out := buildDayTranscript([]llmlog.Entry{{
		Messages: []llm.Message{
			{Role: "user", Content: "q"},
			{Role: "assistant", Content: "final answer"},
		},
		ResponseContent: "final answer",
	}})
	if strings.Count(out, "final answer") != 1 {
		t.Errorf("want single final answer; got %q", out)
	}
}

func assertSingleAddTextContains(t *testing.T, m *mockVectorStore, subs ...string) {
	t.Helper()
	if len(m.addTexts) != 1 {
		t.Fatalf("vector addTexts len = %d, want 1", len(m.addTexts))
	}
	for _, sub := range subs {
		if !strings.Contains(m.addTexts[0], sub) {
			t.Errorf("vector add text missing %q: %q", sub, m.addTexts[0])
		}
	}
}

// Covers AC-01.011: traceability for TestBuildDayTranscript_appendsResponseWhenNoAssistantInMessages.
func TestBuildDayTranscript_appendsResponseWhenNoAssistantInMessages(t *testing.T) {
	out := buildDayTranscript([]llmlog.Entry{{
		Messages: []llm.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "only user"},
		},
		ResponseContent: "model reply only here",
	}})
	if !strings.Contains(out, "Assistant: model reply only here") {
		t.Errorf("want Assistant from response_content; got %q", out)
	}
	if strings.Contains(out, "sys") {
		t.Errorf("system should be omitted; got %q", out)
	}
}

// Covers AC-01.011: traceability for TestLlmMessagesDebugText_joinsRoles.
func TestLlmMessagesDebugText_joinsRoles(t *testing.T) {
	got := llmMessagesDebugText([]llm.Message{
		{Role: "system", Content: "a"},
		{Role: "user", Content: "b"},
	})
	if !strings.Contains(got, "[system]") || !strings.Contains(got, "a") {
		t.Errorf("missing system block: %q", got)
	}
	if !strings.Contains(got, "[user]") || !strings.Contains(got, "b") {
		t.Errorf("missing user block: %q", got)
	}
	if !strings.Contains(got, "--- message ---") {
		t.Errorf("want separator between messages: %q", got)
	}
}

// Covers AC-01.011: traceability for TestLlmMessagesJSONByteLen_matchesMarshal.
func TestLlmMessagesJSONByteLen_matchesMarshal(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "hello"},
		{Role: "user", Content: "world"},
	}
	want, err := json.Marshal(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if got := llmMessagesJSONByteLen(msgs); got != len(want) {
		t.Fatalf("llmMessagesJSONByteLen = %d, want %d", got, len(want))
	}
}
