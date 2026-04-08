# EP-011 — Audit report

**Date and time of creation:** 2026-04-08 (UTC)  
**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 9  
**Related:** [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md)

## Summary

Implementation for native **web_search** (Brave + DuckDuckGo, **primary** + optional **fallback**), **web_fetch** (HTTPS-only with SSRF checks), in-process search cache (TTL + LRU), and configuration block `web_tools` is **complete** on branch `epic/EP-011-native-web-search-https-fetch`. **`make check`** passes; **`./bin/validate EP-011`** reports **16/16** AC traced (100%).

## Implementation vs plan

| Task area | Status | Notes |
|-----------|--------|--------|
| Config (`web_tools`, validation, ResolvePaths) | Done | `internal/config/webtools.go`, `validateWebTools`, `ResolvePaths` |
| `internal/httpsafety` SSRF / HTTPS-only | Done | `ValidateFetchURL`, tests |
| Search cache LRU + TTL | Done | `internal/tools/searchcache.go` |
| `web_search` Brave + DDG + fallback chain | Done | `internal/tools/web_search.go`, `fallback_provider` in config |
| `web_fetch` redirects + limits | Done | `internal/tools/web_fetch.go` |
| `cmd/pa` registration | Done | `registerWebToolsIfEnabled` |
| Tests + AC comments | Done | `web_tools_ep011_test.go`, `httpsafety`, `config` tests |
| Quality gates | Done | `make check`, `./bin/validate EP-011` |

## Test results and coverage

- **Command:** `make check` (fmt, vet, golangci-lint, `go test -race -tags=integration ./...`, coverage).
- **Result:** Pass (0 linter issues).
- **Total statement coverage (project-wide from `coverage.out`):** **72.8%** (aggregate across `./...` with `-coverpkg=./...`).

## REQ/AC test coverage matrix

| AC | REQ (summary) | Unit | Integration | E2E | Manual | Link |
|----|---------------|------|-------------|-----|--------|------|
| [AC-11.001](ep-acceptance-criteria.md#ac-11-001) | Config + registration | ✓ | — | — | — | `cmd/pa/main_test.go`, `internal/config/webtools_test.go`, `web_tools_ep011_test.go` |
| [AC-11.002](ep-acceptance-criteria.md#ac-11-002) | Empty query | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.003](ep-acceptance-criteria.md#ac-11-003) | Brave results | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.004](ep-acceptance-criteria.md#ac-11-004) | DDG results | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.005](ep-acceptance-criteria.md#ac-11-005) | Brave key missing | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.006](ep-acceptance-criteria.md#ac-11-006) | Cache TTL / eviction | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.007](ep-acceptance-criteria.md#ac-11-007) | Distinct cache keys | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.008](ep-acceptance-criteria.md#ac-11-008) | HTTPS-only / scheme | ✓ | — | — | — | `internal/httpsafety/ssrf_test.go` |
| [AC-11.009](ep-acceptance-criteria.md#ac-11-009) | SSRF literals / metadata | ✓ | — | — | — | `internal/httpsafety/ssrf_test.go` |
| [AC-11.010](ep-acceptance-criteria.md#ac-11-010) | Redirect policy | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.011](ep-acceptance-criteria.md#ac-11-011) | Body cap / fetch timeout | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.012](ep-acceptance-criteria.md#ac-11-012) | Search timeout | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.013](ep-acceptance-criteria.md#ac-11-013) | No secret in output | ✓ | — | — | — | `web_tools_ep011_test.go` |
| [AC-11.014](ep-acceptance-criteria.md#ac-11-014) | Structured errors | ✓ | — | — | — | `httpsafety`, `web_tools_ep011_test.go` |
| [AC-11.015](ep-acceptance-criteria.md#ac-11-015) | CI-safe tests | ✓ | ✓ | — | — | `httptest` TLS + mocked upstream |
| [AC-11.016](ep-acceptance-criteria.md#ac-11-016) | Search fallback + config validation | ✓ | — | — | — | `web_tools_ep011_test.go`, `internal/config/webtools_test.go` |

## Quality gate

- **golangci-lint:** pass  
- **govulncheck:** no vulnerabilities reported  
- **Module boundaries:** OK  

## Gaps, risks, recommendations

- **Gap:** Live Brave / DuckDuckGo behaviour is not exercised in CI; adapters are validated via `httptest` stubs and fixed HTML/JSON fixtures. **Risk:** upstream HTML/API shape drift for DuckDuckGo or Brave. **Recommendation:** optional periodic manual smoke test with real keys and monitoring of parse failures in logs.  
- **Risk:** DNS rebinding class attacks are mitigated only at validation time (standard trade-off). Documented in system design; no periodic re-validation mid-download beyond redirect checks.  
- **Recommendation:** operators enable `web_tools` only with explicit `brave_api_key_path` and review `fetch` limits for their environment.
