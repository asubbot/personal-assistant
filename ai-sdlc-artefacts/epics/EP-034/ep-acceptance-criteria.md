---
artefact: ep-acceptance-criteria
epic_id: EP-034
status: draft
source_of_truth: true
updated_at: 2026-05-29
---

# EP-034 — Remove tool-path LLM escalation — Acceptance criteria

This document defines acceptance criteria for [ep-scope.md](ep-scope.md), traced to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC | REQ (trace) | Summary |
|----|-------------|---------|
| [AC-34.001](#ac-34-001) | [REQ-34.001](ep-requirements.md#removal) | Tool failure does not change provider index |
| [AC-34.002](#ac-34-002) | [REQ-34.002](ep-requirements.md#removal) | No `escalationpolicy` in product code |
| [AC-34.003](#ac-34-003) | [REQ-34.003](ep-requirements.md#removal) | No `toolfailure` in product code |
| [AC-34.004](#ac-34-004) | [REQ-34.004](ep-requirements.md#router) | Transport fallback switches provider |
| [AC-34.005](#ac-34-005) | [REQ-34.005](ep-requirements.md#router) | No tool-path escalation API in router |
| [AC-34.006](#ac-34-006) | [REQ-34.006](ep-requirements.md#router) | New turn starts at provider index 0 |
| [AC-34.007](#ac-34-007) | [REQ-34.007](ep-requirements.md#config) | Config rejects `tools.llm_escalation` |
| [AC-34.008](#ac-34-008) | [REQ-34.008](ep-requirements.md#config) | Examples have no escalation block |
| [AC-34.009](#ac-34-009) | [REQ-34.009](ep-requirements.md#tool-execution) | Tool paths use plain errors |
| [AC-34.010](#ac-34-010) | [REQ-34.010](ep-requirements.md#observability) | No tool-escalation logs |
| [AC-34.011](#ac-34-011) | [REQ-34.011](ep-requirements.md#documentation) | Docs updated |
| [AC-34.012](#ac-34-012) | [REQ-34.012](ep-requirements.md#nfr--traceability-tests-quality) | EP-006 supersession recorded |
| [AC-34.013](#ac-34-013) | [REQ-34.013](ep-requirements.md#nfr--traceability-tests-quality) | EP-006 escalation tests removed/rewritten |
| [AC-34.014](#ac-34-014) | [REQ-34.014](ep-requirements.md#nfr--traceability-tests-quality) | Regression tests present |
| [AC-34.015](#ac-34-015) | [REQ-34.015](ep-requirements.md#nfr--traceability-tests-quality) | `make check` passes |
| [AC-34.016](#ac-34-016) | [REQ-34.016](ep-requirements.md#req-34-016--validate-ep-034-passes) | `./bin/validate EP-034` passes |

---

## Scenarios

### AC-34.001 No escalation on tool failure (Trace: REQ-34.001)

Given two configured providers and a qualifying tool failure during a tool round  
When the handler calls `Complete` again  
Then the active provider index SHALL remain unchanged.

### AC-34.004 Transport fallback (Trace: REQ-34.004)

Given two configured providers and provider index 0 returns a retryable transport error on `Complete`  
When the router handles the call  
Then `Complete` SHALL be attempted on provider index 1.

### AC-34.007 Config rejects escalation key (Trace: REQ-34.007)

Given a `config.json` containing `tools.llm_escalation`  
When config load runs  
Then load SHALL fail with an explicit validation error.

### AC-34.014 Regression tests (Trace: REQ-34.014)

Given EP-034 implementation  
When the test suite runs  
Then tests SHALL cover no tool escalation and transport fallback.

---

## Acceptance criteria

### AC-34.001

Given escalation was previously enabled with two providers and a qualifying tool failure  
When the tool round completes and the handler calls `Complete` again  
Then the active provider index SHALL remain unchanged from the index used before the tool failure.

### AC-34.002

Given EP-034 is implemented  
When searching product packages under `cmd/` and `internal/`  
Then no import path SHALL reference `pa/internal/escalationpolicy`.

### AC-34.003

Given EP-034 is implemented  
When searching product packages under `cmd/` and `internal/`  
Then no import path SHALL reference `pa/internal/core/toolfailure`.

### AC-34.004

Given two configured `llm_providers` and provider index 0 returns a retryable transport error on `Complete`  
When the router handles the call  
Then `Complete` SHALL be attempted on provider index 1.

### AC-34.005

Given EP-034 is implemented  
When inspecting `internal/llmrouter` public API  
Then `OnQualifyingFailure`, `ActionEscalatePolicy`, and per-turn escalation counters SHALL be absent.

### AC-34.006

Given a prior user turn used provider index 1 due to transport fallback  
When a new user message is handled  
Then the first `Complete` for that message SHALL use provider index 0.

### AC-34.007

Given a `config.json` containing `tools.llm_escalation`  
When config load runs  
Then load SHALL fail with an explicit validation error.

### AC-34.008

Given repository example configs under `config.examples/`  
When inspected  
Then none SHALL contain an `llm_escalation` key under `tools`.

### AC-34.009

Given a catalog validation error or SSH exec failure on a tool call  
When the error is returned to the handler  
Then the error SHALL NOT be wrapped in escalation policy types.

### AC-34.010

Given a tool execution failure during a user turn  
When application logs are emitted for that turn  
Then no log event SHALL record tool-path LLM escalation or `escalations_used` for tool failure.

### AC-34.011

Given EP-034 is implemented  
When reading `docs/configuration.md` and `docs/llm-provider-roles-and-logging.md`  
Then tool-path escalation and `baseline_index` SHALL not be documented as active product behaviour.

### AC-34.012

Given EP-034 artefacts are complete  
When reading [ep-scope.md](ep-scope.md) traceability  
Then EP-006 tool-path escalation supersession SHALL be explicitly stated.

### AC-34.013

Given EP-034 is implemented  
When running the test suite  
Then tests whose only purpose is EP-006 tool-path escalation SHALL be removed or rewritten.

### AC-34.014

Given EP-034 is implemented  
When running unit or integration tests tagged for EP-034  
Then at least one test SHALL cover AC-34.001 and at least one SHALL cover AC-34.004.

### AC-34.015

Given EP-034 implementation is complete  
When `make check` runs from the repository root  
Then it SHALL exit successfully.

### AC-34.016

Given EP-034 implementation is complete  
When `./bin/validate EP-034` runs from the repository root  
Then it SHALL exit successfully with full AC coverage.
