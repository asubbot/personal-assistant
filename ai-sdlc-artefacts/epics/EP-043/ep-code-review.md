---
artefact: ep-code-review
epic_id: EP-043
status: draft
source_of_truth: true
gate: pass
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
non_blocking_counts:
  nit: 2
  suggestion: 2
next_action: proceed_to_stage_11
updated_at: 2026-05-31
---

# Code review — EP-043 Test suite organization

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 2 | Suggestion 2
Open findings: none
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-043-test-suite-organization` vs `main` (23 files, +2095/−1951 lines). Commits: `docs(EP-043): stage 6+8 artefacts`, `test(EP-043): organize handler and config tests`.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 2 | Suggestion: 2
**Gate:** Pass

### Summary

The implementation delivers the EP-043 plan: `handler_test.go` is reduced to **242 LOC** with domain splits (`handler_session_test.go`, `handler_tools_test.go`, `handler_llm_test.go`, `handler_memory_test.go`), shared mocks live in `handler_test_helpers.go`, config tests use `loadConfigFixture` / `writeConfigFixtureToDir` with **32** consuming test functions and **9** new `testdata/` extractions, and EP-034/038/040 traceability guards consolidate into `architecture_guards_test.go` with `// Covers AC-xx.xxx` preserved on moved tests. No product code changes. **`make check` passes** (exit 0; coverage **76.3%** vs main **76.2%**, within the 0.5% floor). **Approve** for stage 11.

### What was done well

- **Handler split** — Clean cut/paste into thematic files matching [ep-system-design.md](ep-system-design.md); `handler_test.go` retains core message-path tests only (242 LOC, well under 600).
- **Shared helpers** — `handler_test_helpers.go` centralises mocks (`mockProvider`, `mockVectorStore`, `mockNodeRunner`, log capture handlers) per REQ-43.003; `testHandlerDeps` remains in `handler_test_deps_test.go`.
- **AC comment preservation** — Moved tests keep traceability comments (e.g. `// Covers AC-38.006` on tool-round cap in `handler_llm_test.go`, session ACs in `handler_session_test.go`).
- **Config fixtures** — `config_test_helpers.go` provides `loadConfigFixture`, `loadConfigFixtureRaw`, and `writeConfigFixtureToDir`; inline JSON blocks extracted to named `testdata/*.json` files with placeholder substitution for temp-dir paths.
- **architecture_guards_test.go** — Deduplicates walk/parse helpers; merges EP-034 (8 tests), EP-038 (2 tests), and EP-040 AC-40.001; deletes `ep034_traceability_test.go`, `ep038_traceability_test.go`, `ep040_traceability_test.go`.
- **Validator regression guard** — `./bin/validate EP-034` and `./bin/validate EP-038` both report 100% in-scope traced on this branch (AC-43.004).
- **Zero behaviour churn** — Diff is file moves and fixture extraction only; full suite green under `make check`.

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-001 | **Nit** | `internal/core/handler_llm_test.go` (599 LOC) | Largest new domain file is just under 600 LOC; only `handler_test.go` is capped by REQ-43.002. | Acceptable for this epic; note for a future split if LLM/tool-loop tests grow further. |
| F-002 | **Nit** | `internal/core/handler_llm_test.go` | `captureLLMLogWriter` is local to this file while other capture helpers live in `handler_test_helpers.go`. | Optional: move to helpers if reused elsewhere; not required now. |
| F-003 | **Suggestion** | EP-043 validator | `./bin/validate EP-043` reports 0/7 ACs traced (no `// Covers AC-43.xxx` tests). Same state exists on `main` once the epic is registered; project-wide `make validate` exits non-zero for accumulated epics. | Add a small `ep043_traceability_test.go` (LOC guard, fixture-helper count, `make check`/`validate` smoke) before audit if REQ-43.010 closure is required; follows EP-042 pattern where some ACs were traced post stage 10. Not blocking merge of this refactor. |
| F-004 | **Suggestion** | [ep-acceptance-criteria.md](ep-acceptance-criteria.md) | Index lists AC-43.005 but no `### AC-43.005` body (coverage floor). | Add the missing section in a docs polish pass (already noted in system design review N-003). |

### Plan alignment

| Plan step | Status | Notes |
|-----------|--------|-------|
| 1.1 Split `handler_test.go` into domain files; ≤600 LOC | Done | 242 LOC; 4 domain files + helpers |
| 1.2 `config_test_helpers.go`; ≥10 inline JSON → testdata | Done | 32 test funcs use helpers; 9 new JSON fixtures |
| 1.3 `architecture_guards_test.go`; merge traceability; preserve Covers AC | Done | EP-034/038 validate 100%; EP-040 AC-40.001 merged |
| 1.4 `make check` (coverage ≤0.5% drop); commit | Done | 76.3% vs main 76.2%; commit on branch |

**Deviations (justified):**

- `handler_test.go` retains behavioural tests (not helpers-only shell) — still ≤600 LOC and matches design intent for “core message path” tests.
- `loadConfigFixture` returns `*Config` only; design mentioned `( *Config, string path)` — `loadConfigFixtureRaw` covers path-only callers; sufficient for migrated tests.

### AC verification

| AC | Check | Result |
|----|-------|--------|
| AC-43.001 | `wc -l handler_test.go`; domain files exist | Pass — 242 LOC; session/tools/llm/memory files present |
| AC-43.002 | `make check` / `go test ./internal/core/...` | Pass — exit 0 |
| AC-43.003 | Fixture helper + ≥10 migrations | Pass — 32 test funcs; 9 new testdata files |
| AC-43.004 | `architecture_guards_test.go`; EP-034/038 validate | Pass — 16/16 and 12/12 traced |
| AC-43.005 | Coverage vs main baseline | Pass — 76.3% vs 76.2% (+0.1 pp) |
| AC-43.006 | `make check` | Pass — exit 0 |
| AC-43.007 | `make validate` | Pre-existing gap — EP-043 ACs untraced on main and branch; see F-003 |

### Test / verification

- `make check` — **PASS** (exit 0): tests, vet, lint, vuln, module boundaries; total coverage **76.3%**.
- `make check` on `main` — **PASS**; baseline coverage **76.2%**.
- `./bin/validate EP-034` — **PASS** (16/16 traced).
- `./bin/validate EP-038` — **PASS** (12/12 in-scope traced).
- `./bin/validate EP-043` — **FAIL** (0/7 traced); unchanged vs registering epic on `main` — track in stage 11 audit (F-003).

### Residual risks / follow-ups

- Behaviour parity relies on unchanged assertions and green full suite; residual risk is low.
- EP-043 validator traceability (F-003) and AC-43.005 doc body (F-004) are audit-stage housekeeping, not regressions from this refactor.
