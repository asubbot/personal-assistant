package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pa/internal/config"
	"pa/internal/httpsafety"
	"strings"
	"time"
	"unicode/utf8"
)

// WebFetchTool implements native web_fetch (EP-011).
type WebFetchTool struct {
	cfg    *config.WebFetchConfig
	client *http.Client
}

// NewWebFetchTool builds web_fetch. cfg and client must be non-nil.
func NewWebFetchTool(cfg *config.WebFetchConfig, client *http.Client) *WebFetchTool {
	return &WebFetchTool{cfg: cfg, client: client}
}

// Name implements Tool.
func (w *WebFetchTool) Name() string { return "web_fetch" }

// Description implements Tool.
func (w *WebFetchTool) Description() string {
	return `Fetch an HTTPS URL and return the response body as UTF-8 text. Argument: url (string, https only). The body is truncated to configured max_body_bytes with a suffix if longer. SSRF mitigation blocks private networks and metadata endpoints. Prefer small, focused URLs (one document). For GitHub files or README content, use raw.githubusercontent.com or the GitHub API instead of the full HTML repository UI. Avoid fetching huge pages when a smaller URL answers the question.`
}

// ParamsSchema implements Tool.
func (w *WebFetchTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{{Name: "url", Required: true, Type: "string"}}
}

// Run implements Tool.
func (w *WebFetchTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if err := ValidateParams(w.ParamsSchema(), params); err != nil {
		return "", err
	}
	raw, _ := params["url"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("web_fetch: url is empty")
	}
	return w.runFetch(ctx, raw, nil)
}

// validateFetchURLFn is used by tests to bypass SSRF for httptest loopback (production passes nil).
type validateFetchURLFn func(ctx context.Context, rawURL string) error

func webFetchRedirectStatus(code int) bool {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func webFetchResolveRedirect(baseURL, location string) (string, error) {
	loc := strings.TrimSpace(location)
	if loc == "" {
		return "", errors.New("web_fetch: redirect without location")
	}
	next, err := url.Parse(loc)
	if err != nil {
		return "", errors.New("web_fetch: invalid redirect URL")
	}
	base, _ := url.Parse(baseURL)
	out := base.ResolveReference(next).String()
	if !strings.HasPrefix(strings.ToLower(out), "https://") {
		return "", errors.New("web_fetch: redirect to non-https URL")
	}
	return out, nil
}

func (w *WebFetchTool) doFetchGET(fetchCtx context.Context, finalURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, finalURL, http.NoBody)
	if err != nil {
		return nil, errors.New("web_fetch: invalid request")
	}
	req.Header.Set("User-Agent", "PersonalAssistant/1.0 (+https://github.com/)")
	return w.client.Do(req)
}

func (w *WebFetchTool) runFetch(ctx context.Context, raw string, validate validateFetchURLFn) (string, error) {
	if validate == nil {
		validate = func(c context.Context, u string) error {
			return httpsafety.ValidateFetchURL(c, u, nil)
		}
	}

	timeout := time.Duration(w.cfg.TimeoutSeconds) * time.Second
	fetchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	finalURL := raw
	redirectsUsed := 0
	maxRedir := w.cfg.MaxRedirects
	if maxRedir < 0 {
		maxRedir = 0
	}
	for {
		if err := validate(fetchCtx, finalURL); err != nil {
			return "", fmt.Errorf("web_fetch: %w", err)
		}
		resp, err := w.doFetchGET(fetchCtx, finalURL)
		if err != nil {
			if fetchCtx.Err() != nil {
				return "", errors.New("web_fetch: timeout")
			}
			return "", errors.New("web_fetch: request failed")
		}
		code := resp.StatusCode
		switch {
		case code >= 200 && code < 300:
			return readFetchBody(resp, w.cfg.MaxBodyBytes)
		case webFetchRedirectStatus(code):
			_ = resp.Body.Close()
			if redirectsUsed >= maxRedir {
				return "", errors.New("web_fetch: too many redirects")
			}
			nextURL, redirErr := webFetchResolveRedirect(finalURL, resp.Header.Get("Location"))
			if redirErr != nil {
				return "", redirErr
			}
			finalURL = nextURL
			redirectsUsed++
			continue
		default:
			_ = resp.Body.Close()
			return "", fmt.Errorf("web_fetch: HTTP status %d", code)
		}
	}
}

func readFetchBody(resp *http.Response, maxBytes int64) (string, error) {
	defer func() { _ = resp.Body.Close() }()
	max := maxBytes
	if max < 1 {
		max = 1
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return "", errors.New("web_fetch: read body failed")
	}
	truncated := int64(len(data)) > max
	if truncated {
		data = data[:max]
	}
	s := strings.ToValidUTF8(string(data), "\uFFFD")
	if truncated {
		s += "\n… [truncated]"
	}
	return s, nil
}

// RuneAwareTruncate trims s to at most maxRunes runes (used for logging).
func RuneAwareTruncate(s string, maxRunes int) string {
	if maxRunes <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return s
}
