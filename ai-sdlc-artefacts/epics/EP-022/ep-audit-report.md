# EP-022 — Epic audit report

**Date and time of creation:** 2026-04-17 (UTC)
**Branch:** `epic/EP-022-reliability-hardening`
**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) — Stage 11
**Epic artefacts:** [ep-scope](ep-scope.md) · [ep-requirements](ep-requirements.md) · [ep-acceptance-criteria](ep-acceptance-criteria.md) · [ep-system-design](ep-system-design.md) · [ep-system-design-review](ep-system-design-review.md) · [ep-implementation-plan](ep-implementation-plan.md) · [ep-code-review](ep-code-review.md)

## Summary

**Status: Pass.** All implementation-plan tasks are complete. `make check` passes (fmt, vet, govulncheck, golangci-lint `0 issues`, `go test -race -tags=integration ./...` green, module boundaries OK). Total statement coverage: **73.7%**. Stage 7 design review — Pass (iteration 2). Stage 10 code review — Pass (iteration 2, 0 open findings). AC traceability — 11/11 (9 automated + 2 manual: AC-22.009 docs, AC-22.011 `make check`). Epic is ready for closeout/merge.

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1.1 `sqlitepragma` package skeleton | Done | [`internal/sqlitepragma/policy.go`](../../../internal/sqlitepragma/policy.go) |
| 1.2 `BuildDSN` | Done | Table-driven tests: happy path + all rejection cases |
| 1.3 `VerifyOnOpen` | Done | Split into four helpers (gocyclo) |
| 2.1 Config types (reliability + http_timeout) | Done | `SQLiteStoreReliabilityConfig`; `HTTPTimeout` in LLM / embedding / **web_tools** |
| 2.2 Fail-fast validation | Done | `validateHTTPTimeout`, `validateSQLiteStoreReliability` — all five rejection categories |
| 2.3 Shipped `.config/config.json` | Done | Patched via `jq`; all 54 testdata configs updated |
| 3.1 Vector store on `sqlitepragma` | Done | `NewWithTable(..., sqlitepragma.Policy)` |
| 3.2 `jobs.Open` on `sqlitepragma` | Done | Removed explicit `PRAGMA foreign_keys=ON` from `initSchema` |
| 4.1 LLM constructor | Done | `parseHTTPTimeout`; silent fallback removed (M1 from review) |
| 4.2 Embedding constructor | Done | Same pattern as 4.1 |
| 4.3 `registerWebToolsIfEnabled` | Done | `*http.Client{Timeout: d}`, returns `error` (B1 from review) |
| 4.4 Composition root | Done | `cfg.VectorStoreReliability.ToPolicy()` at every call site |
| 5.1 Concurrent-write integration test | Done | `internal/reliability/concurrent_write_test.go`, atomic counters, race passes |
| 6.1 Docs: PRAGMA + single-writer | Done | [`docs/configuration.md`](../../../docs/configuration.md) §Local SQLite stores |
| 6.2 Docs: HTTP timeouts | Done | Same doc §Outbound HTTP timeouts (incl. `web_tools.http_timeout`) |
| 7.1 Checkpoint after Task 3 | Done | `go test -race ./internal/sqlitepragma/... ./internal/vector/... ./internal/jobs/...` |
| 7.2 Checkpoint after Task 4 | Done | `go build ./...` |
| 7.3 Final `make check` | Done | `0 issues`; all packages `ok` |

## Test results and coverage

- **Command:** `make check` (fmt, vet, govulncheck, golangci-lint v2.5.0, `go test -race -tags=integration ./...`, coverage, module boundaries).
- **Result:** **Pass**, exit 0.
- **Lint:** `0 issues`.
- **Vulnerabilities:** `No vulnerabilities found`.
- **Total test coverage (statements):** **73.7%**.
- **Module boundaries:** OK.

Key new EP-022 tests:

- `internal/sqlitepragma/policy_test.go` — 5 tests (AC-22.001/002/003).
- `internal/llm/openai_test.go` — 4 new `HTTPTimeout_*` tests (AC-22.004/007/008).
- `internal/embedding/openai_test.go` — 4 new `HTTPTimeout_*` tests (AC-22.005/007/008).
- `internal/config/webtools_test.go::TestValidateWebTools_httpTimeout_requiredAndPositive` (AC-22.006/008).
- `internal/reliability/concurrent_write_test.go` — race-enabled, 4 writers × 200 iterations (AC-22.010).
- `tests/integration/ep022_manual_test.go` — manual traces (AC-22.009, AC-22.011).

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-22.001](ep-acceptance-criteria.md#ac-22001) | [REQ-22.001](ep-requirements.md#req-22001), [REQ-22.002](ep-requirements.md#req-22002) | ✓ | — | — | — | `internal/sqlitepragma/policy_test.go` |
| [AC-22.002](ep-acceptance-criteria.md#ac-22002) | [REQ-22.001](ep-requirements.md#req-22001), [REQ-22.002](ep-requirements.md#req-22002) | ✓ | — | — | — | `internal/sqlitepragma/policy_test.go` |
| [AC-22.003](ep-acceptance-criteria.md#ac-22003) | [REQ-22.003](ep-requirements.md#req-22003), [REQ-22.004](ep-requirements.md#req-22004) | ✓ | — | — | — | `internal/sqlitepragma/policy_test.go` |
| [AC-22.004](ep-acceptance-criteria.md#ac-22004) | [REQ-22.006](ep-requirements.md#req-22006) | ✓ | — | — | — | `internal/llm/openai_test.go::TestNewOpenAICompatible_HTTPTimeout_AppliedToClient` |
| [AC-22.005](ep-acceptance-criteria.md#ac-22005) | [REQ-22.007](ep-requirements.md#req-22007) | ✓ | — | — | — | `internal/embedding/openai_test.go::TestNewOpenAICompatible_HTTPTimeout_AppliedToClient` |
| [AC-22.006](ep-acceptance-criteria.md#ac-22006) | [REQ-22.008](ep-requirements.md#req-22008) | ✓ | — | — | — | `internal/config/webtools_test.go::TestValidateWebTools_httpTimeout_requiredAndPositive` |
| [AC-22.007](ep-acceptance-criteria.md#ac-22007) | [REQ-22.009](ep-requirements.md#req-22009) | ✓ | — | — | — | `internal/llm/openai_test.go`, `internal/embedding/openai_test.go` (`HTTPTimeout_InvalidRejected`) |
| [AC-22.008](ep-acceptance-criteria.md#ac-22008) | [REQ-22.010](ep-requirements.md#req-22010) | ✓ | — | — | — | `HTTPTimeout_ZeroRejected` + `HTTPTimeout_EmptyRejected` in llm/embedding/webtools |
| [AC-22.009](ep-acceptance-criteria.md#ac-22009) | [REQ-22.011](ep-requirements.md#req-22011), [REQ-22.012](ep-requirements.md#req-22012) | — | — | — | ✓ | `tests/integration/ep022_manual_test.go::TestManual_AC22009_*` → [`docs/configuration.md`](../../../docs/configuration.md) |
| [AC-22.010](ep-acceptance-criteria.md#ac-22010) | [REQ-22.013](ep-requirements.md#req-22013) | — | ✓ | — | — | `internal/reliability/concurrent_write_test.go::TestConcurrentWrites_NoBusyErrors` (race) |
| [AC-22.011](ep-acceptance-criteria.md#ac-22011) | [REQ-22.014](ep-requirements.md#req-22014) | — | — | — | ✓ | `tests/integration/ep022_manual_test.go::TestManual_AC22011_MakeCheckPasses` |

**Notes.** Unit = Go unit tests within the logic package. Integration = `-tags=integration` plus cross-package scenarios (e.g. reliability package). Manual = `t.Skip` placeholders anchoring documentation or operator-run commands. `./bin/validate EP-022` → `11/11 traced (100%)`, 9 automated + 2 manual.

## Quality gate

| Check | Result |
|-------|--------|
| `go fmt` | clean |
| `go vet` | clean |
| `govulncheck` | no vulnerabilities |
| `golangci-lint v2.5.0` | **0 issues** |
| `go test -race -tags=integration ./...` | all packages `ok` |
| Coverage (total statements) | **73.7%** |
| Module boundaries | OK (no cycles, no forbidden edges) |
| `./bin/validate EP-022` | 100% traced (exit 0) |
| Stage 7 design review | Pass (iteration 2) |
| Stage 10 code review | Pass (iteration 2, 0 findings) |

## Gaps, risks, recommendations

**Gaps (vs plan):** none.

**Residual risks:**

- Telegram `go-telegram/bot` long-poll HTTP client remains outside this epic (explicit non-goal per [ep-scope.md](ep-scope.md)).
- Config stores `busy_timeout` and `http_timeout` as strings; any new consumer bypassing `config.Load` must re-parse. Protected by `ToPolicy` (panic on invariant violation) and `parseHTTPTimeout` (fail-fast at constructors).
- `busy_timeout` upper bound is only a recommendation in docs, not enforced by the validator.

**Recommendations:**

1. Consider a follow-up task for the Telegram long-poll HTTP timeout if operationally important.
2. Optionally add a validator cap on `busy_timeout` (e.g. 60 s) to avoid accidental unbounded waits.
3. Keep the EP-022 pattern (explicit config + fail-fast validation + `VerifyOnOpen`) as the template for future reliability epics touching local SQLite files.
