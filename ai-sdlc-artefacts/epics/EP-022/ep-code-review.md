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
