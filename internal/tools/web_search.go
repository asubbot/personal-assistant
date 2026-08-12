package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"pa/internal/config"
	"regexp"
	"strings"
	"time"
)

// WebSearchTool implements native web_search (EP-011).
type WebSearchTool struct {
	cfg    *config.WebToolsConfig
	client *http.Client
	cache  *searchCache
	now    func() time.Time
}

// NewWebSearchTool builds web_search. cfg must be non-nil with Enabled true; client must be non-nil.
func NewWebSearchTool(cfg *config.WebToolsConfig, client *http.Client, now func() time.Time) *WebSearchTool {
	if now == nil {
		now = time.Now
	}
	ttl := time.Duration(cfg.Search.CacheTTLSeconds) * time.Second
	return &WebSearchTool{
		cfg:    cfg,
		client: client,
		cache:  newSearchCache(cfg.Search.CacheMaxEntries, ttl, now),
		now:    now,
	}
}

// Name implements Tool.
func (w *WebSearchTool) Name() string { return "web_search" }

// Description implements Tool.
func (w *WebSearchTool) Description() string {
	return `Search the public web. Input: query (string). Returns JSON array of {title, url, snippet}. Uses configured primary provider (Brave Search or DuckDuckGo) and optional fallback on failure.`
}

// ParamsSchema implements Tool.
func (w *WebSearchTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{{Name: "query", Required: true, Type: "string"}}
}

type webHit struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Run implements Tool.
func (w *WebSearchTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if err := ValidateParams(w.ParamsSchema(), params); err != nil {
		return "", err
	}
	q, _ := params["query"].(string)
	nq := normalizeSearchQuery(q)
	if nq == "" {
		return "", errors.New("web_search: query is empty")
	}
	chain := searchProviderChain(w.cfg.Search)
	key := searchCacheKey(chain, nq)
	if payload, hit := w.cache.get(key); hit {
		return payload, nil
	}

	timeout := time.Duration(w.cfg.Search.TimeoutSeconds) * time.Second
	var hits []webHit
	var err error
	for i, prov := range chain {
		upCtx, cancel := context.WithTimeout(ctx, timeout)
		hits, err = w.searchByProvider(upCtx, prov, q)
		cancel()
		if err == nil {
			break
		}
		if i == len(chain)-1 {
			return "", err
		}
	}
	out, err := json.Marshal(hits)
	if err != nil {
		return "", fmt.Errorf("web_search: encode: %w", err)
	}
	s := string(out)
	w.cache.set(key, s)
	return s, nil
}

func normalizeSearchQuery(q string) string {
	s := strings.TrimSpace(q)
	if s == "" {
		return ""
	}
	parts := strings.Fields(s)
	return strings.ToLower(strings.Join(parts, " "))
}

func searchProviderChain(s config.WebSearchConfig) []string {
	p := strings.ToLower(strings.TrimSpace(s.Provider))
	out := []string{p}
	if fb := strings.TrimSpace(s.FallbackProvider); fb != "" {
		out = append(out, strings.ToLower(fb))
	}
	return out
}

func searchCacheKey(chain []string, rawQuery string) string {
	return strings.Join(chain, ">") + "|" + normalizeSearchQuery(rawQuery)
}

func (w *WebSearchTool) searchByProvider(ctx context.Context, prov, query string) ([]webHit, error) {
	switch prov {
	case "brave":
		return w.searchBrave(ctx, query)
	case "duckduckgo":
		return w.searchDDG(ctx, query)
	default:
		return nil, fmt.Errorf("web_search: unknown provider %q", prov)
	}
}

func readHTTPBody(resp *http.Response, max int64) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func readBraveAPIKey(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("web_search: brave_api_key_path is not configured")
	}
	keyBytes, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(keyBytes)) == "" {
		return "", errors.New("web_search: brave API key file missing or empty")
	}
	return strings.TrimSpace(string(keyBytes)), nil
}

func braveSearchRequestURL(query string) (string, error) {
	u, err := url.Parse("https://api.search.brave.com/res/v1/web/search")
	if err != nil {
		return "", err
	}
	// sql-rows-close: safe — url.URL.Query returns url.Values, not database/sql rows
	q := u.Query()
	q.Set("q", query)
	q.Set("count", "10")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseBraveSearchJSON(body []byte) ([]webHit, error) {
	var br struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &br); err != nil {
		return nil, errors.New("web_search: invalid brave response")
	}
	var hits []webHit
	for _, r := range br.Web.Results {
		if strings.TrimSpace(r.URL) == "" {
			continue
		}
		hits = append(hits, webHit{Title: r.Title, URL: r.URL, Snippet: r.Description})
	}
	if len(hits) == 0 {
		return nil, errors.New("web_search: no results")
	}
	return hits, nil
}

func (w *WebSearchTool) searchBrave(ctx context.Context, query string) ([]webHit, error) {
	apiKey, err := readBraveAPIKey(w.cfg.Search.BraveAPIKeyPath)
	if err != nil {
		return nil, err
	}
	urlStr, err := braveSearchRequestURL(query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Subscription-Token", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("web_search: upstream timeout")
		}
		return nil, fmt.Errorf("web_search: upstream request failed")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("web_search: upstream status %d", resp.StatusCode)
	}
	body, err := readHTTPBody(resp, 4<<20)
	if err != nil {
		return nil, errors.New("web_search: failed to read upstream body")
	}
	return parseBraveSearchJSON(body)
}

var (
	reDDGLink = regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>([^<]*)</a>`)
	reDDGSnip = regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>([^<]*)</a>`)
)

func ddgHTMLSearchURL(query string) (string, error) {
	u, err := url.Parse("https://html.duckduckgo.com/html/")
	if err != nil {
		return "", err
	}
	// sql-rows-close: safe — url.URL.Query returns url.Values, not database/sql rows
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseDDGHTMLHits(body []byte) ([]webHit, error) {
	links := reDDGLink.FindAllStringSubmatch(string(body), -1)
	snips := reDDGSnip.FindAllStringSubmatch(string(body), -1)
	var hits []webHit
	for i, m := range links {
		if len(m) < 3 {
			continue
		}
		href, title := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		if href == "" || !strings.HasPrefix(href, "http") {
			continue
		}
		snippet := ""
		if i < len(snips) && len(snips[i]) > 1 {
			snippet = strings.TrimSpace(snips[i][1])
		}
		hits = append(hits, webHit{Title: title, URL: href, Snippet: snippet})
		if len(hits) >= 10 {
			break
		}
	}
	if len(hits) == 0 {
		return nil, errors.New("web_search: no results")
	}
	return hits, nil
}

func (w *WebSearchTool) searchDDG(ctx context.Context, query string) ([]webHit, error) {
	urlStr, err := ddgHTMLSearchURL(query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "PersonalAssistant/1.0 (+https://github.com/)")

	resp, err := w.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("web_search: upstream timeout")
		}
		return nil, fmt.Errorf("web_search: upstream request failed")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("web_search: upstream status %d", resp.StatusCode)
	}
	body, err := readHTTPBody(resp, 4<<20)
	if err != nil {
		return nil, errors.New("web_search: failed to read upstream body")
	}
	return parseDDGHTMLHits(body)
}
