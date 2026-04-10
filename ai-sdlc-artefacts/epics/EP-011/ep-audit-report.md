# EP-011 — Audit report

**Date and time of creation:** 2026-04-09 16:36 UTC  
**Last updated:** 2026-04-09 16:55 UTC — extended `web_tools` test coverage (redirect success, real `ValidateFetchURL` path, `web_search` edge cases).  
**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 11  
**Related:** [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md)

## Summary

Native **web_search** (Brave and DuckDuckGo, **primary** + optional **fallback**), **web_fetch** (HTTPS-only, SSRF checks, redirect and body limits), `web_tools` configuration, and automated tests are **implemented** and align with the implementation plan. CI tests in [`internal/tools/web_tools_ep011_test.go`](../../../internal/tools/web_tools_ep011_test.go) additionally cover: a **successful** HTTPS redirect chain with production `ValidateFetchURL` (public IP literals + mock transport), a **200** response on the same validator path without SSRF bypass, and **web_search** edge cases (invalid Brave JSON, empty Brave results, empty DuckDuckGo HTML, primary+fallback both failing). **`make check`** **passed**; **`./bin/validate EP-011`** reports **16/16** AC with automated trace comments (100%). Project-wide statement coverage (`make coverage`, `-coverpkg=./...`): **72.9%**. Epic **Status** in [ep-scope.md](ep-scope.md): **DONE** (closed at project audit 2026-04-09).

## Implementation vs plan

Reference: [ep-implementation-plan.md](ep-implementation-plan.md).

| Section | Tasks | Status | Notes |
|---------|-------|--------|--------|
| §1 Config | 1.1–1.3 | **Done** | `internal/config/webtools.go`, `ResolvePaths`, `resolve_webtools_test.go` |
| §2 httpsafety | 2.1–2.3 | **Done** | `internal/httpsafety/ssrf.go`, tests |
| §3 Cache | 3.1–3.3 | **Done** | `internal/tools/searchcache.go` |
| §4 web_search | 4.1–4.6 | **Done** | `internal/tools/web_search.go`; per-attempt timeout; no separate named Go `interface` for providers (logic via `searchByProvider` — matches KISS) |
| §5 web_fetch | 5.1–5.5 | **Done** | `internal/tools/web_fetch.go` |
| §6 Registration | 6.1–6.2 | **Done** | `cmd/pa/main.go` `registerWebToolsIfEnabled` |
| §7 Tests | 7.1–7.3 | **Done** | `web_tools_ep011_test.go` (incl. redirect success, real `ValidateFetchURL`, search error paths), `httpsafety`, config tests, `httptest` TLS |
| §8 Trace / gates | 8.1–8.3 | **Done** | `Covers AC-11.*` comments; plan text still says AC-11.001–015 in §8.1/8.3 — **AC-11.016** included in validator and tests |

## Test results and coverage

- **Command:** `make check` (fmt, vet, govulncheck, golangci-lint, `go test -race -tags=integration ./...`, coverage, module boundaries).
- **Result:** **Pass** (0 linter issues).
- **AC validation:** `./bin/validate EP-011` — exit 0; **16/16** in-scope AC traced.
- **Total statement coverage (project-wide, `coverage.out`, `-coverpkg=./...`):** **72.9%** (`total: (statements) 72.9%` from `make coverage`, 2026-04-09).

## REQ/AC test coverage matrix

**Notes:** In this repo, **Unit** denotes package tests under `internal/` and `cmd/`; **Integration** denotes tests using `httptest`, TLS test servers, or multi-component wiring without a separate E2E harness ([strategy.md](../../strategy.md)). Extra scenarios for **AC-11.003**, **AC-11.004**, **AC-11.010**, **AC-11.015**, **AC-11.016** live in `TestWebSearch_BraveInvalidJSON`, `TestWebSearch_BraveEmptyResults`, `TestWebSearch_DDGEhtmlNoResults`, `TestWebSearch_FallbackBothFail`, `TestWebFetch_HTTPSRedirectChain_Success`, and `TestWebFetch_RealValidate_AllowedPublicIP_200` in `web_tools_ep011_test.go`.

| AC | REQ (trace) | Unit | Integration | E2E | Manual | Link |
|----|-------------|------|---------------|-----|--------|------|
| [AC-11.001](ep-acceptance-criteria.md#ac-11-001) | [REQ-11.001](ep-requirements.md#tools--providers), [REQ-11.002](ep-requirements.md#tools--providers), [REQ-11.019](ep-requirements.md#config--limits), [REQ-11.020](ep-requirements.md#config--limits) | ✓ | — | — | — | `cmd/pa/main_test.go`, `internal/config/webtools_test.go`, `internal/tools/web_tools_ep011_test.go` |
| [AC-11.002](ep-acceptance-criteria.md#ac-11-002) | [REQ-11.003](ep-requirements.md#tools--providers) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.003](ep-acceptance-criteria.md#ac-11-003) | [REQ-11.004](ep-requirements.md#tools--providers), [REQ-11.006](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go`, `internal/config/resolve_webtools_test.go` |
| [AC-11.004](ep-acceptance-criteria.md#ac-11-004) | [REQ-11.005](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.005](ep-acceptance-criteria.md#ac-11-005) | [REQ-11.007](ep-requirements.md#tools--providers) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.006](ep-acceptance-criteria.md#ac-11-006) | [REQ-11.009](ep-requirements.md#search-cache)–[REQ-11.012](ep-requirements.md#search-cache) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.007](ep-acceptance-criteria.md#ac-11-007) | [REQ-11.010](ep-requirements.md#search-cache) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.008](ep-acceptance-criteria.md#ac-11-008) | [REQ-11.013](ep-requirements.md#web_fetch--ssrf) | ✓ | — | — | — | `internal/httpsafety/ssrf_test.go` |
| [AC-11.009](ep-acceptance-criteria.md#ac-11-009) | [REQ-11.014](ep-requirements.md#web_fetch--ssrf) | ✓ | — | — | — | `internal/httpsafety/ssrf_test.go` |
| [AC-11.010](ep-acceptance-criteria.md#ac-11-010) | [REQ-11.015](ep-requirements.md#web_fetch--ssrf) | ✓ | ✓ | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.011](ep-acceptance-criteria.md#ac-11-011) | [REQ-11.016](ep-requirements.md#web_fetch--ssrf), [REQ-11.017](ep-requirements.md#config--limits) | ✓ | ✓ | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.012](ep-acceptance-criteria.md#ac-11-012) | [REQ-11.018](ep-requirements.md#config--limits) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.013](ep-acceptance-criteria.md#ac-11-013) | [REQ-11.021](ep-requirements.md#security--observability)–[REQ-11.023](ep-requirements.md#security--observability) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go` |
| [AC-11.014](ep-acceptance-criteria.md#ac-11-014) | [REQ-11.022](ep-requirements.md#security--observability) | ✓ | — | — | — | `internal/httpsafety/ssrf_test.go`, `internal/tools/web_tools_ep011_test.go` |
| [AC-11.015](ep-acceptance-criteria.md#ac-11-015) | [REQ-11.024](ep-requirements.md#testing) | ✓ | ✓ | — | — | `internal/tools/web_tools_ep011_test.go`, `internal/httpsafety/ssrf_test.go` |
| [AC-11.016](ep-acceptance-criteria.md#ac-11-016) | [REQ-11.025](ep-requirements.md#tools--providers), [REQ-11.026](ep-requirements.md#tools--providers), [REQ-11.024](ep-requirements.md#testing) | ✓ | — | — | — | `internal/tools/web_tools_ep011_test.go`, `internal/config/webtools_test.go` |

## Quality gate

- **golangci-lint:** pass  
- **govulncheck:** no vulnerabilities reported  
- **Module boundaries:** OK  
- **`./bin/validate EP-011`:** all AC covered  

## Gaps, risks, recommendations

- **Gap (unchanged):** Live Brave / DuckDuckGo responses are not exercised in CI; adapters use `httptest` fixtures. **Risk:** HTML/API shape drift. **Recommendation:** periodic manual smoke with real credentials; monitor parse/upstream errors in ops logs.  
- **Mitigated in CI (2026-04-09):** Previously weak spots — **successful** HTTPS redirect chain, **`runFetch` with production `ValidateFetchURL`** (no test-only SSRF bypass), and **`web_search`** failure branches (invalid JSON, empty results, empty DDG HTML, both providers failing) — are now covered by automated tests in [`internal/tools/web_tools_ep011_test.go`](../../../internal/tools/web_tools_ep011_test.go).  
- **Gap:** [ep-implementation-plan.md](ep-implementation-plan.md) §8.1/8.3 still mention AC through **AC-11.015** only; **AC-11.016** is implemented and validated — **Recommendation:** align plan wording on next doc pass.  
- **Risk:** DNS/SSRF trade-offs documented in [ep-system-design.md](ep-system-design.md); no mid-body re-validation beyond redirect policy.  
- **Recommendation:** Optional operator smoke test of live `web_tools` (Brave / DuckDuckGo) in the target environment; not required for automated quality gate.
