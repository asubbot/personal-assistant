# EP-011 Native web search and HTTPS content fetch — Implementation plan

**Pipeline:** Stage 7 — see `ai-sdlc/specification/pipeline.spec.md` in the repo (not under `ai-sdlc-artefacts/`; no link per implementation-planning skill).  
**Test strategy:** [../../strategy.md](../../strategy.md)

**Related artefacts**

| Document            | Path                                                   |
| ------------------- | ------------------------------------------------------ |
| Scope               | [ep-scope.md](ep-scope.md)                             |
| Requirements        | [ep-requirements.md](ep-requirements.md)               |
| Acceptance criteria | [ep-acceptance-criteria.md](ep-acceptance-criteria.md) |
| System design       | [ep-system-design.md](ep-system-design.md)             |

**Purpose:** Ordered, verifiable coding tasks for EP-011. Each task lists dependencies, per-task verification, and traceability to requirements and acceptance criteria. **Product code changes** require explicit user approval (see `AGENTS.md` at repo root) before implementation.

**Contents**

- [Checkpoints](#checkpoints)
- [1. Configuration: structs, validation, ResolvePaths](#1-configuration-structs-validation-resolvepaths)
- [2. Package `internal/httpsafety` (SSRF and URL policy)](#2-package-internalhttpsafety-ssrf-and-url-policy)
- [3. In-memory search result cache (LRU, TTL)](#3-in-memory-search-result-cache-lru-ttl)
- [4. `web_search`: Brave Search and DuckDuckGo adapters](#4-web_search-brave-search-and-duckduckgo-adapters)
- [5. `web_fetch`: HTTPS GET, redirect loop, limits](#5-web_fetch-https-get-redirect-loop-limits)
- [6. Registration and wiring (`cmd/pa/main.go` and registry)](#6-registration-and-wiring-cmdpamaingo-and-registry)
- [7. Automated tests (unit, integration, `httptest` TLS)](#7-automated-tests-unit-integration-httptest-tls)
- [8. Acceptance-criteria trace comments and quality gates](#8-acceptance-criteria-trace-comments-and-quality-gates)

---

## Checkpoints

- **CP-A:** After §1, run targeted tests for config packages touched (e.g. `go test ./internal/config/...` or the concrete package path used). Confirm invalid `web_tools` rejects load and resolved paths are absolute where required — [AC-11.001](ep-acceptance-criteria.md#ac-11-001), [REQ-11.019](ep-requirements.md#config--limits), [REQ-11.020](ep-requirements.md#config--limits).
- **CP-B:** After §2–§5, run `make check`; fix fmt, vet, lint, and test failures before further integration — [REQ-11.024](ep-requirements.md#testing), [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
- **CP-C:** After §7–§8, run `./bin/validate EP-011` and confirm every AC has declared test coverage per project validation rules — [AC-11.001](ep-acceptance-criteria.md#ac-11-001)–[AC-11.016](ep-acceptance-criteria.md#ac-11-016), [REQ-11.024](ep-requirements.md#testing).
- **CP-D:** User approval before starting **product code** work if not already granted for EP-011 (`AGENTS.md`).

---

## 1. Configuration: structs, validation, ResolvePaths

**Depends on:** None (foundation). **Blocks:** §3–§6.

- **1.1** Add configuration structs for `web_tools` (or equivalent nested block) aligned with [ep-system-design.md](ep-system-design.md#configuration-keys): `enabled`, **primary search provider** (`brave` / `duckduckgo`), optional **fallback search provider** (same enum, must differ from primary when set), Brave API key filesystem path (required when either role uses Brave), cache TTL, cache max entries, `web_fetch` max body bytes, max redirect hops, `web_search` upstream timeout, `web_fetch` overall timeout, and any tool-registration flags needed alongside existing tools — [REQ-11.019](ep-requirements.md#config--limits), [REQ-11.025](ep-requirements.md#tools--providers), [REQ-11.001](ep-requirements.md#tools--providers), [REQ-11.002](ep-requirements.md#tools--providers).
  - **Verification:** JSON (or project config format) unmarshal round-trip tests; reject unknown provider enum; reject **fallback search provider** equal to **primary**; reject zero or negative numeric bounds where [AC-11.001](ep-acceptance-criteria.md#ac-11-001) requires startup failure — [REQ-11.020](ep-requirements.md#config--limits), [AC-11.016](ep-acceptance-criteria.md#ac-11-016).
  - *Requirements:* [REQ-11.019](ep-requirements.md#config--limits), [REQ-11.020](ep-requirements.md#config--limits)
  - *Acceptance criteria:* [AC-11.001](ep-acceptance-criteria.md#ac-11-001)

- **1.2** Implement validation at load time: when `web_tools.enabled` is true, all required fields for the selected provider and shared limits are present and semantically valid (paths non-empty where required, durations positive, integers strictly positive for caps) — [REQ-11.020](ep-requirements.md#config--limits), [AC-11.001](ep-acceptance-criteria.md#ac-11-001).
  - **Verification:** Table-driven tests: valid minimal config passes; each missing/invalid field fails with a clear configuration error — [AC-11.001](ep-acceptance-criteria.md#ac-11-001).
  - *Requirements:* [REQ-11.020](ep-requirements.md#config--limits)
  - *Acceptance criteria:* [AC-11.001](ep-acceptance-criteria.md#ac-11-001)

- **1.3** Add `ResolvePaths` (or extend existing config resolution) so Brave API key path and any other filesystem paths under `web_tools` resolve to **absolute** paths consistent with the rest of the product (e.g. relative to config file directory or documented base) — [REQ-11.006](ep-requirements.md#tools--providers), [ep-system-design.md](ep-system-design.md#configuration-keys).
  - **Verification:** Unit test: relative path in config becomes absolute; missing parent behaviour documented and tested — [REQ-11.006](ep-requirements.md#tools--providers).
  - *Requirements:* [REQ-11.006](ep-requirements.md#tools--providers), [REQ-11.019](ep-requirements.md#config--limits)
  - *Acceptance criteria:* [AC-11.003](ep-acceptance-criteria.md#ac-11-003), [AC-11.005](ep-acceptance-criteria.md#ac-11-005)

---

## 2. Package `internal/httpsafety` (SSRF and URL policy)

**Depends on:** §1 (limits and policy constants may live in config; pure classification here). **Blocks:** §5.

- **2.1** Create package `internal/httpsafety` with URL parsing helpers: enforce **HTTPS-only** input for `web_fetch` callers (reject non-`https` schemes before any network I/O) — [REQ-11.013](ep-requirements.md#web_fetch--ssrf), [ep-system-design.md](ep-system-design.md#ssrf-and-url-policy-web_fetch).
  - **Verification:** Unit tests: `http://`, `file://`, empty, malformed URLs return classified errors; `https://` passes scheme gate — [AC-11.008](ep-acceptance-criteria.md#ac-11-008).
  - *Requirements:* [REQ-11.013](ep-requirements.md#web_fetch--ssrf)
  - *Acceptance criteria:* [AC-11.008](ep-acceptance-criteria.md#ac-11-008)

- **2.2** Implement resolved-destination checks: loopback IPv4/IPv6, RFC 1918 private IPv4, IPv6 ULA, IPv6 link-local, IPv4 link-local, and metadata hostnames/IPs per [ep-system-design.md](ep-system-design.md#ssrf-and-url-policy-web_fetch) — [REQ-11.014](ep-requirements.md#web_fetch--ssrf).
  - **Verification:** Unit tests table: `127.0.0.1`, `::1`, `10.x`, `192.168.x`, `fc00::`, `fe80::`, `169.254.x`, `169.254.169.254`, and representative metadata hostnames; assert **no TCP dial** in validation-only tests — [AC-11.009](ep-acceptance-criteria.md#ac-11-009), [AC-11.014](ep-acceptance-criteria.md#ac-11-014).
  - *Requirements:* [REQ-11.014](ep-requirements.md#web_fetch--ssrf)
  - *Acceptance criteria:* [AC-11.009](ep-acceptance-criteria.md#ac-11-009), [AC-11.014](ep-acceptance-criteria.md#ac-11-014)

- **2.3** Expose a single validation entry point (e.g. `ValidateFetchURL`) used by `web_fetch` before request and **after each redirect target** is known, so redirect targets re-run scheme + SSRF rules — [REQ-11.015](ep-requirements.md#web_fetch--ssrf).
  - **Verification:** Unit tests calling validator with synthesized redirect URLs — [AC-11.010](ep-acceptance-criteria.md#ac-11-010).
  - *Requirements:* [REQ-11.014](ep-requirements.md#web_fetch--ssrf), [REQ-11.015](ep-requirements.md#web_fetch--ssrf)
  - *Acceptance criteria:* [AC-11.010](ep-acceptance-criteria.md#ac-11-010)

---

## 3. In-memory search result cache (LRU, TTL)

**Depends on:** §1 (TTL and max entry config). **Blocks:** §4.

- **3.1** Implement cache structure per [ep-system-design.md](ep-system-design.md#search-result-cache-entry): entries store results, `stored_at`, `last_access_at`; **eviction policy** is **LRU** when inserting would exceed max entries — [REQ-11.009](ep-requirements.md#search-cache), [REQ-11.012](ep-requirements.md#search-cache).
  - **Verification:** Unit tests: insert until cap+1 evicts LRU; access updates LRU order — [AC-11.006](ep-acceptance-criteria.md#ac-11-006).
  - *Requirements:* [REQ-11.009](ep-requirements.md#search-cache), [REQ-11.012](ep-requirements.md#search-cache)
  - *Acceptance criteria:* [AC-11.006](ep-acceptance-criteria.md#ac-11-006)

- **3.2** Implement **cache lookup key** from normalized query (trim, whitespace collapse, case fold per implementation spec in design), active provider identifier (`brave` / `duckduckgo`), and optional result-limit facet if the tool exposes it — [REQ-11.010](ep-requirements.md#search-cache), [ep-system-design.md](ep-system-design.md#search-result-cache-entry).
  - **Verification:** Unit tests: same normalization → same key; different normalization or provider → distinct keys — [AC-11.007](ep-acceptance-criteria.md#ac-11-007).
  - *Requirements:* [REQ-11.010](ep-requirements.md#search-cache)
  - *Acceptance criteria:* [AC-11.007](ep-acceptance-criteria.md#ac-11-007)

- **3.3** On cache hit with age &lt; TTL, return cached results without **upstream search request**; on miss or expiry, delegate to provider adapter — [REQ-11.011](ep-requirements.md#search-cache).
  - **Verification:** Integration-style test with mock HTTP transport counting requests: two calls, one upstream; advance clock or shorten TTL for expiry path — [AC-11.006](ep-acceptance-criteria.md#ac-11-006), [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - *Requirements:* [REQ-11.011](ep-requirements.md#search-cache)
  - *Acceptance criteria:* [AC-11.006](ep-acceptance-criteria.md#ac-11-006), [AC-11.015](ep-acceptance-criteria.md#ac-11-015)

---

## 4. `web_search`: Brave Search and DuckDuckGo adapters

**Depends on:** §1, §3. **Blocks:** §6, §7.

- **4.1** Define a small provider interface (search method + timeout from config) and shared **HTTP client** usage with per-upstream timeout — [REQ-11.018](ep-requirements.md#config--limits), [ep-system-design.md](ep-system-design.md#components-and-interfaces).
  - **Verification:** Compile-time interface satisfaction; mock server test for timeout — [AC-11.012](ep-acceptance-criteria.md#ac-11-012).
  - *Requirements:* [REQ-11.018](ep-requirements.md#config--limits)
  - *Acceptance criteria:* [AC-11.012](ep-acceptance-criteria.md#ac-11-012)

- **4.2** **Brave Search** adapter: read API key from resolved filesystem path when provider is Brave; set header or query per Brave API; parse JSON into ranked items (title, URL, snippet) — [REQ-11.004](ep-requirements.md#tools--providers), [REQ-11.006](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers).
  - **Verification:** `httptest` handler returns minimal valid Brave-style JSON; tool result contains non-empty title, URL, snippet — [AC-11.003](ep-acceptance-criteria.md#ac-11-003).
  - *Requirements:* [REQ-11.004](ep-requirements.md#tools--providers), [REQ-11.006](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers)
  - *Acceptance criteria:* [AC-11.003](ep-acceptance-criteria.md#ac-11-003)

- **4.3** **DuckDuckGo** adapter: HTTPS GET to stable endpoint chosen in implementation (per [ep-system-design.md](ep-system-design.md#module-boundaries)); parse HTML (or lite JSON if used) into same ranked item shape — [REQ-11.005](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers).
  - **Verification:** `httptest` returns minimal DDG HTML fixture; assert parsed items — [AC-11.004](ep-acceptance-criteria.md#ac-11-004).
  - *Requirements:* [REQ-11.005](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers)
  - *Acceptance criteria:* [AC-11.004](ep-acceptance-criteria.md#ac-11-004)

- **4.4** **web_search** tool handler: validate query non-empty after trim — [REQ-11.003](ep-requirements.md#tools--providers); on Brave with missing/unreadable/empty key file, return **structured error** (no key bytes in message) — [REQ-11.007](ep-requirements.md#tools--providers), [REQ-11.021](ep-requirements.md#security--observability), [REQ-11.022](ep-requirements.md#security--observability).
  - **Verification:** Tests for empty query (no upstream call); missing key file returns structured error; secret substring absent from tool output — [AC-11.002](ep-acceptance-criteria.md#ac-11-002), [AC-11.005](ep-acceptance-criteria.md#ac-11-005), [AC-11.013](ep-acceptance-criteria.md#ac-11-013).
  - *Requirements:* [REQ-11.003](ep-requirements.md#tools--providers), [REQ-11.007](ep-requirements.md#tools--providers), [REQ-11.021](ep-requirements.md#security--observability), [REQ-11.022](ep-requirements.md#security--observability)
  - *Acceptance criteria:* [AC-11.002](ep-acceptance-criteria.md#ac-11-002), [AC-11.005](ep-acceptance-criteria.md#ac-11-005), [AC-11.013](ep-acceptance-criteria.md#ac-11-013)

- **4.5** Operator logging: truncate or redact query text per project norms; never log Brave key material — [REQ-11.023](ep-requirements.md#security--observability), [REQ-11.021](ep-requirements.md#security--observability).
  - **Verification:** Tests or log capture: key file content never appears in log buffer; query length bounded — [AC-11.013](ep-acceptance-criteria.md#ac-11-013).
  - *Requirements:* [REQ-11.021](ep-requirements.md#security--observability), [REQ-11.023](ep-requirements.md#security--observability)
  - *Acceptance criteria:* [AC-11.013](ep-acceptance-criteria.md#ac-11-013)

- **4.6** **Primary** / **fallback** chain: after cache miss, run **primary search provider** **upstream search request** with per-request timeout; on any failure outcome listed in [ep-system-design.md](ep-system-design.md#error-handling), if **fallback search provider** is configured, run one **upstream search request** for the **fallback search provider**; include the ordered provider chain in the **cache lookup key** — [REQ-11.025](ep-requirements.md#tools--providers), [REQ-11.026](ep-requirements.md#tools--providers), [REQ-11.010](ep-requirements.md#search-cache), [REQ-11.018](ep-requirements.md#config--limits).
  - **Verification:** Mock transport counts two hosts in order when primary fails and fallback succeeds; cached second call does not re-hit upstream — [AC-11.016](ep-acceptance-criteria.md#ac-11-016), [AC-11.006](ep-acceptance-criteria.md#ac-11-006), [AC-11.007](ep-acceptance-criteria.md#ac-11-007).
  - *Requirements:* [REQ-11.025](ep-requirements.md#tools--providers), [REQ-11.026](ep-requirements.md#tools--providers), [REQ-11.010](ep-requirements.md#search-cache)
  - *Acceptance criteria:* [AC-11.016](ep-acceptance-criteria.md#ac-11-016)

---

## 5. `web_fetch`: HTTPS GET, redirect loop, limits

**Depends on:** §1, §2. **Blocks:** §6, §7.

- **5.1** Implement manual or controlled redirect loop (do **not** rely on default `http.Client` unlimited redirects): for each hop, parse `Location`, require `https`, run `httpsafety` validation on new host/IP, increment hop counter against configured max — [REQ-11.015](ep-requirements.md#web_fetch--ssrf), [REQ-11.017](ep-requirements.md#config--limits).
  - **Verification:** Integration tests: `Location: http://...` rejected; chain longer than max rejected; valid short HTTPS chain succeeds — [AC-11.010](ep-acceptance-criteria.md#ac-11-010).
  - *Requirements:* [REQ-11.015](ep-requirements.md#web_fetch--ssrf), [REQ-11.017](ep-requirements.md#config--limits)
  - *Acceptance criteria:* [AC-11.010](ep-acceptance-criteria.md#ac-11-010)

- **5.2** Enforce overall **wall-clock timeout** covering DNS, TLS, redirects, and body read — [REQ-11.017](ep-requirements.md#config--limits).
  - **Verification:** `httptest` TLS server with artificial delay exceeds timeout; tool returns timeout-class structured error — [AC-11.011](ep-acceptance-criteria.md#ac-11-011).
  - *Requirements:* [REQ-11.017](ep-requirements.md#config--limits)
  - *Acceptance criteria:* [AC-11.011](ep-acceptance-criteria.md#ac-11-011)

- **5.3** Read response body with hard cap: stop after configured max bytes for stored result — [REQ-11.016](ep-requirements.md#web_fetch--ssrf).
  - **Verification:** Handler streams body larger than cap; result length ≤ cap — [AC-11.011](ep-acceptance-criteria.md#ac-11-011).
  - *Requirements:* [REQ-11.016](ep-requirements.md#web_fetch--ssrf)
  - *Acceptance criteria:* [AC-11.011](ep-acceptance-criteria.md#ac-11-011)

- **5.4** Surface **structured errors** for validation, SSRF, redirect, timeout, and upstream failures; no stack traces in LLM-visible tool strings — [REQ-11.022](ep-requirements.md#security--observability), [AC-11.014](ep-acceptance-criteria.md#ac-11-014).
  - **Verification:** Assert error payload shape and absence of `runtime.Stack` or multiline trace in tool result — [AC-11.014](ep-acceptance-criteria.md#ac-11-014).
  - *Requirements:* [REQ-11.022](ep-requirements.md#security--observability)
  - *Acceptance criteria:* [AC-11.014](ep-acceptance-criteria.md#ac-11-014)

- **5.5** Redact or truncate **web_fetch** URL and body fragments in operator logs — [REQ-11.023](ep-requirements.md#security--observability).
  - **Verification:** Log helper tests or golden log lines — [AC-11.013](ep-acceptance-criteria.md#ac-11-013).
  - *Requirements:* [REQ-11.023](ep-requirements.md#security--observability)
  - *Acceptance criteria:* [AC-11.013](ep-acceptance-criteria.md#ac-11-013)

---

## 6. Registration and wiring (`cmd/pa/main.go` and registry)

**Depends on:** §1, §4, §5. **Blocks:** §7–§8.

- **6.1** Register **web_search** and **web_fetch** as **native tools** when `web_tools.enabled` is true, using the same registry pattern as existing tools — [REQ-11.001](ep-requirements.md#tools--providers), [REQ-11.002](ep-requirements.md#tools--providers).
  - **Verification:** Startup with valid config: tools appear in registry index; `go build ./cmd/pa` — [AC-11.001](ep-acceptance-criteria.md#ac-11-001).
  - *Requirements:* [REQ-11.001](ep-requirements.md#tools--providers), [REQ-11.002](ep-requirements.md#tools--providers)
  - *Acceptance criteria:* [AC-11.001](ep-acceptance-criteria.md#ac-11-001)

- **6.2** Wire dependencies in `cmd/pa/main.go` (or thin factory): pass resolved config, shared HTTP transport/settings, cache instance, and filesystem reader for Brave key — [REQ-11.019](ep-requirements.md#config--limits), [ep-system-design.md](ep-system-design.md#module-boundaries).
  - **Verification:** Binary starts with `web_tools` disabled (no registration) and enabled (both tools registered); invalid config exits non-zero — [AC-11.001](ep-acceptance-criteria.md#ac-11-001), [REQ-11.020](ep-requirements.md#config--limits).
  - *Requirements:* [REQ-11.019](ep-requirements.md#config--limits), [REQ-11.020](ep-requirements.md#config--limits)
  - *Acceptance criteria:* [AC-11.001](ep-acceptance-criteria.md#ac-11-001)

---

## 7. Automated tests (unit, integration, `httptest` TLS)

**Depends on:** §2–§6 (tests added incrementally alongside features). **Blocks:** §8.

- **7.1** **Unit tests:** `httpsafety` classification; cache key + LRU + TTL; query validation; Brave JSON mapping; DDG HTML mapping; redirect-policy helpers — [REQ-11.024](ep-requirements.md#testing), [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - **Verification:** `go test` for packages under `internal/httpsafety`, cache, and tool packages — [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - *Requirements:* [REQ-11.024](ep-requirements.md#testing)
  - *Acceptance criteria:* [AC-11.015](ep-acceptance-criteria.md#ac-11-015)

- **7.2** **Integration tests** with `net/http/httptest` **TLS** servers (`httptest.NewTLSServer` or equivalent): **web_fetch** happy path over HTTPS; redirect chains; scheme downgrade; SSRF-blocked targets without dialing disallowed addresses (use validation-only cases where integration would otherwise connect) — [REQ-11.024](ep-requirements.md#testing), [ep-system-design.md](ep-system-design.md#testing-strategy).
  - **Verification:** Tests pass with `-short` or default CI (no live Brave/DuckDuckGo) — [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - *Requirements:* [REQ-11.024](ep-requirements.md#testing)
  - *Acceptance criteria:* [AC-11.008](ep-acceptance-criteria.md#ac-11-008)–[AC-11.011](ep-acceptance-criteria.md#ac-11-011), [AC-11.015](ep-acceptance-criteria.md#ac-11-015)

- **7.3** **web_search** integration tests: Brave and DuckDuckGo adapters against `httptest` handlers (JSON/HTML fixtures); timeout test with slow handler — [REQ-11.018](ep-requirements.md#config--limits), [REQ-11.024](ep-requirements.md#testing).
  - **Verification:** Count upstream calls for cache hit/miss/expiry; timeout returns timeout-class error — [AC-11.003](ep-acceptance-criteria.md#ac-11-003), [AC-11.004](ep-acceptance-criteria.md#ac-11-004), [AC-11.006](ep-acceptance-criteria.md#ac-11-006), [AC-11.012](ep-acceptance-criteria.md#ac-11-012), [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - *Requirements:* [REQ-11.018](ep-requirements.md#config--limits), [REQ-11.024](ep-requirements.md#testing)
  - *Acceptance criteria:* [AC-11.003](ep-acceptance-criteria.md#ac-11-003), [AC-11.004](ep-acceptance-criteria.md#ac-11-004), [AC-11.006](ep-acceptance-criteria.md#ac-11-006), [AC-11.012](ep-acceptance-criteria.md#ac-11-012), [AC-11.015](ep-acceptance-criteria.md#ac-11-015)

---

## 8. Acceptance-criteria trace comments and quality gates

**Depends on:** §7 (tests must exist before final tagging). **Blocks:** epic closure.

- **8.1** Add `// Covers AC-11.NNN` (and optionally `// REQ-11.NNN`) comments on test functions or focused subtests so `./bin/validate EP-011` can map acceptance criteria to tests — follow [ai-sdlc/tools/validate/README.md](../../../ai-sdlc/tools/validate/README.md) and [VALIDATION.md](../../../ai-sdlc/tools/validate/VALIDATION.md). Cover **AC-11.001** through **AC-11.015** — [REQ-11.024](ep-requirements.md#testing), [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - **Verification:** Grep `Covers AC-11.` in test files; each AC id appears at least once — [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - *Requirements:* [REQ-11.024](ep-requirements.md#testing)
  - *Acceptance criteria:* [AC-11.001](ep-acceptance-criteria.md#ac-11-001)–[AC-11.015](ep-acceptance-criteria.md#ac-11-015)

- **8.2** Run **`make check`** from repository root (fmt, vet, lint, tests with coverage per Makefile) — [../../strategy.md](../../strategy.md), [REQ-11.024](ep-requirements.md#testing).
  - **Verification:** Exit code zero; fix all reported issues — [AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - *Requirements:* [REQ-11.024](ep-requirements.md#testing)
  - *Acceptance criteria:* [AC-11.015](ep-acceptance-criteria.md#ac-11-015)

- **8.3** Run **`./bin/validate EP-011`** and resolve any missing AC↔test linkage — [ai-sdlc/tools/validate/VALIDATION.md](../../../ai-sdlc/tools/validate/VALIDATION.md) (repo path).
  - **Verification:** Validator reports full coverage for EP-011 acceptance criteria — [AC-11.001](ep-acceptance-criteria.md#ac-11-001)–[AC-11.015](ep-acceptance-criteria.md#ac-11-015).
  - *Requirements:* [REQ-11.024](ep-requirements.md#testing)
  - *Acceptance criteria:* [AC-11.001](ep-acceptance-criteria.md#ac-11-001)–[AC-11.015](ep-acceptance-criteria.md#ac-11-015)

---

## Dependency summary (high level)

```text
§1 Config ─────────────────────────────────────────┐
     │                                              │
     ├──────────────────► §3 Cache ────────────────┼──► §4 web_search ──► §6 main.go
     │                                              │         │
     └──► §2 httpsafety ──► §5 web_fetch ──────────┘         │
              │                                                │
              └────────────────────────────────────────────────┴──► §7 Tests ──► §8 Comments + make check + validate
```

---

## Traceability matrix (quick reference)

| Area | Primary REQ sections | Primary AC ids |
|------|------------------------|----------------|
| Config / startup | [Config & limits](ep-requirements.md#config--limits), [Tools & providers](ep-requirements.md#tools--providers) | [AC-11.001](ep-acceptance-criteria.md#ac-11-001) |
| web_search behaviour | [Tools & providers](ep-requirements.md#tools--providers) | [AC-11.002](ep-acceptance-criteria.md#ac-11-002)–[AC-11.005](ep-acceptance-criteria.md#ac-11-005), [AC-11.012](ep-acceptance-criteria.md#ac-11-012) |
| Cache | [Search cache](ep-requirements.md#search-cache) | [AC-11.006](ep-acceptance-criteria.md#ac-11-006), [AC-11.007](ep-acceptance-criteria.md#ac-11-007) |
| web_fetch / SSRF | [web_fetch & SSRF](ep-requirements.md#web_fetch--ssrf), [Config & limits](ep-requirements.md#config--limits) | [AC-11.008](ep-acceptance-criteria.md#ac-11-008)–[AC-11.011](ep-acceptance-criteria.md#ac-11-011) |
| Security / errors | [Security & observability](ep-requirements.md#security--observability) | [AC-11.013](ep-acceptance-criteria.md#ac-11-013), [AC-11.014](ep-acceptance-criteria.md#ac-11-014) |
| Test suite | [Testing](ep-requirements.md#testing) | [AC-11.015](ep-acceptance-criteria.md#ac-11-015) |
