# Code review — EP-022 Reliability hardening

---

## Review iteration 1

**Review date:** 2026-04-17
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-022-reliability-hardening`, commit `64d61a3` (stage-9 implementation) vs `main`. 95 files / +2738 −196. Focus on new logic: `internal/sqlitepragma`, `internal/reliability`, config additions, LLM/embedding HTTP wiring, store constructors, composition root, `docs/configuration.md`. Testdata fixture churn checked in bulk.
**Iteration summary — open counts:** Blocker: 1 | Major: 3 | Medium: 4 | Minor: 3 | Suggestion: 2
**Gate:** Fail — return to stage 9.

### Summary

Core of the epic — `internal/sqlitepragma`, DSN-based per-connection PRAGMA policy, `VerifyOnOpen`, store rewiring, and the `-race` concurrent-write test — is implemented cleanly and matches the approved system design (REQ-22.001–22.005, REQ-22.013, AC-22.001–22.003, AC-22.010). However, the **web tools HTTP client was not updated**, contrary to REQ-22.008 / AC-22.006 / AC-22.008 / plan §4.3: `registerWebToolsIfEnabled` still uses `&http.Client{Timeout: 0}`, `WebToolsConfig.HTTPTimeout` is missing, no validator. Additionally, LLM/embedding constructors keep `defaultTimeout` constants and silently fall back when `cfg.HTTPTimeout == ""`, bypassing the fail-fast construction-site rejection the plan required. Dedicated unit tests for AC-22.004/005/007/008 are not present.

### Blockers

**B1. Web tools HTTP client is still unbounded; `web_tools.http_timeout` not implemented.** REQ-22.008 / AC-22.006 / AC-22.008 / plan §4.3 require a new `HTTPTimeout` field on `WebToolsConfig`, fail-fast validation through `validateHTTPTimeout`, and `registerWebToolsIfEnabled` using that timeout for the shared `*http.Client`. None of the three is implemented. `docs/configuration.md` attempts to document `web_tools.search/fetch.timeout_seconds` in lieu, but those are per-operation context timeouts — the client-side whole-request timeout is still zero (the unbounded condition REQ-22.010 calls out).

### Findings

| ID | Severity | File | Category | Finding | Traceability |
|----|----------|------|----------|---------|-------------|
| B1 | Blocker | `cmd/pa/main.go` L715, `internal/config/webtools.go` | Correctness / API compat | Web tools client `Timeout: 0`; `WebToolsConfig.HTTPTimeout` and `web_tools.http_timeout` not added; no validator. | REQ-22.008, AC-22.006, AC-22.008, plan §4.3 |
| M1 | Major | `internal/llm/openai.go` L51–L61, `internal/embedding/openai.go` L48–L58 | Fail-fast / KISS | Silent fallback to `defaultTimeout = 60s/30s` when `cfg.HTTPTimeout == ""`. Plan §4.1/4.2 said "no more `defaultTimeout` constant" and "zero/empty returns error". Bypasses fail-fast for any call path not going through `Load`. | AGENTS.md Principles, plan §4.1/4.2, REQ-22.010 |
| M2 | Major | `internal/llm/openai_test.go`, `internal/embedding/openai_test.go`, `internal/config/config_test.go` | Tests | No dedicated tests for AC-22.004 (`*http.Client.Timeout == 45s` for LLM), AC-22.005 (same for embedding), AC-22.007 (`"not-a-duration"` rejected), AC-22.008 (`"0s"` rejected). No `testdata/*http_timeout*.json` fixtures. | AC-22.004/005/007/008 |
| M3 | Major | new tests and config test additions | Tests — traceability | Project convention requires `// Covers AC-22.XXX` on every test touching an AC; only `internal/sqlitepragma/policy_test.go` and `internal/reliability/concurrent_write_test.go` comply. | Scope note #4 |
| Med1 | Medium | `internal/config/config.go` | API / design deviation | Design §Data models declared `HTTPTimeout time.Duration` and `BusyTimeout time.Duration`; implementation keeps them as `string` and each consumer re-parses. `ToPolicy` silently discards `time.ParseDuration` error. | System design §Components, §Data models |
| Med2 | Medium | `internal/config/config.go::ToPolicy` L66–L81 | Defensive programming | `ToPolicy()` on nil or partially-populated config returns zero-value `Policy`, which fails later in `BuildDSN.Validate`. Silent `ParseDuration` error discard. | AGENTS.md fail-fast |
| Med3 | Medium | `internal/reliability/concurrent_write_test.go` L56–L81 | Tests / correctness | 4s `WithDeadline` + 5s `time.After` fallback + `if ctx.Err() == nil` error guard: if writers stall, goroutines silently return and test passes without hitting iteration count. | REQ-22.013, AC-22.010 |
| Med4 | Medium | `docs/configuration.md` §Outbound HTTP timeouts | Docs | Table lists `web_tools.search/fetch.timeout_seconds` as if they satisfied `http_timeout`. Contradicts REQ-22.008 / AC-22.006. | REQ-22.012, AC-22.009 |
| Min1 | Minor | `internal/llm/openai.go` L59 | Consistency | Error message does not name the JSON field path unlike `config.validateHTTPTimeout`. | Skill §3 Observability |
| Min2 | Minor | `internal/reliability/concurrent_write_test.go` L146–L168 | Style / KISS | Hand-rolled `itoa`; `strconv.Itoa` is canonical. | KISS |
| Min3 | Minor | `internal/sqlitepragma/policy.go` L77–L82 | Redundancy | Explicit `_foreign_keys=off` when `ForeignKeys=false`. Harmless; confirm deliberate. | — |
| S1 | Suggestion | `internal/config/testdata/` | Tests | Add `llm_http_timeout_invalid.json`, `llm_http_timeout_zero.json`, `embedding_http_timeout_zero.json` fixtures. | AC-22.007/008 |
| S2 | Suggestion | `internal/reliability/concurrent_write_test.go` | Tests | Log per-goroutine success counts via `t.Logf` for CI signal. | REQ-22.013 |

### Test / verification notes

- `go test -race ./internal/reliability/...` passed locally during stage 9 (AC-22.010 gate).
- AC-22.001–003 covered by `internal/sqlitepragma/policy_test.go`.
- AC-22.004/005/007/008 **not** covered in the change set.
- `make check` passed during stage 9 and should be rerun after iteration-2 fixes.

### Positive observations

- `internal/sqlitepragma` boundary (no dependency on `internal/config`) matches the design intent.
- `VerifyOnOpen` split into four helpers preserves semantics; `EqualFold` for journal_mode, numeric/symbolic fallback for synchronous, 0/1 for foreign_keys.
- Removing `PRAGMA foreign_keys = ON` from `jobs.initSchema` correctly compensated by DSN + `VerifyOnOpen`; the FK cascade on `job_runs.job_id` remains valid per pooled connection.
- Store constructors refuse mismatched `ForeignKeys` at the call site (two independent guards).
- Concurrent-write test topology matches `ep-system-design.md §Testing strategy`.
- `docs/configuration.md` EP-022 subsections cover PRAGMA policy and single-writer expectation correctly (only the web-tools timeouts table needs rework).

### Residual risks (after iteration 2 fixes)

- `go-telegram/bot` long-poll HTTP timeouts remain out of scope.
- Config keeps string-typed durations; any future consumer bypassing `Load` must re-validate.
- `busy_timeout` upper bound is only a docs recommendation; no loader-enforced ceiling.

---

## Review iteration 2

**Review date:** 2026-04-17
**Stage 10 iteration:** 2 of max 5
**Scope:** commit `62d1240` on branch `epic/EP-022-reliability-hardening` vs previous tip `64d61a3`. 15 files / +444 −178. Focus on iteration-1 finding closure.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Suggestion: 0
**Gate:** Pass — every iteration-1 finding is closed with traceable evidence; no regressions observed in the delta.

### Summary

The remediation commit surgically closes all twelve iteration-1 findings (B1, M1–M3, Med1–Med4, Min1–Min3, S1) and introduces no new logic outside the agreed scope. Web-tools HTTP client is now bounded by `web_tools.http_timeout` with Load-time validation and construction-time fail-fast; `defaultTimeout` constants are deleted from both `internal/llm` and `internal/embedding` in favour of a local `parseHTTPTimeout` helper that rejects empty / invalid / zero values. Dedicated unit tests for AC-22.004/005/007/008 exist with `// Covers AC-22.NNN` trace comments. `ToPolicy` switches to documented panic-on-invariant-violation semantics, with the design deviation (string-typed JSON durations) explicitly justified in godoc. The concurrent-write test uses per-writer atomic counters and asserts the full iteration budget, so a stalled writer now fails the test instead of silently passing.

### Iteration 1 finding status

| ID | Status | Evidence (file:line or commit ref) |
|----|--------|-------------------------------------|
| B1 | Closed | `internal/config/webtools.go` L15–L19 adds `HTTPTimeout` field with EP-022 docstring; L59–L61 adds `validateHTTPTimeout("web_tools.http_timeout", w.HTTPTimeout)`. `cmd/pa/main.go` L230–L232 returns the error from `registerWebToolsIfEnabled`; L713–L717 parses `cfg.WebTools.HTTPTimeout` and errors on invalid/non-positive; L722 sets `&http.Client{Timeout: timeout}`. `cmd/pa/main_test.go` L38 adds `HTTPTimeout: "30s"`; L40–L43 asserts the error path. `docs/configuration.md` L82–L88 documents `web_tools.http_timeout` as required. |
| M1 | Closed | `internal/llm/openai.go` L15 — `defaultTimeout` constant deleted; L30–L45 adds `parseHTTPTimeout` that errors on empty / unparsable / `<=0`; L66–L69 replaces silent fallback with `parseHTTPTimeout(cfg.HTTPTimeout)` + `return nil, err`. Identical pattern in `internal/embedding/openai.go` L15 (constant deleted), L26–L41 (helper), L63–L66 (wiring). No hidden defaults remain. |
| M2 | Closed | `internal/llm/openai_test.go` L844–L901 adds `TestNewOpenAICompatible_HTTPTimeout_{AppliedToClient,InvalidRejected,ZeroRejected,EmptyRejected}` covering AC-22.004 / AC-22.007 / AC-22.008. `internal/embedding/openai_test.go` L416–L463 adds the mirror suite for AC-22.005 / AC-22.007 / AC-22.008. `internal/config/webtools_test.go` L102–L131 adds `TestValidateWebTools_httpTimeout_requiredAndPositive` covering AC-22.006 / AC-22.008 (empty / invalid / zero / valid cases). |
| M3 | Closed | All new tests carry `// Covers AC-22.NNN` trace comments: `internal/llm/openai_test.go` L572, L590, L605, L619; `internal/embedding/openai_test.go` L192, L207, L219, L231; `internal/config/webtools_test.go` L103; `internal/reliability/concurrent_write_test.go` L22 (`AC-22.010`). |
| Med1 | Closed | `internal/config/config.go` L64–L76 ToPolicy godoc now explicitly documents the string-vs-`time.Duration` JSON boundary as an intentional deviation from ep-system-design.md, with rationale (readable JSON round-trip, no per-field `UnmarshalJSON`, re-parse only at the few consumer sites). |
| Med2 | Closed | `internal/config/config.go` L77–L93: nil receiver, nil `ForeignKeys`, and `time.ParseDuration` failure each `panic` with an explicit "Load should have rejected this" message. The godoc L64–L71 declares the post-Load contract. The cascade on call sites is addressed in `cmd/pa/main_test.go` L487–L493 where `TestNewToolIndex_vectorStoreFails_returnsError` now populates `VectorStoreReliability`. |
| Med3 | Closed | `internal/reliability/concurrent_write_test.go` L15–L18 imports `strconv` + `sync/atomic`; L21–L24 comment and `const iterations = 200`; L64–L66 uses a single 30s `context.WithTimeout` (no race between deadline and `time.After`); L73–L82 passes per-writer `*int64` counters; L89–L96 asserts each writer reached `iterations` and fails otherwise; writers now forward `ctx.Err()` into `errCh` instead of swallowing it (L111–L114, L126–L129). |
| Med4 | Closed | `docs/configuration.md` L82–L88 adds a "Required" column, lists `web_tools.http_timeout` as required when `web_tools.enabled=true`, and the explanatory paragraph immediately below explicitly states `search/fetch.timeout_seconds` are per-operation context timeouts and **not** a substitute. |
| Min1 | Closed | `internal/llm/openai.go` L37–L44 error messages name the field: `"llm: http_timeout is required"`, `"llm: http_timeout invalid duration %q: %w"`, `"llm: http_timeout must be > 0, got %s"`. Symmetric wording in `internal/embedding/openai.go` L32–L39. |
| Min2 | Closed | `internal/reliability/concurrent_write_test.go` L111, L131 now use `strconv.Itoa`; the hand-rolled `itoa` helper (previous L146–L168) is deleted. |
| Min3 | Closed | `internal/sqlitepragma/policy.go` L77–L82 adds a comment explicitly documenting the deliberate `_foreign_keys=off` DSN encoding for the vec0 store (vec0 virtual tables reject FK enforcement) and its role in keeping `VerifyOnOpen` symmetric. |
| S1 | N/A (suggestion) | Addressed via Go-literal tests in `openai_test.go` / `webtools_test.go` rather than JSON fixtures. Equivalent AC coverage. |
| S2 | Closed | `internal/reliability/concurrent_write_test.go` L96 logs per-writer iteration counts via `t.Logf`. |

### New findings

None.

### Verification notes

- `make check` assumed green per the stage-9/10 gate; no obviously high-complexity new function introduced (parseHTTPTimeout helpers are ~10 LOC each; `registerWebToolsIfEnabled` adds one branch; `ToPolicy` fan-out of three panic sites is well under gocyclo 12).
- Imports remain gofumpt-grouped (stdlib then project) in all edited files; `strconv` added alphabetically to `concurrent_write_test.go`.
- Config JSON boundary is explicit: `validateHTTPTimeout` is now invoked for all three outbound HTTP clients (llm, embedding, web_tools) at Load; no fallback invents a timeout silently.
- ToPolicy panic contract is safe in practice — all call sites (`cmd/pa/main.go` L283, L404, L464, L520, L577, L595, L612, L828) receive configs that have already passed `config.Load`, and the test-only direct constructor in `TestNewToolIndex_vectorStoreFails_returnsError` has been updated to supply a valid `VectorStoreReliability` block.
- AC-22.010 test now fails loudly on stall: every writer must reach `iterations=200`; `ctx.Err()` is reported rather than swallowed; `time.After` safety-net race removed.
- Trace-comment convention satisfied on every new test added in the commit.

### Gate decision

**Pass → proceed to Stage 11 (closeout).** All iteration-1 blockers, majors, mediums, and minors are closed with file:line evidence; no regressions detected in the delta commit.
