package memoryjob

import (
	"context"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/memory"
	"pa/internal/summarize"
	"pa/internal/vector"
	"strings"
	"time"
)

// ReindexDaySummary embeds an existing day summary file into the vector store (no LLM).
func ReindexDaySummary(ctx context.Context, mem *memory.Store, vs vector.Store, emb embedding.Embedder, loc *time.Location, dateISO string) error {
	if mem == nil || vs == nil || emb == nil {
		return fmt.Errorf("reindex day: missing dependency")
	}
	if loc == nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(dateISO), loc)
	if err != nil {
		return fmt.Errorf("reindex day: date: %w", err)
	}
	day := time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, loc)
	body, err := mem.ReadDaySummary(ctx, day)
	if err != nil {
		return err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	dateStr := day.In(loc).Format("2006-01-02")
	id := summarize.VectorIDPrefixDay + dateStr
	vecText := summarize.FormatDayVectorText(dateStr, body)
	_ = vs.Delete(ctx, id)
	ev, err := emb.Embed(ctx, vecText)
	if err != nil {
		return err
	}
	return vs.Add(ctx, id, ev, vecText)
}

// ReindexMonthSummary embeds an existing month summary file (no LLM).
func ReindexMonthSummary(ctx context.Context, mem *memory.Store, vs vector.Store, emb embedding.Embedder, year, month int) error {
	if mem == nil || vs == nil || emb == nil {
		return fmt.Errorf("reindex month: missing dependency")
	}
	body, err := mem.ReadMonthSummary(ctx, year, month)
	if err != nil {
		return err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	id := summarize.VectorIDPrefixMonth + fmt.Sprintf("%04d-%02d", year, month)
	vecText := summarize.FormatMonthVectorText(year, month, body)
	_ = vs.Delete(ctx, id)
	ev, err := emb.Embed(ctx, vecText)
	if err != nil {
		return err
	}
	return vs.Add(ctx, id, ev, vecText)
}

// ReindexYearSummary embeds an existing year summary file (no LLM).
func ReindexYearSummary(ctx context.Context, mem *memory.Store, vs vector.Store, emb embedding.Embedder, year int) error {
	if mem == nil || vs == nil || emb == nil {
		return fmt.Errorf("reindex year: missing dependency")
	}
	body, err := mem.ReadYearSummary(ctx, year)
	if err != nil {
		return err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	id := summarize.VectorIDPrefixYear + fmt.Sprintf("%04d", year)
	vecText := summarize.FormatYearVectorText(year, body)
	_ = vs.Delete(ctx, id)
	ev, err := emb.Embed(ctx, vecText)
	if err != nil {
		return err
	}
	return vs.Add(ctx, id, ev, vecText)
}
