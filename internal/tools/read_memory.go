package tools

import (
	"context"
	"fmt"
	"os"
	"pa/internal/memory"
	"path/filepath"
	"strings"
	"time"
)

// ReadMemoryTool reads day summaries from memory_dir for ISO date or inclusive from–to range (EP-002).
type ReadMemoryTool struct {
	store       *memory.Store
	maxSpanDays int
	maxOutBytes int
}

// NewReadMemoryTool returns a native tool; store must be non-nil.
func NewReadMemoryTool(store *memory.Store, maxSpanDays, maxOutputBytes int) *ReadMemoryTool {
	return &ReadMemoryTool{store: store, maxSpanDays: maxSpanDays, maxOutBytes: maxOutputBytes}
}

func (t *ReadMemoryTool) Name() string { return "read_memory" }

func (t *ReadMemoryTool) Description() string {
	return "Read long-term memory day summaries from the assistant memory store. Use either a single ISO date (YYYY-MM-DD) or from and to (inclusive range). Dates are interpreted in pa_timezone."
}

func (t *ReadMemoryTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{
		{Name: "date", Required: false, Type: "string"},
		{Name: "from", Required: false, Type: "string"},
		{Name: "to", Required: false, Type: "string"},
	}
}

func (t *ReadMemoryTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("read_memory: memory store not configured")
	}
	if err := ValidateParams(t.ParamsSchema(), params); err != nil {
		return "", err
	}
	date := strings.TrimSpace(stringParam(params, "date"))
	from := strings.TrimSpace(stringParam(params, "from"))
	to := strings.TrimSpace(stringParam(params, "to"))
	loc := t.store.Location()
	if date != "" && (from != "" || to != "") {
		return "", fmt.Errorf("read_memory: use either date or from/to, not both")
	}
	if date != "" {
		dt, err := parseISODateInLoc(date, loc)
		if err != nil {
			return "", fmt.Errorf("read_memory: %w", err)
		}
		return t.readDayRange(ctx, dt, dt, loc)
	}
	if from == "" || to == "" {
		return "", fmt.Errorf("read_memory: provide date or both from and to")
	}
	return t.runRange(ctx, from, to, loc)
}

func (t *ReadMemoryTool) runRange(ctx context.Context, from, to string, loc *time.Location) (string, error) {
	a, err := parseISODateInLoc(from, loc)
	if err != nil {
		return "", fmt.Errorf("read_memory: from: %w", err)
	}
	b, err := parseISODateInLoc(to, loc)
	if err != nil {
		return "", fmt.Errorf("read_memory: to: %w", err)
	}
	if b.Before(a) {
		a, b = b, a
	}
	days := calendarDaysInclusive(a, b, loc)
	if days > t.maxSpanDays {
		return "", fmt.Errorf("read_memory: range spans %d days (max %d)", days, t.maxSpanDays)
	}
	return t.readDayRange(ctx, a, b, loc)
}

func stringParam(params map[string]any, key string) string {
	v, ok := params[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func parseISODateInLoc(s string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	t, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid ISO date %q (use YYYY-MM-DD)", s)
	}
	return t, nil
}

func calendarDaysInclusive(a, b time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.UTC
	}
	ay, am, ad := a.In(loc).Date()
	by, bm, bd := b.In(loc).Date()
	start := time.Date(ay, am, ad, 0, 0, 0, 0, loc)
	end := time.Date(by, bm, bd, 0, 0, 0, 0, loc)
	if end.Before(start) {
		start, end = end, start
	}
	n := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		n++
	}
	return n
}

// readMemoryDayBlock returns one formatted day section, skip=true when both summary and notes are empty.
func (t *ReadMemoryTool) readMemoryDayBlock(ctx context.Context, d time.Time, loc *time.Location) (bs string, skip bool, err error) {
	day := d
	sumPath := daySummaryPathForCheck(t.store, day)
	if !underMemoryRoot(t.store.RootDir(), sumPath) {
		return "", false, fmt.Errorf("read_memory: path outside memory_dir")
	}
	notesPath := t.store.NotesPathForDay(day)
	if !underMemoryRoot(t.store.RootDir(), notesPath) {
		return "", false, fmt.Errorf("read_memory: path outside memory_dir")
	}
	summaryText, err := t.store.ReadDaySummary(ctx, day)
	if err != nil {
		return "", false, err
	}
	notesText, err := t.store.ReadDayNotes(ctx, day)
	if err != nil {
		return "", false, err
	}
	summaryText = strings.TrimSpace(summaryText)
	notesText = strings.TrimSpace(notesText)
	if summaryText == "" && notesText == "" {
		return "", true, nil
	}
	dateStr := d.In(loc).Format("2006-01-02")
	var block strings.Builder
	block.WriteString("## ")
	block.WriteString(dateStr)
	block.WriteByte('\n')
	if summaryText != "" {
		block.WriteString("### Automatic summary\n")
		block.WriteString(summaryText)
		block.WriteByte('\n')
	}
	if notesText != "" {
		block.WriteString("### Manual notes\n")
		block.WriteString(notesText)
		block.WriteByte('\n')
	}
	block.WriteByte('\n')
	return block.String(), false, nil
}

func (t *ReadMemoryTool) readDayRange(ctx context.Context, from, to time.Time, loc *time.Location) (string, error) {
	if loc == nil {
		loc = time.UTC
	}
	// Anchor iteration at noon in pa_timezone (same as memory/summarize paths) to avoid DST
	// midnight edge cases when from/to come from ParseInLocation at 00:00:00.
	fl := from.In(loc)
	tl := to.In(loc)
	from = time.Date(fl.Year(), fl.Month(), fl.Day(), 12, 0, 0, 0, loc)
	to = time.Date(tl.Year(), tl.Month(), tl.Day(), 12, 0, 0, 0, loc)
	if to.Before(from) {
		from, to = to, from
	}
	var b strings.Builder
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		bs, skip, err := t.readMemoryDayBlock(ctx, d, loc)
		if err != nil {
			return "", err
		}
		if skip {
			continue
		}
		if b.Len()+len(bs) > t.maxOutBytes {
			return "", fmt.Errorf("read_memory: output would exceed max_output_bytes (%d)", t.maxOutBytes)
		}
		b.WriteString(bs)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "(no day summaries or notes in range)", nil
	}
	if len(out) > t.maxOutBytes {
		return "", fmt.Errorf("read_memory: output exceeds max_output_bytes (%d)", t.maxOutBytes)
	}
	return out, nil
}

func daySummaryPathForCheck(s *memory.Store, day time.Time) string {
	// Use same layout as memory.Store without duplicating unexported logic: resolve via Read path construction.
	// We only need a path for prefix check; mirror memory.Store path layout.
	loc := s.Location()
	y, m, d := day.In(loc).Date()
	return filepath.Join(s.RootDir(), fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d", d), "summary.md")
}

func underMemoryRoot(root, path string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".."
}
