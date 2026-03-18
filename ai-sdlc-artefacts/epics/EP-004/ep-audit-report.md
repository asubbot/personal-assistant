# EP-004 Structured tools and Tool-calling API — Audit Report

**Date and time:** 2026-03-18 (UTC)  
**Purpose:** Stage 9 audit — implementation vs plan, tests, coverage, quality gate, gaps/risks.  
**Pipeline:** [ai-sdlc/specification/pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)  
**Epic artefacts:** [ep-implementation-plan.md](ep-implementation-plan.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md), [ep-manual-tests.md](ep-manual-tests.md)

---

## Summary

**Status: PASS.** Plan tasks **1.1–8.1** are done ([ep-implementation-plan.md](ep-implementation-plan.md)). Epic **Status: DONE** in [ep-scope.md](ep-scope.md). **`make check`** passes. Coverage **78.1%**. **[ep-manual-tests.md](ep-manual-tests.md)** — matrix column **Manual** is **✓** when a procedure exists there (including [Optional manual checks](ep-manual-tests.md#optional-manual-checks-strategy-23)), else **—**.

---

## Manual test catalogue ([ep-manual-tests.md](ep-manual-tests.md))

| Section / bullet | AC | Role |
|------------------|-----|------|
| [Sonos tool end-to-end](ep-manual-tests.md#sonos-tool-end-to-end) | AC-04.010 | Formal E2E Sonos (no auto test id `sonos`). |
| [Tool index build logging](ep-manual-tests.md#tool-index-build-logging) | AC-04.021 (+ AC-04.018 failure path) | Live INFO/ERROR; staging break of embedding/store. |
| [system_prompt in system message](ep-manual-tests.md#system_prompt-in-system-message) | AC-04.026 | LLM log check. |
| [Hermes tool list in prompt](ep-manual-tests.md#hermes-tool-list-in-prompt) | AC-04.027 | Real Hermes provider. |
| [Text-based tool flow](ep-manual-tests.md#text-based-tool-flow) | AC-04.022–024 | E2E text-based tools. |
| [Shell metacharacter rejection](ep-manual-tests.md#shell-metacharacter-rejection) | AC-04.029 | Live model / spot-check with string args. |
| [Tool invocation in logs](ep-manual-tests.md#tool-invocation-in-logs) | AC-04.013 | App log traceability. |
| [Optional manual checks](ep-manual-tests.md#optional-manual-checks-strategy-23) | AC-04.017, AC-04.015, AC-04.016, AC-04.028; native tool loop → AC-04.003, AC-04.004 | Spot-checks. |

**Matrix rule — column Manual:** **✓** = at least one procedure in [ep-manual-tests.md](ep-manual-tests.md) applies to this AC (table above). **—** = no procedure in that document for this AC.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1.1–8.1 | Done | [ep-implementation-plan.md](ep-implementation-plan.md) — incl. **8.1** tests/regression; epic closure reflected in [ep-scope.md](ep-scope.md) (**DONE**). |

---

## Test results and coverage

- **Command:** `make check`
- **Result:** **PASS**
- **Coverage:** **78.1%** statements

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Int | E2E | Manual | Automated / notes |
|----|-----|------|-----|-----|--------|-------------------|
| [AC-04.001](ep-acceptance-criteria.md#ac-04-001) | REQ-04.001–002 | ✓ | — | — | — | catalog_test, config_test |
| [AC-04.002](ep-acceptance-criteria.md#ac-04-002) | REQ-04.003 | ✓ | — | — | — | config_test, tooldefs_test |
| [AC-04.003](ep-acceptance-criteria.md#ac-04-003) | REQ-04.004–005, 019 | ✓ | ✓ | — | **✓** | [Optional: native tool loop](ep-manual-tests.md#optional-manual-checks-strategy-23) |
| [AC-04.004](ep-acceptance-criteria.md#ac-04-004) | REQ-04.006 | ✓ | ✓ | — | **✓** | [Optional: native tool loop](ep-manual-tests.md#optional-manual-checks-strategy-23) |
| [AC-04.005](ep-acceptance-criteria.md#ac-04-005) | REQ-04.007 | ✓ | — | — | — | validate_test |
| [AC-04.006](ep-acceptance-criteria.md#ac-04-006) | REQ-04.008 | ✓ | ✓ | — | — | handler_test |
| [AC-04.007](ep-acceptance-criteria.md#ac-04-007) | REQ-04.009–010 | ✓ | ✓ | — | — | substitute_test, handler_test |
| [AC-04.008](ep-acceptance-criteria.md#ac-04-008) | REQ-04.011 | ✓ | ✓ | — | — | handler_test |
| [AC-04.009](ep-acceptance-criteria.md#ac-04-009) | REQ-04.012, 034 | ✓ | — | — | — | tools_test, openai_test |
| [AC-04.010](ep-acceptance-criteria.md#ac-04-010) | REQ-04.013 | — | — | — | **✓** | [**Sonos E2E**](ep-manual-tests.md#sonos-tool-end-to-end) |
| [AC-04.011](ep-acceptance-criteria.md#ac-04-011) | REQ-04.014, 017 | ✓ | ✓ | — | — | Same path as other tools |
| [AC-04.012](ep-acceptance-criteria.md#ac-04-012) | REQ-04.015 | ✓ | ✓ | — | — | make check |
| [AC-04.013](ep-acceptance-criteria.md#ac-04-013) | REQ-04.016 | ✓ | ✓ | — | **✓** | [**Tool invocation in logs**](ep-manual-tests.md#tool-invocation-in-logs) |
| [AC-04.014](ep-acceptance-criteria.md#ac-04-014) | REQ-04.018 | ✓ | ✓ | — | — | toolindex build_test, sqlite_test |
| [AC-04.015](ep-acceptance-criteria.md#ac-04-015) | REQ-04.019 | ✓ | ✓ | — | **✓** | [Optional §](ep-manual-tests.md#optional-manual-checks-strategy-23) |
| [AC-04.016](ep-acceptance-criteria.md#ac-04-016) | REQ-04.020 | ✓ | ✓ | — | **✓** | [Optional §](ep-manual-tests.md#optional-manual-checks-strategy-23) |
| [AC-04.017](ep-acceptance-criteria.md#ac-04-017) | REQ-04.021 | ✓ | — | — | **✓** | [Optional §](ep-manual-tests.md#optional-manual-checks-strategy-23) |
| [AC-04.018](ep-acceptance-criteria.md#ac-04-018) | REQ-04.022 | ✓ | — | — | **✓** | [Tool index logging — failure path](ep-manual-tests.md#tool-index-build-logging) |
| [AC-04.019](ep-acceptance-criteria.md#ac-04-019) | REQ-04.023 | ✓ | ✓ | — | — | select_test |
| [AC-04.020](ep-acceptance-criteria.md#ac-04-020) | REQ-04.024 | ✓ | — | — | — | config_test, embedding tests |
| [AC-04.021](ep-acceptance-criteria.md#ac-04-021) | REQ-04.025 | ✓ | — | — | **✓** | build_log_test + [**live logs**](ep-manual-tests.md#tool-index-build-logging) |
| [AC-04.022](ep-acceptance-criteria.md#ac-04-022) | REQ-04.026 | ✓ | ✓ | — | **✓** | [**Text-based flow**](ep-manual-tests.md#text-based-tool-flow) |
| [AC-04.023](ep-acceptance-criteria.md#ac-04-023) | REQ-04.027–028 | ✓ | ✓ | — | **✓** | Same § |
| [AC-04.024](ep-acceptance-criteria.md#ac-04-024) | REQ-04.029 | ✓ | ✓ | — | **✓** | Same § |
| [AC-04.025](ep-acceptance-criteria.md#ac-04-025) | REQ-04.030 | ✓ | — | — | — | config_test |
| [AC-04.026](ep-acceptance-criteria.md#ac-04-026) | REQ-04.032 | ✓ | ✓ | — | **✓** | Handler test + [**LLM log**](ep-manual-tests.md#system_prompt-in-system-message) |
| [AC-04.027](ep-acceptance-criteria.md#ac-04-027) | REQ-04.033 | ✓ | — | — | **✓** | [**Hermes tool list**](ep-manual-tests.md#hermes-tool-list-in-prompt) |
| [AC-04.028](ep-acceptance-criteria.md#ac-04-028) | REQ-04.034 | ✓ | — | — | **✓** | [Optional: omit tools HTTP](ep-manual-tests.md#optional-manual-checks-strategy-23) |
| [AC-04.029](ep-acceptance-criteria.md#ac-04-029) | REQ-04.031 | ✓ | ✓ | — | **✓** | cmdsafe, noderunner, core tests + [**Shell metacharacter**](ep-manual-tests.md#shell-metacharacter-rejection) |

---

## Quality gate

**PASS.** `make check`.

---

## Gaps, risks, recommendations

| Type | Item |
|------|------|
| **Manual** | Run procedures in [ep-manual-tests.md](ep-manual-tests.md) where **✓** and record sign-off (minimum **AC-04.010** Sonos if Sonos is in production catalog). |
| **Optional** | Automated test with tool id **sonos**. |
| **Risk** | Hermes models may omit tool_call. **verify-nodes** command must not contain forbidden metacharacter sequences ([AC-04.029](ep-acceptance-criteria.md#ac-04-029)). |

---

## Artefact consistency

[ep-acceptance-criteria.md](ep-acceptance-criteria.md) includes **AC-04.029** for **REQ-04.031**. Requirements, system design, and scope aligned with implementation.
