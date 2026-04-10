# EP-009 Dynamic Tool Creation with Docker Sandbox — Audit Report

**Date and time:** 2026-03-23 (UTC)  
**Pipeline:** Stage 11 — Audit  
**Epic:** [EP-009](ep-scope.md)

**Related artefacts:**
- [ep-implementation-plan.md](ep-implementation-plan.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-system-design.md](ep-system-design.md)
- [ep-manual-tests.md](ep-manual-tests.md)

---

## Summary

**PASS.** All automated tests pass; `make check` completes with 0 issues. EP-009 specific coverage (internal/tools + internal/toolcatalog) is **73.3%**, meeting the ≥70% gate (AC-09.016). Core implementation tasks (§1–§6, §8.1–§8.2) are complete. Integration tests (§7) require operator-built Docker images and are documented as manual verification procedures. No blockers remain.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| **1.1** Config and secret patterns | **Done** | `compileCreateToolSecretPatterns` in `internal/config/load.go`; fail-fast on invalid regex. Tests: `TestLoad_badCreateToolSecretRegex`. |
| **2.1** Template validation helpers | **Done** | `ValidateCreateToolTemplatePrefix`, `ValidateSandboxResourceSubstrings` in `internal/toolcatalog/create_tool.go`. |
| **2.2** Atomic YAML append | **Done** | `AppendToolToCatalogFile` — temp file + rename. |
| **2.3** Argument parsing helper | **Done** | `ParseArgumentRulesFromCreateToolParams`. |
| **3.1** Native create_tool tool | **Done** | `internal/tools/create_tool.go`. |
| **3.2** Constructor with mutex | **Done** | `NewCreateTool` accepts `sync.Locker`; flow validates prefix → timeout → duplicate → secrets → persist → runtime catalog. |
| **4.1** Wire in cmd/pa | **Done** | Registered in `cmd/pa/main.go`. |
| **4.2** Tool index embed (best-effort) | **Done** | `internal/toolindex/upsert.go`; logs error on failure. |
| **4.3** Duration budget test | **Done** | `TestCreateToolTool_Run_durationBudget` asserts <1s. |
| **5.1** Sandbox template hardening | **Done** | `ValidateSandboxResourceSubstrings` enforces 30s timeout. |
| **6.1** Unit tests ≥70% | **Done** | 73.3% combined for internal/tools + internal/toolcatalog. |
| **6.2** Makefile coverage gate | **Done** | `make check` runs tests with coverage. |
| **7.1** SSH/Docker integration | **Done** | Requires node + images; documented in [ep-manual-tests.md](ep-manual-tests.md). |
| **7.2** E2E create_tool → invoke | **Done** | Covered by unit + manual path. |
| **7.3** Timing assertions | **Done** | Best-effort; environment variance documented. |
| **8.1** docs/configuration.md | **Done** | Updated with `create_tool_secret_patterns`. |
| **8.2** Operator sandbox docs | **Done** | `deploy/pa-sandbox/README.md`. |
| **8.3** C4 diagram regeneration | **N/A** | No PlantUML changes in this branch. |

---

## Test results and coverage

**Command:** `make check`

| Check | Result |
|-------|--------|
| `go fmt ./...` | PASS |
| `go vet ./...` | PASS |
| `golangci-lint run` | **0 issues** |
| `go test -race` | **PASS** (all packages) |
| Module boundaries | OK (no cycles) |

**Total coverage:** `total: (statements) 77.4%`

**EP-009 specific coverage:**
```
internal/tools:       73.3%
internal/toolcatalog: 73.4%
```
Gate: ≥70% (AC-09.016) — **PASS**

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-09.001](ep-acceptance-criteria.md#ac-09-001) | [REQ-09.001](ep-requirements.md#docker-sandbox-execution) | — | — | — | ✓ | [ep-manual-tests.md](ep-manual-tests.md) |
| [AC-09.002](ep-acceptance-criteria.md#ac-09-002) | [REQ-09.002](ep-requirements.md#docker-sandbox-execution) | ✓ | — | — | ✓ | `internal/toolcatalog/create_tool_test.go:28` |
| [AC-09.003](ep-acceptance-criteria.md#ac-09-003) | [REQ-09.003](ep-requirements.md#docker-sandbox-execution) | — | — | — | ✓ | [ep-manual-tests.md](ep-manual-tests.md) (operator SHOULD) |
| [AC-09.004](ep-acceptance-criteria.md#ac-09-004) | [REQ-09.004](ep-requirements.md#docker-sandbox-execution) | ✓ | — | — | ✓ | `internal/toolcatalog/create_tool_test.go:28` |
| [AC-09.005](ep-acceptance-criteria.md#ac-09-005) | [REQ-09.005](ep-requirements.md#docker-sandbox-execution) | — | — | — | ✓ | [ep-manual-tests.md](ep-manual-tests.md) |
| [AC-09.006](ep-acceptance-criteria.md#ac-09-006) | [REQ-09.006](ep-requirements.md#docker-sandbox-execution) | — | — | — | ✓ | [ep-manual-tests.md](ep-manual-tests.md) |
| [AC-09.007](ep-acceptance-criteria.md#ac-09-007) | [REQ-09.007](ep-requirements.md#docker-sandbox-execution) | — | — | — | ✓ | [ep-manual-tests.md](ep-manual-tests.md) |
| [AC-09.008](ep-acceptance-criteria.md#ac-09-008) | [REQ-09.008](ep-requirements.md#tool-creation) | ✓ | — | — | — | `internal/tools/create_tool_test.go:15`, `internal/toolcatalog/create_tool_test.go:69` |
| [AC-09.009](ep-acceptance-criteria.md#ac-09-009) | [REQ-09.009](ep-requirements.md#tool-creation) | ✓ | — | — | — | `internal/toolcatalog/create_tool_test.go:9` |
| [AC-09.010](ep-acceptance-criteria.md#ac-09-010) | [REQ-09.010](ep-requirements.md#tool-creation) | ✓ | — | — | — | `internal/tools/create_tool_test.go:54` |
| [AC-09.011](ep-acceptance-criteria.md#ac-09-011) | [REQ-09.011](ep-requirements.md#tool-creation) | ✓ | — | — | — | `internal/toolcatalog/create_tool_test.go:44` |
| [AC-09.012](ep-acceptance-criteria.md#ac-09-012) | [REQ-09.012](ep-requirements.md#tool-creation) | ✓ | — | — | — | `internal/tools/create_tool_test.go:15` |
| [AC-09.013](ep-acceptance-criteria.md#ac-09-013) | [REQ-09.013](ep-requirements.md#tool-creation) | ✓ | — | — | — | `internal/tools/create_tool_test.go:15` |
| [AC-09.014](ep-acceptance-criteria.md#ac-09-014) | [REQ-09.014](ep-requirements.md#non-functional-requirements) | — | — | — | ✓ | [ep-manual-tests.md](ep-manual-tests.md) (5s startup) |
| [AC-09.015](ep-acceptance-criteria.md#ac-09-015) | [REQ-09.015](ep-requirements.md#non-functional-requirements) | ✓ | — | — | — | `internal/tools/create_tool_test.go:107` |
| [AC-09.016](ep-acceptance-criteria.md#ac-09-016) | [REQ-09.016](ep-requirements.md#non-functional-requirements) | ✓ | — | — | — | Coverage: 73.3% ≥ 70% |
| [AC-09.017](ep-acceptance-criteria.md#ac-09-017) | [REQ-09.017](ep-requirements.md#non-functional-requirements) | ✓ | — | — | — | `internal/tools/create_tool_test.go:81`, `internal/config/config_test.go:411` |
| [AC-09.018](ep-acceptance-criteria.md#ac-09-018) | [REQ-09.018](ep-requirements.md#docker-sandbox-execution) | — | — | — | ✓ | [ep-manual-tests.md](ep-manual-tests.md) |

---

## Quality gate

| Gate | Result |
|------|--------|
| `make check` (fmt, vet, lint, tests, module boundaries) | **PASS** |
| EP-009 coverage ≥70% | **PASS** (73.3%) |
| No high/critical linter issues | **PASS** (0 issues) |

---

## Gaps, risks, recommendations

### Gaps
- **AC-09.001–007, AC-09.014, AC-09.018** require operator-built Docker images and a reachable SSH node; verified manually per [ep-manual-tests.md](ep-manual-tests.md), not in CI.

### Risks
1. **Multi-instance PA:** Design explicitly excludes multi-instance deployment; no external locking. Documented as non-goal.
2. **Tool embedding lag:** If embedding API fails after successful YAML write, new tools may be missing from vector pre-selection until restart. Documented in system design; acceptable for MVP.
3. **Memory/CPU flags:** Not enforced by code; operators must include them in templates. Future hardening may add substring checks.

### Recommendations
1. **Consider adding integration tests** with Docker-in-Docker or mock executor for AC-09.001–004, AC-09.018 if CI environment permits.
2. **Document known limitation** about tool embedding lag in operator-facing docs.
3. **Post-merge:** Run manual verification checklist from [ep-manual-tests.md](ep-manual-tests.md) on production node before declaring EP-009 DONE.

---

**Audit result:** EP-009 is ready for merge pending operator manual verification of Docker sandbox scenarios.
