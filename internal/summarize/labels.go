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

// IsSummaryVectorID reports whether id uses a rollup summary prefix (EP-016 legacy vec_items filtering).
func IsSummaryVectorID(id string) bool {
	return strings.HasPrefix(id, VectorIDPrefixDay) ||
		strings.HasPrefix(id, VectorIDPrefixMonth) ||
		strings.HasPrefix(id, VectorIDPrefixYear)
}

// VectorChunkLabel returns the retrieval label for REQ-02.009 from a vector document id.
func VectorChunkLabel(id string) string {
	switch {
	case strings.HasPrefix(id, "notes:"):
		return "notes"
	case strings.HasPrefix(id, "turn:"):
		return "turn"
	case strings.HasPrefix(id, VectorIDPrefixDay):
		return "summary:day"
	case strings.HasPrefix(id, VectorIDPrefixMonth):
		return "summary:month"
	case strings.HasPrefix(id, VectorIDPrefixYear):
		return "summary:year"
	default:
		return "unknown"
	}
}

// FormatNotesVectorText builds stored vector text for a manual notes entry (EP-016).
func FormatNotesVectorText(dateISO, noteBody, kind string) string {
	var b strings.Builder
	b.WriteString("Date: ")
	b.WriteString(dateISO)
	b.WriteString("\n[notes]\n")
	if strings.TrimSpace(kind) != "" {
		b.WriteString("kind=")
		b.WriteString(strings.TrimSpace(strings.ToLower(kind)))
		b.WriteByte('\n')
	}
	b.WriteString(strings.TrimSpace(noteBody))
	return b.String()
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
