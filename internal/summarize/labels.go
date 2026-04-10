package summarize

import (
	"fmt"
	"strings"
)

// Stable vector document id prefixes for summaries (upsert / reconciliation).
const (
	VectorIDPrefixDay   = "summary:day:"
	VectorIDPrefixMonth = "summary:month:"
	VectorIDPrefixYear  = "summary:year:"
)

// VectorChunkLabel returns the retrieval label for REQ-02.009 from a vector document id.
func VectorChunkLabel(id string) string {
	switch {
	case strings.HasPrefix(id, VectorIDPrefixDay):
		return "summary:day"
	case strings.HasPrefix(id, VectorIDPrefixMonth):
		return "summary:month"
	case strings.HasPrefix(id, VectorIDPrefixYear):
		return "summary:year"
	default:
		return "turn"
	}
}

// FormatDayVectorText builds stored vector text for a day summary (REQ-02.008).
func FormatDayVectorText(dateISO, summaryBody string) string {
	return "Date: " + dateISO + "\n[summary:day]\n" + strings.TrimSpace(summaryBody)
}

// FormatMonthVectorText builds stored vector text for a month summary.
func FormatMonthVectorText(year, month int, summaryBody string) string {
	return "Date: " + fmtYMDMonth(year, month) + "\n[summary:month]\n" + strings.TrimSpace(summaryBody)
}

// FormatYearVectorText builds stored vector text for a year summary.
func FormatYearVectorText(year int, summaryBody string) string {
	return "Date: " + fmt.Sprintf("%04d", year) + "\n[summary:year]\n" + strings.TrimSpace(summaryBody)
}

func fmtYMDMonth(year, month int) string {
	return fmt.Sprintf("%04d-%02d", year, month)
}
