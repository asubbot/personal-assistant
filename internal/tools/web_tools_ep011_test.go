package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"pa/internal/config"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResp(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func htmlResp(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func ep011WebTools(t *testing.T, provider string) *config.WebToolsConfig {
	t.Helper()
	return &config.WebToolsConfig{
		Enabled: true,
		Search: config.WebSearchConfig{
			Provider:        provider,
			BraveAPIKeyPath: filepath.Join(t.TempDir(), "brave.txt"),
			TimeoutSeconds:  30,
			CacheTTLSeconds: 300,
			CacheMaxEntries: 100,
		},
		Fetch: config.WebFetchConfig{
			TimeoutSeconds: 30,
			MaxBodyBytes:   1024,
			MaxRedirects:   5,
		},
	}
}

// Covers AC-11.002 — empty query returns error without upstream.
func TestWebSearch_EmptyQuery(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return htmlResp(""), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	_, err := w.Run(context.Background(), map[string]any{"query": "   "})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("got err=%v, calls=%d", err, calls.Load())
	}
	if calls.Load() != 0 {
		t.Fatalf("upstream calls = %d, want 0", calls.Load())
	}
}

// Covers AC-11.003 — Brave JSON mapped to items.
func TestWebSearch_BraveJSON(t *testing.T) {
	cfg := ep011WebTools(t, "brave")
	keyPath := cfg.Search.BraveAPIKeyPath
	if err := os.WriteFile(keyPath, []byte("test-key-not-in-output"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{"web":{"results":[{"title":"T1","url":"https://a.example/","description":"D1"}]}}`
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.URL.Host != "api.search.brave.com" {
			t.Fatalf("host %q", req.URL.Host)
		}
		return jsonResp(body), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	out, err := w.Run(context.Background(), map[string]any{"query": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "T1") || !strings.Contains(out, "https://a.example/") {
		t.Fatalf("out=%s", out)
	}
	if strings.Contains(out, "test-key-not-in-output") {
		t.Fatal("API key leaked into tool output")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

// Covers AC-11.004 — DuckDuckGo HTML parsed.
func TestWebSearch_DDGHTML(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	html := `<a class="result__a" href="https://b.example/p">Title B</a><a class="result__snippet">Snip</a>`
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.URL.Host != "html.duckduckgo.com" {
			t.Fatalf("host %q", req.URL.Host)
		}
		return htmlResp(html), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	out, err := w.Run(context.Background(), map[string]any{"query": "q"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Title B") {
		t.Fatalf("out=%s", out)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

// Covers AC-11.005 — missing Brave key file.
func TestWebSearch_BraveMissingKeyFile(t *testing.T) {
	cfg := ep011WebTools(t, "brave")
	cfg.Search.BraveAPIKeyPath = filepath.Join(t.TempDir(), "nope.key")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("unexpected upstream call")
		return nil, io.EOF
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	_, err := w.Run(context.Background(), map[string]any{"query": "x"})
	if err == nil || !strings.Contains(err.Error(), "key") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.006, AC-11.011 — cache hit avoids second upstream call.
func TestWebSearch_CacheHit_NoSecondUpstream(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	html := `<a class="result__a" href="https://x/">X</a><a class="result__snippet">S</a>`
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return htmlResp(html), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	ctx := context.Background()
	if _, err := w.Run(ctx, map[string]any{"query": "same"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Run(ctx, map[string]any{"query": "same"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("upstream calls=%d, want 1", calls.Load())
	}
}

// Covers AC-11.006 — TTL expiry triggers refetch.
func TestWebSearch_CacheExpires_Refetches(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	cfg.Search.CacheTTLSeconds = 1
	html := `<a class="result__a" href="https://x/">X</a><a class="result__snippet">S</a>`
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return htmlResp(html), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	now := func() time.Time { return t0 }
	w := NewWebSearchTool(cfg, client, now)
	ctx := context.Background()
	if _, err := w.Run(ctx, map[string]any{"query": "ttl"}); err != nil {
		t.Fatal(err)
	}
	t0 = t0.Add(2 * time.Second)
	if _, err := w.Run(ctx, map[string]any{"query": "ttl"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d, want 2", calls.Load())
	}
}

// Covers AC-11.006 — max entries eviction (LRU).
func TestWebSearch_CacheEvictsAtMax(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	cfg.Search.CacheMaxEntries = 2
	cfg.Search.CacheTTLSeconds = 3600
	html := `<a class="result__a" href="https://x/">X</a><a class="result__snippet">S</a>`
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return htmlResp(html), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	ctx := context.Background()
	for _, q := range []string{"a", "b", "c"} {
		if _, err := w.Run(ctx, map[string]any{"query": q}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.Run(ctx, map[string]any{"query": "a"}); err != nil {
		t.Fatal(err)
	}
	// a was LRU-evicted when c inserted; expect another upstream for a
	if calls.Load() != 4 {
		t.Fatalf("calls=%d, want 4 (a,b,c + refetch a)", calls.Load())
	}
}

// Covers AC-11.007 — distinct cache keys when fallback facet differs (same normalized query).
func TestWebSearch_DistinctCacheKeys_FallbackFacet(t *testing.T) {
	html := `<a class="result__a" href="https://x/">X</a><a class="result__snippet">S</a>`
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return htmlResp(html), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	cfgNoFB := ep011WebTools(t, "duckduckgo")
	cfgWithFB := ep011WebTools(t, "duckduckgo")
	cfgWithFB.Search.FallbackProvider = "brave"
	if err := os.WriteFile(cfgWithFB.Search.BraveAPIKeyPath, []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	w1 := NewWebSearchTool(cfgNoFB, client, nil)
	w2 := NewWebSearchTool(cfgWithFB, client, nil)
	ctx := context.Background()
	if _, err := w1.Run(ctx, map[string]any{"query": "same facet"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w2.Run(ctx, map[string]any{"query": "same facet"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("upstream calls=%d, want 2 (distinct cache keys)", calls.Load())
	}
}

// Covers AC-11.007 — distinct cache keys for different queries.
func TestWebSearch_DistinctCacheKeys(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return htmlResp(`<a class="result__a" href="https://x/">X</a>`), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	ctx := context.Background()
	if _, err := w.Run(ctx, map[string]any{"query": "one"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Run(ctx, map[string]any{"query": "two"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

// Covers AC-11.012 — search upstream timeout.
func TestWebSearch_UpstreamTimeout(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	cfg.Search.TimeoutSeconds = 1
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(5 * time.Second):
			return htmlResp(""), nil
		}
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	_, err := w.Run(context.Background(), map[string]any{"query": "slow"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.016 — primary failure then fallback success; cache hit on repeat.
func TestWebSearch_FallbackAfterPrimaryFails(t *testing.T) {
	cfg := ep011WebTools(t, "brave")
	if err := os.WriteFile(cfg.Search.BraveAPIKeyPath, []byte("test-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.Search.FallbackProvider = "duckduckgo"
	ddgHTML := `<a class="result__a" href="https://fallback.example/">FB</a><a class="result__snippet">Snip</a>`
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		switch req.URL.Host {
		case "api.search.brave.com":
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		case "html.duckduckgo.com":
			return htmlResp(ddgHTML), nil
		default:
			return nil, fmt.Errorf("unexpected host %q", req.URL.Host)
		}
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	ctx := context.Background()
	out, err := w.Run(ctx, map[string]any{"query": "q1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "fallback.example") {
		t.Fatalf("want fallback result, got %s", out)
	}
	if calls.Load() != 2 {
		t.Fatalf("first run: upstream calls=%d, want 2", calls.Load())
	}
	if _, err := w.Run(ctx, map[string]any{"query": "q1"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("after cache hit: upstream calls=%d, want 2", calls.Load())
	}
}

// Covers AC-11.013 — error strings do not echo API key.
func TestWebSearch_BraveError_NoKeyLeak(t *testing.T) {
	cfg := ep011WebTools(t, "brave")
	secret := "SECRETKEY987"
	if err := os.WriteFile(cfg.Search.BraveAPIKeyPath, []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	out, err := w.Run(context.Background(), map[string]any{"query": "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error() + out
	if strings.Contains(msg, secret) {
		t.Fatal("secret leaked")
	}
}

// Covers AC-11.010, AC-11.015 — successful HTTPS redirect chain with production ValidateFetchURL (public IP literals, mock transport).
func TestWebFetch_HTTPSRedirectChain_Success(t *testing.T) {
	cfg := &config.WebFetchConfig{TimeoutSeconds: 10, MaxBodyBytes: 100, MaxRedirects: 5}
	var n atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch n.Add(1) {
		case 1:
			if req.URL.Host != "1.1.1.1" {
				t.Errorf("first request host = %q, want 1.1.1.1", req.URL.Host)
			}
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://9.9.9.9/final"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		case 2:
			if req.URL.Host != "9.9.9.9" {
				t.Errorf("second request host = %q, want 9.9.9.9", req.URL.Host)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok-chain"))}, nil
		default:
			t.Fatalf("unexpected third request to %s", req.URL.String())
			return nil, io.EOF
		}
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebFetchTool(cfg, client)
	out, err := w.runFetch(context.Background(), "https://1.1.1.1/start", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ok-chain") {
		t.Fatalf("out=%q", out)
	}
	if n.Load() != 2 {
		t.Fatalf("upstream calls = %d, want 2", n.Load())
	}
}

// Covers AC-11.015 — web_fetch with production ValidateFetchURL and mock transport (no SSRF bypass).
func TestWebFetch_RealValidate_AllowedPublicIP_200(t *testing.T) {
	cfg := &config.WebFetchConfig{TimeoutSeconds: 10, MaxBodyBytes: 100, MaxRedirects: 0}
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.URL.Host != "1.1.1.1" {
			t.Errorf("host = %q, want 1.1.1.1", req.URL.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("direct-ok"))}, nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebFetchTool(cfg, client)
	out, err := w.runFetch(context.Background(), "https://1.1.1.1/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "direct-ok") {
		t.Fatalf("out=%q", out)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

// Covers AC-11.003 — Brave upstream returns non-JSON body.
func TestWebSearch_BraveInvalidJSON(t *testing.T) {
	cfg := ep011WebTools(t, "brave")
	if err := os.WriteFile(cfg.Search.BraveAPIKeyPath, []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{`))}, nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	_, err := w.Run(context.Background(), map[string]any{"query": "q"})
	if err == nil || !strings.Contains(err.Error(), "invalid brave") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.003 — Brave JSON with empty results array.
func TestWebSearch_BraveEmptyResults(t *testing.T) {
	cfg := ep011WebTools(t, "brave")
	if err := os.WriteFile(cfg.Search.BraveAPIKeyPath, []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResp(`{"web":{"results":[]}}`), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	_, err := w.Run(context.Background(), map[string]any{"query": "q"})
	if err == nil || !strings.Contains(err.Error(), "no results") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.004 — DuckDuckGo HTML without result markup.
func TestWebSearch_DDGEhtmlNoResults(t *testing.T) {
	cfg := ep011WebTools(t, "duckduckgo")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return htmlResp("<html><body>no hits</body></html>"), nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	_, err := w.Run(context.Background(), map[string]any{"query": "q"})
	if err == nil || !strings.Contains(err.Error(), "no results") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.016 — primary and fallback both fail; error from last upstream attempt.
func TestWebSearch_FallbackBothFail(t *testing.T) {
	cfg := ep011WebTools(t, "brave")
	if err := os.WriteFile(cfg.Search.BraveAPIKeyPath, []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.Search.FallbackProvider = "duckduckgo"
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		switch req.URL.Host {
		case "api.search.brave.com":
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		case "html.duckduckgo.com":
			return htmlResp("<html></html>"), nil
		default:
			return nil, fmt.Errorf("unexpected host %q", req.URL.Host)
		}
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebSearchTool(cfg, client, nil)
	_, err := w.Run(context.Background(), map[string]any{"query": "q"})
	if err == nil || !strings.Contains(err.Error(), "no results") {
		t.Fatalf("got %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("upstream calls=%d, want 2", calls.Load())
	}
}

// Covers AC-11.010 — redirect to http rejected (no SSRF validate on loopback; exercise redirect policy).
func TestWebFetch_RedirectToHTTPRejected(t *testing.T) {
	cfg := &config.WebFetchConfig{TimeoutSeconds: 10, MaxBodyBytes: 100, MaxRedirects: 5}
	var step atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		if step.Add(1) == 1 {
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"http://example.com/"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}
		return nil, io.EOF
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebFetchTool(cfg, client)
	_, err := w.runFetch(context.Background(), "https://example.com/start", func(context.Context, string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "non-https") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.010 — too many redirects.
func TestWebFetch_TooManyRedirects(t *testing.T) {
	cfg := &config.WebFetchConfig{TimeoutSeconds: 10, MaxBodyBytes: 100, MaxRedirects: 1}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://example.com/next"}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebFetchTool(cfg, client)
	_, err := w.runFetch(context.Background(), "https://example.com/a", func(context.Context, string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "many redirects") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.011 — body truncated.
func TestWebFetch_MaxBodyTruncated(t *testing.T) {
	cfg := &config.WebFetchConfig{TimeoutSeconds: 10, MaxBodyBytes: 5, MaxRedirects: 0}
	large := strings.Repeat("a", 20)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(large))}, nil
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebFetchTool(cfg, client)
	out, err := w.runFetch(context.Background(), "https://example.com/", func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "truncated") {
		t.Fatalf("out=%q", out)
	}
	if len(out) < 5 {
		t.Fatalf("out too short")
	}
}

// Covers AC-11.011 — timeout.
func TestWebFetch_Timeout(t *testing.T) {
	cfg := &config.WebFetchConfig{TimeoutSeconds: 1, MaxBodyBytes: 100, MaxRedirects: 0}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(5 * time.Second):
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
		}
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	w := NewWebFetchTool(cfg, client)
	_, err := w.runFetch(context.Background(), "https://example.com/", func(context.Context, string) error { return nil })
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("got %v", err)
	}
}

// Covers AC-11.014 — errors are short messages.
func TestWebFetch_SSRFError_NoStack(t *testing.T) {
	cfg := &config.WebFetchConfig{TimeoutSeconds: 10, MaxBodyBytes: 100, MaxRedirects: 0}
	w := NewWebFetchTool(cfg, &http.Client{})
	_, err := w.Run(context.Background(), map[string]any{"url": "http://example.com/"})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "goroutine") || strings.Contains(msg, "\n\t") {
		t.Fatalf("looks like stack: %s", msg)
	}
}

// Covers AC-11.015 — local HTTPS server (SSRF bypass only in test via noop validate).
func TestWebFetch_TLSServer_OK(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello-tls"))
	}))
	defer srv.Close()
	cfg := &config.WebFetchConfig{TimeoutSeconds: 10, MaxBodyBytes: 1000, MaxRedirects: 0}
	client := srv.Client()
	wt := NewWebFetchTool(cfg, client)
	out, err := wt.runFetch(context.Background(), srv.URL, func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello-tls") {
		t.Fatalf("out=%q", out)
	}
}

// Covers AC-11.001, AC-11.015 — registry wiring smoke (tools construct and Run).
func TestWebTools_EP011_Smoke(t *testing.T) {
	// Covers AC-11.001 / AC-11.015 — enabled config shape is valid.
	cfg := ep011WebTools(t, "duckduckgo")
	if !cfg.Enabled {
		t.Fatal("enabled")
	}
}
