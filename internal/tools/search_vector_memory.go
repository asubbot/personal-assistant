package tools

import (
	"context"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/vector"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultSearchVectorMemoryTopK       = 5
	defaultSearchVectorMemoryMaxTopK    = 10
	defaultSearchVectorMemoryMaxOutByte = 4096
	defaultSearchVectorMemorySnippetLen = 200
)

var vectorMemoryLaneOrder = []string{"notes", "summaries", "turns"}

// SearchVectorMemoryTool performs read-only semantic retrieval from vector memory lanes.
type SearchVectorMemoryTool struct {
	notes          vector.Store
	summaries      vector.Store
	turns          vector.Store
	embedder       embedding.Embedder
	defaultTopK    int
	maxTopK        int
	maxOutputBytes int
}

// NewSearchVectorMemoryTool constructs search_vector_memory native tool with bounded output.
func NewSearchVectorMemoryTool(notes, summaries, turns vector.Store, embedder embedding.Embedder, defaultTopK, maxTopK, maxOutputBytes int) *SearchVectorMemoryTool {
	if defaultTopK < 1 {
		defaultTopK = defaultSearchVectorMemoryTopK
	}
	if maxTopK < 1 {
		maxTopK = defaultSearchVectorMemoryMaxTopK
	}
	if defaultTopK > maxTopK {
		defaultTopK = maxTopK
	}
	if maxOutputBytes < 256 {
		maxOutputBytes = defaultSearchVectorMemoryMaxOutByte
	}
	return &SearchVectorMemoryTool{
		notes:          notes,
		summaries:      summaries,
		turns:          turns,
		embedder:       embedder,
		defaultTopK:    defaultTopK,
		maxTopK:        maxTopK,
		maxOutputBytes: maxOutputBytes,
	}
}

func (t *SearchVectorMemoryTool) Name() string { return "search_vector_memory" }

func (t *SearchVectorMemoryTool) Description() string {
	return "Search vector memory semantically across notes, summaries, and turns. Arguments: query (required), lanes (optional array of notes/summaries/turns), top_k (optional integer). Returns bounded compact snippets."
}

func (t *SearchVectorMemoryTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{
		{Name: "query", Required: true, Type: "string"},
		{Name: "lanes", Required: false, Type: "array"},
		{Name: "top_k", Required: false, Type: "number"},
	}
}

func (t *SearchVectorMemoryTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if t == nil || t.embedder == nil {
		return "", fmt.Errorf("search_vector_memory: embedder is not configured")
	}
	if err := ValidateParams(t.ParamsSchema(), params); err != nil {
		return "", err
	}
	query := strings.TrimSpace(stringParam(params, "query"))
	if query == "" {
		return "", fmt.Errorf("search_vector_memory: query is required")
	}
	lanes, err := t.parseLanes(params["lanes"])
	if err != nil {
		return "", err
	}
	topK, err := t.parseTopK(params["top_k"])
	if err != nil {
		return "", err
	}
	emb, err := t.embedder.Embed(ctx, query)
	if err != nil {
		return "", fmt.Errorf("search_vector_memory: embed query failed: %w", err)
	}
	hits, err := t.collectHits(ctx, emb, lanes, topK)
	if err != nil {
		return "", err
	}
	return t.formatHits(lanes, topK, hits)
}

type vectorLaneHit struct {
	lane  string
	id    string
	score float64
	text  string
}

func (t *SearchVectorMemoryTool) collectHits(ctx context.Context, emb []float32, lanes []string, topK int) ([]vectorLaneHit, error) {
	var out []vectorLaneHit
	for _, lane := range lanes {
		store := t.storeForLane(lane)
		if store == nil {
			return nil, fmt.Errorf("search_vector_memory: lane %q is unavailable", lane)
		}
		res, err := store.Search(ctx, emb, topK)
		if err != nil {
			return nil, fmt.Errorf("search_vector_memory: lane %q search failed: %w", lane, err)
		}
		sort.SliceStable(res, func(i, j int) bool {
			if res[i].Score != res[j].Score {
				return res[i].Score < res[j].Score
			}
			return res[i].ID < res[j].ID
		})
		for _, r := range res {
			txt := strings.TrimSpace(r.Text)
			if txt == "" {
				continue
			}
			out = append(out, vectorLaneHit{
				lane:  lane,
				id:    strings.TrimSpace(r.ID),
				score: r.Score,
				text:  compactSingleLine(txt, defaultSearchVectorMemorySnippetLen),
			})
		}
	}
	return out, nil
}

func (t *SearchVectorMemoryTool) formatHits(lanes []string, topK int, hits []vectorLaneHit) (string, error) {
	header := fmt.Sprintf("Vector memory hits (lanes=%s, top_k=%d)\n", strings.Join(lanes, ","), topK)
	if len(hits) == 0 {
		return header + "(no hits)", nil
	}
	var b strings.Builder
	b.WriteString(header)
	included := 0
	omitted := 0
	for _, h := range hits {
		row := fmt.Sprintf("- [%s] %s score=%.6f %s\n", h.lane, fallbackID(h.id), h.score, h.text)
		if b.Len()+len(row) > t.maxOutputBytes {
			omitted++
			continue
		}
		b.WriteString(row)
		included++
	}
	if included == 0 {
		return "", fmt.Errorf("search_vector_memory: output exceeds max_output_bytes (%d)", t.maxOutputBytes)
	}
	if omitted > 0 {
		foot := fmt.Sprintf("[truncated: %d items omitted]", omitted)
		if b.Len()+len(foot)+1 <= t.maxOutputBytes {
			b.WriteString(foot)
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func fallbackID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "(no-id)"
	}
	return strings.TrimSpace(id)
}

func compactSingleLine(s string, maxRunes int) string {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return ""
	}
	joined := strings.Join(parts, " ")
	if maxRunes < 1 {
		return joined
	}
	r := []rune(joined)
	if len(r) <= maxRunes {
		return joined
	}
	return strings.TrimSpace(string(r[:maxRunes])) + "..."
}

func (t *SearchVectorMemoryTool) parseLanes(raw any) ([]string, error) {
	if raw == nil {
		return t.defaultLanes(), nil
	}
	var lanes []string
	switch vv := raw.(type) {
	case []string:
		lanes = append(lanes, vv...)
	case []any:
		for _, x := range vv {
			s, ok := x.(string)
			if !ok {
				return nil, fmt.Errorf("search_vector_memory: lanes must contain strings only")
			}
			lanes = append(lanes, s)
		}
	default:
		return nil, fmt.Errorf("search_vector_memory: lanes must be an array")
	}
	if len(lanes) == 0 {
		return t.defaultLanes(), nil
	}
	seen := make(map[string]struct{}, len(lanes))
	var out []string
	for _, lane := range lanes {
		n := strings.ToLower(strings.TrimSpace(lane))
		if !slices.Contains(vectorMemoryLaneOrder, n) {
			return nil, fmt.Errorf("search_vector_memory: invalid lane %q", lane)
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return slices.Index(vectorMemoryLaneOrder, out[i]) < slices.Index(vectorMemoryLaneOrder, out[j])
	})
	return out, nil
}

func (t *SearchVectorMemoryTool) parseTopK(raw any) (int, error) {
	if raw == nil {
		return t.defaultTopK, nil
	}
	n, err := toInt(raw)
	if err != nil {
		return 0, fmt.Errorf("search_vector_memory: top_k must be an integer")
	}
	if n < 1 || n > t.maxTopK {
		return 0, fmt.Errorf("search_vector_memory: top_k must be in 1..%d", t.maxTopK)
	}
	return n, nil
}

func toInt(v any) (int, error) {
	switch x := v.(type) {
	case int:
		return x, nil
	case int64:
		return int(x), nil
	case float64:
		if x != float64(int(x)) {
			return 0, fmt.Errorf("not integer")
		}
		return int(x), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(x))
	default:
		return 0, fmt.Errorf("unsupported")
	}
}

func (t *SearchVectorMemoryTool) defaultLanes() []string {
	out := make([]string, 0, len(vectorMemoryLaneOrder))
	for _, lane := range vectorMemoryLaneOrder {
		if t.storeForLane(lane) != nil {
			out = append(out, lane)
		}
	}
	return out
}

func (t *SearchVectorMemoryTool) storeForLane(lane string) vector.Store {
	switch lane {
	case "notes":
		return t.notes
	case "summaries":
		return t.summaries
	case "turns":
		return t.turns
	default:
		return nil
	}
}
