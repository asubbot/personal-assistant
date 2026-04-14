package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/memory"
	"pa/internal/promptmarkers"
	"pa/internal/summarize"
	"pa/internal/vector"
	"strings"
	"time"
)

// WriteMemoryTool appends to notes.md and indexes vec_notes (EP-016).
type WriteMemoryTool struct {
	store      *memory.Store
	noteVector vector.Store
	embedder   embedding.Embedder
	maxAppend  int
	maxFile    int
}

// NewWriteMemoryTool returns a native tool; store, noteVector, and embedder must be non-nil for full operation.
func NewWriteMemoryTool(store *memory.Store, noteVector vector.Store, embedder embedding.Embedder, maxAppend, maxFile int) *WriteMemoryTool {
	return &WriteMemoryTool{store: store, noteVector: noteVector, embedder: embedder, maxAppend: maxAppend, maxFile: maxFile}
}

func (t *WriteMemoryTool) Name() string { return "write_memory" }

func (t *WriteMemoryTool) Description() string {
	return "Append a manual note to long-term memory (notes.md) for one calendar day in pa_timezone, then index it for semantic retrieval. Arguments: text (required), optional date (YYYY-MM-DD, default today), optional kind (fact, guideline, preference, other)."
}

func (t *WriteMemoryTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{
		{Name: "text", Required: true, Type: "string"},
		{Name: "date", Required: false, Type: "string"},
		{Name: "kind", Required: false, Type: "string"},
	}
}

func resolveWriteMemoryDay(store *memory.Store, dateStr string) (day time.Time, loc *time.Location, err error) {
	loc = store.Location()
	if strings.TrimSpace(dateStr) == "" {
		day = time.Now().In(loc)
		day = time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc)
		return day, loc, nil
	}
	day, err = parseISODateInLoc(dateStr, loc)
	if err != nil {
		return time.Time{}, loc, err
	}
	return day, loc, nil
}

func (t *WriteMemoryTool) indexNoteVector(ctx context.Context, day time.Time, loc *time.Location, text, kind string, nowUTC time.Time) (string, error) {
	dateISO := day.In(loc).Format("2006-01-02")
	idSeed := nowUTC.Format(time.RFC3339Nano) + "\n" + text
	sum := sha256.Sum256([]byte(idSeed))
	id := "notes:" + dateISO + ":" + hex.EncodeToString(sum[:8])
	vecBody := summarize.FormatNotesVectorText(dateISO, text, kind)
	if promptmarkers.TextContainsForbiddenMarkerLine(vecBody) {
		return "", fmt.Errorf("write_memory: indexed text contains forbidden PA marker line")
	}
	_ = t.noteVector.Delete(ctx, id)
	emb, err := t.embedder.Embed(ctx, vecBody)
	if err != nil {
		return "", fmt.Errorf("write_memory: vector embed failed (note saved on disk): %w", err)
	}
	if err := t.noteVector.Add(ctx, id, emb, vecBody); err != nil {
		return "", fmt.Errorf("write_memory: vector index failed (note saved on disk): %w", err)
	}
	return id, nil
}

func (t *WriteMemoryTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("write_memory: memory store not configured")
	}
	if err := ValidateParams(t.ParamsSchema(), params); err != nil {
		return "", err
	}
	text := strings.TrimSpace(stringParam(params, "text"))
	if text == "" {
		return "", fmt.Errorf("write_memory: text is required")
	}
	dateStr := strings.TrimSpace(stringParam(params, "date"))
	kind := strings.TrimSpace(stringParam(params, "kind"))
	day, loc, err := resolveWriteMemoryDay(t.store, dateStr)
	if err != nil {
		return "", fmt.Errorf("write_memory: %w", err)
	}
	path := t.store.NotesPathForDay(day)
	if !underMemoryRoot(t.store.RootDir(), path) {
		return "", fmt.Errorf("write_memory: path outside memory_dir")
	}
	nowUTC := time.Now().UTC()
	if err := t.store.AppendDayNote(ctx, day, text, kind, nowUTC, t.maxAppend, t.maxFile); err != nil {
		return "", fmt.Errorf("write_memory: %w", err)
	}
	if t.noteVector == nil || t.embedder == nil {
		return "Note saved (vector index not configured).", nil
	}
	dateISO := day.In(loc).Format("2006-01-02")
	id, err := t.indexNoteVector(ctx, day, loc, text, kind, nowUTC)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Saved note for %s (id %s).", dateISO, id), nil
}
