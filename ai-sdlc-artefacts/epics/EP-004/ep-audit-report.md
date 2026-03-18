# EP-004 Structured tools and Tool-calling API — Audit Report

**Date and time:** 2026-03-16 (UTC)  
**Purpose:** Stage 9 audit — implementation vs plan, tests, coverage, quality gate, gaps/risks.  
**Pipeline:** [ai-sdlc/specification/pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)  
**Epic artefacts:** [ep-implementation-plan.md](ep-implementation-plan.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md)

---

## Summary

**Status: PASS with open scope.** Tasks 1.1–6.1 from the implementation plan are done. `make check` passes (fmt, vet, golangci-lint, tests with integration tag, module boundaries). Total statement coverage **76.8%**. Quality gate clean (0 linter issues, module boundaries OK).

**Open:** Task 6.2 (Sonos tool in catalog), optional block 7.1–7.3 (text-based tool invocation), and 8.1 (final tests/regression). AC-04.010 (Sonos) and AC-04.022–AC-04.025 (text-based) are not yet covered by automated tests; AC-04.018 (startup fail when tool index store cannot be created) and AC-04.021 (tool index build logging) are only partially or indirectly covered.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1.1 | Done | Tool catalog in config, YAML schema, parse at startup — [ep-implementation-plan](ep-implementation-plan.md) §1 |
| 2.1 | Done | Dedicated tool table in vector store (same DB as memory) |
| 2.2 | Done | Tool index build at startup (batch/background, 20 s or fallback) |
| 3.1 | Done | Tool pre-selection and fallback (top-k, bounded list) |
| 4.1 | Done | LLM provider interface extended for tools and tool_calls |
| 4.2 | Done | Tools + tool_calls for at least one provider (OpenAI-compatible) |
| 5.1 | Done | Core: pre-selection and tool list per request |
| 5.2 | Done | Core: validate tool_calls, reject invalid/unknown |
| 5.3 | Done | Core: template substitution and execution via run_on_node |
| 5.4 | Done | Core: tool-result loop and errors to chat |
| 6.1 | Done | Tool invocation logging (id, arguments, result/error) |
| 6.2 | Pending | Sonos tool in catalog; catalog example + integration/E2E |
| 7.1 | Pending | Optional: detect provider without Tool-calling API and config flag |
| 7.2 | Pending | Optional: prompt injection and parsing of assistant text |
| 7.3 | Pending | Optional: same validation/execution path; parse failure handling |
| 8.1 | Pending | Tests and regression (full coverage of new behaviour; all tests pass) |

---

## Test results and coverage

- **Command run:** `make check` (go fmt, go vet, golangci-lint with `--build-tags=integration`, go test with coverage, module-boundary check).
- **Result:** **PASS** (exit code 0).
- **Total coverage:** **76.8%** of statements (`total: (statements) 76.8%` from `go tool cover -func=coverage.out`).
- **Per-package (selected):** internal/core 13.5%, internal/toolcatalog 7.1%, internal/toolindex 7.2%, internal/llm 6.5%, internal/embedding 4.8%, internal/config 9.5%, internal/vector/sqlite 3.1%, tests/integration 18.1%.

Integration tests run with `make check` and cover tool flow (handler_test with mock provider), pre-selection, validation, execution, errors to chat, and tool invocation logging.

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-04.001](ep-acceptance-criteria.md#ac-04-001) | [REQ-04.001](ep-requirements.md#tool-catalog-and-source-of-truth), [REQ-04.002](ep-requirements.md#tool-catalog-and-source-of-truth) | ✓ | — | — | — | internal/config/config_test.go, internal/toolcatalog/catalog_test.go |
| [AC-04.002](ep-acceptance-criteria.md#ac-04-002) | [REQ-04.003](ep-requirements.md#tool-catalog-and-source-of-truth) | ✓ | — | — | — | internal/config/config_test.go, internal/toolcatalog/catalog_test.go, internal/toolcatalog/tooldefs_test.go |
| [AC-04.003](ep-acceptance-criteria.md#ac-04-003) | [REQ-04.004](ep-requirements.md#tool-calling-api), [REQ-04.005](ep-requirements.md#tool-calling-api), [REQ-04.019](ep-requirements.md#tool-index-and-pre-selection) | ✓ | ✓ | — | — | internal/llm/openai_test.go, internal/core/handler_test.go, internal/toolcatalog/tooldefs_test.go |
| [AC-04.004](ep-acceptance-criteria.md#ac-04-004) | [REQ-04.006](ep-requirements.md#tool-calling-api) | ✓ | ✓ | — | — | internal/core/handler_test.go, internal/llm/openai_test.go |
| [AC-04.005](ep-acceptance-criteria.md#ac-04-005) | [REQ-04.007](ep-requirements.md#validation-and-execution) | ✓ | — | — | — | internal/toolcatalog/validate_test.go |
| [AC-04.006](ep-acceptance-criteria.md#ac-04-006) | [REQ-04.008](ep-requirements.md#validation-and-execution) | ✓ | ✓ | — | — | internal/toolcatalog/validate_test.go, internal/core/handler_test.go |
| [AC-04.007](ep-acceptance-criteria.md#ac-04-007) | [REQ-04.009](ep-requirements.md#validation-and-execution), [REQ-04.010](ep-requirements.md#validation-and-execution) | ✓ | ✓ | — | — | internal/toolcatalog/substitute_test.go, internal/core/handler_test.go |
| [AC-04.008](ep-acceptance-criteria.md#ac-04-008) | [REQ-04.011](ep-requirements.md#errors-to-chat) | ✓ | ✓ | — | — | internal/core/handler_test.go |
| [AC-04.009](ep-acceptance-criteria.md#ac-04-009) | [REQ-04.012](ep-requirements.md#provider-interface) | ✓ | — | — | — | internal/llm/tools_test.go |
| [AC-04.010](ep-acceptance-criteria.md#ac-04-010) | [REQ-04.013](ep-requirements.md#sonos-support) | — | — | — | — | **Pending** (task 6.2) |
| [AC-04.011](ep-acceptance-criteria.md#ac-04-011) | [REQ-04.014](ep-requirements.md#nfr--security-testability-observability-consistency), [REQ-04.017](ep-requirements.md#nfr--security-testability-observability-consistency) | ✓ | ✓ | — | — | Implied by validation/execution tests; Sonos path same as other tools once 6.2 done |
| [AC-04.012](ep-acceptance-criteria.md#ac-04-012) | [REQ-04.015](ep-requirements.md#nfr--security-testability-observability-consistency) | ✓ | ✓ | — | — | make check passes; new behaviour covered by unit/integration tests |
| [AC-04.013](ep-acceptance-criteria.md#ac-04-013) | [REQ-04.016](ep-requirements.md#nfr--security-testability-observability-consistency) | ✓ | ✓ | — | — | internal/core/handler_test.go (tool invocation logging) |
| [AC-04.014](ep-acceptance-criteria.md#ac-04-014) | [REQ-04.018](ep-requirements.md#tool-index-and-pre-selection) | ✓ | ✓ | — | — | internal/toolindex/build_test.go, internal/vector/sqlite/store_test.go |
| [AC-04.015](ep-acceptance-criteria.md#ac-04-015) | [REQ-04.019](ep-requirements.md#tool-index-and-pre-selection) | ✓ | ✓ | — | — | internal/toolindex/select_test.go, internal/core/handler_test.go |
| [AC-04.016](ep-acceptance-criteria.md#ac-04-016) | [REQ-04.020](ep-requirements.md#tool-index-and-pre-selection) | ✓ | ✓ | — | — | internal/toolindex/select_test.go |
| [AC-04.017](ep-acceptance-criteria.md#ac-04-017) | [REQ-04.021](ep-requirements.md#tool-index-and-pre-selection) | ✓ | — | — | — | internal/toolindex/build_test.go, internal/embedding/openai_test.go (EmbedBatch, chunking) |
| [AC-04.018](ep-acceptance-criteria.md#ac-04-018) | [REQ-04.022](ep-requirements.md#tool-index-and-pre-selection) | ✓ | — | — | — | internal/config/config_test.go (invalid catalog/path → startup failure); store creation failure not explicitly tested |
| [AC-04.019](ep-acceptance-criteria.md#ac-04-019) | [REQ-04.023](ep-requirements.md#tool-index-and-pre-selection) | ✓ | ✓ | — | — | internal/toolindex/select_test.go (index not ready → fallback) |
| [AC-04.020](ep-acceptance-criteria.md#ac-04-020) | [REQ-04.024](ep-requirements.md#tool-index-and-pre-selection) | ✓ | — | — | — | internal/config/config_test.go (batch_size 1–1000), internal/embedding/openai_test.go (chunking by batch_size) |
| [AC-04.021](ep-acceptance-criteria.md#ac-04-021) | [REQ-04.025](ep-requirements.md#tool-index-and-pre-selection) | — | — | — | ✓ | Tool index build success/failure logging — manual or log assertion in test |
| [AC-04.022](ep-acceptance-criteria.md#ac-04-022) | [REQ-04.026](ep-requirements.md#tool-invocation-without-tool-calling-api) | — | — | — | — | **Optional** — not implemented (step 7.1) |
| [AC-04.023](ep-acceptance-criteria.md#ac-04-023) | [REQ-04.027](ep-requirements.md#tool-invocation-without-tool-calling-api), [REQ-04.028](ep-requirements.md#tool-invocation-without-tool-calling-api) | — | — | — | — | **Optional** — not implemented (step 7.2, 7.3) |
| [AC-04.024](ep-acceptance-criteria.md#ac-04-024) | [REQ-04.029](ep-requirements.md#tool-invocation-without-tool-calling-api) | — | — | — | — | **Optional** — not implemented (step 7.3) |
| [AC-04.025](ep-acceptance-criteria.md#ac-04-025) | [REQ-04.030](ep-requirements.md#tool-invocation-without-tool-calling-api) | — | — | — | — | **Optional** — not implemented (step 7.1) |

**Notes:** Unit = `*_test.go` in packages; Integration = `tests/integration/*_test.go` (build tag `integration`); E2E = none in repo; Manual = manual run or log inspection. Run `make check` for all automated checks.

---

## Quality gate

**Result: PASS.**  
`make check` runs: `go fmt ./...`, `go vet ./...`, `golangci-lint run --build-tags=integration ./...`, `go test -tags=integration -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic`, and module-boundary check. All passed; **0 issues** from the linter; module boundaries OK (no cycles, no forbidden edges).

---

## Gaps, risks, recommendations

**Gaps**

- **6.2 Sonos:** No Sonos tool defined in catalog and no test that invokes it; AC-04.010 not covered.
- **AC-04.018:** No explicit test for "tool index store cannot be created" → startup fail (partially covered by invalid catalog/path).
- **AC-04.021:** Tool index build success/failure logging (INFO/ERROR) not asserted in tests; manual check only unless a test captures logs.
- **7.x (text-based):** Optional block not implemented; AC-04.022–AC-04.025 have no tests (expected until step 7 is implemented).

**Risks**

- Without 6.2, epic does not fully satisfy Sonos support (REQ-04.013).
- Lack of automated check for tool index logging (AC-04.021) may allow regressions when changing startup behaviour.

**Recommendations**

1. **Complete 6.2:** Add at least one Sonos tool to the catalog and an integration (or E2E) test that invokes it and asserts the same validation/execution path as other tools.
2. **Before 8.1:** Consider adding a test or helper that asserts tool index build success logs INFO and failure logs ERROR with reason (e.g. by capturing log output in tests).
3. **If implementing 7.x:** Add unit tests for the text-based parser and an integration scenario: provider without tools → prompt with tool description → model output in defined format → parse → validate/execute; parse failure → no execution.
4. Re-run audit after 6.2 and 8.1 (and 7.x if implemented) to refresh coverage and matrix.
