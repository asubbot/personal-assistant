package tools

import (
	"context"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/vector"
	"sort"
	"strings"
)

type searchVectorKnowledgeTool struct {
	name           string
	description    string
	headerLabel    string
	store          vector.Store
	embedder       embedding.Embedder
	defaultTopK    int
	maxTopK        int
	maxOutputBytes int
	snippetRunes   int
}

func NewSearchVectorToolKnowledgeTool(store vector.Store, embedder embedding.Embedder, defaultTopK, maxTopK, maxOutputBytes, snippetRunes int) Tool {
	return &searchVectorKnowledgeTool{
		name:           "search_vector_tool",
		description:    "Search tool knowledge vectors semantically. Arguments: query (required), top_k (optional integer). Returns bounded compact snippets.",
		headerLabel:    "Tool knowledge hits",
		store:          store,
		embedder:       embedder,
		defaultTopK:    normalizeTopK(defaultTopK),
		maxTopK:        normalizeMaxTopK(maxTopK),
		maxOutputBytes: normalizeMaxOutputBytes(maxOutputBytes),
		snippetRunes:   normalizeSnippetRunes(snippetRunes),
	}
}

func NewSearchVectorSkillKnowledgeTool(store vector.Store, embedder embedding.Embedder, defaultTopK, maxTopK, maxOutputBytes, snippetRunes int) Tool {
	return &searchVectorKnowledgeTool{
		name:           "search_vector_skill",
		description:    "Search skill knowledge vectors semantically. Arguments: query (required), top_k (optional integer). Returns bounded compact snippets.",
		headerLabel:    "Skill knowledge hits",
		store:          store,
		embedder:       embedder,
		defaultTopK:    normalizeTopK(defaultTopK),
		maxTopK:        normalizeMaxTopK(maxTopK),
		maxOutputBytes: normalizeMaxOutputBytes(maxOutputBytes),
		snippetRunes:   normalizeSnippetRunes(snippetRunes),
	}
}

func normalizeTopK(v int) int {
	if v < 1 {
		return 5
	}
	return v
}

func normalizeMaxTopK(v int) int {
	if v < 1 {
		return 10
	}
	return v
}

func normalizeMaxOutputBytes(v int) int {
	if v < 256 {
		return 4096
	}
	return v
}

func normalizeSnippetRunes(v int) int {
	if v < 32 {
		return 200
	}
	return v
}

func (t *searchVectorKnowledgeTool) Name() string { return t.name }

func (t *searchVectorKnowledgeTool) Description() string { return t.description }

func (t *searchVectorKnowledgeTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{
		{Name: "query", Required: true, Type: "string"},
		{Name: "top_k", Required: false, Type: "number"},
	}
}

func (t *searchVectorKnowledgeTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if t == nil || t.embedder == nil {
		return "", fmt.Errorf("%s: embedder is not configured", t.name)
	}
	if t.store == nil {
		return "", fmt.Errorf("%s: store is not configured", t.name)
	}
	if err := ValidateParams(t.ParamsSchema(), params); err != nil {
		return "", err
	}
	query := strings.TrimSpace(stringParam(params, "query"))
	if query == "" {
		return "", fmt.Errorf("%s: query is required", t.name)
	}
	topK, err := t.parseTopK(params["top_k"])
	if err != nil {
		return "", err
	}
	emb, err := t.embedder.Embed(ctx, query)
	if err != nil {
		return "", fmt.Errorf("%s: embed query failed: %w", t.name, err)
	}
	res, err := t.store.Search(ctx, emb, topK)
	if err != nil {
		return "", fmt.Errorf("%s: search failed: %w", t.name, err)
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].Score != res[j].Score {
			return res[i].Score < res[j].Score
		}
		return res[i].ID < res[j].ID
	})
	return t.formatHits(topK, res)
}

func (t *searchVectorKnowledgeTool) parseTopK(raw any) (int, error) {
	if raw == nil {
		return t.defaultTopK, nil
	}
	n, err := toInt(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: top_k must be an integer", t.name)
	}
	if n < 1 || n > t.maxTopK {
		return 0, fmt.Errorf("%s: top_k must be in 1..%d", t.name, t.maxTopK)
	}
	return n, nil
}

func (t *searchVectorKnowledgeTool) formatHits(topK int, hits []vector.SearchResult) (string, error) {
	header := fmt.Sprintf("%s (top_k=%d)\n", t.headerLabel, topK)
	if len(hits) == 0 {
		return header + "(no hits)", nil
	}
	var b strings.Builder
	b.WriteString(header)
	included := 0
	omitted := 0
	for _, h := range hits {
		text := compactSingleLine(strings.TrimSpace(h.Text), t.snippetRunes)
		if text == "" {
			continue
		}
		row := fmt.Sprintf("- %s score=%.6f %s\n", fallbackID(strings.TrimSpace(h.ID)), h.Score, text)
		if b.Len()+len(row) > t.maxOutputBytes {
			omitted++
			continue
		}
		b.WriteString(row)
		included++
	}
	if included == 0 {
		return "", fmt.Errorf("%s: output exceeds max_output_bytes (%d)", t.name, t.maxOutputBytes)
	}
	if omitted > 0 {
		foot := fmt.Sprintf("[truncated: %d items omitted]", omitted)
		if b.Len()+len(foot)+1 <= t.maxOutputBytes {
			b.WriteString(foot)
		}
	}
	return strings.TrimSpace(b.String()), nil
}
