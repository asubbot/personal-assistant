# Code review — EP-020 Natural-Language Scheduled Job Creation from Telegram

---

## Review iteration 1

**Review date:** 2026-04-16  
**Stage 10 iteration:** 1 of max 5  
**Scope:** EP-020 working-tree change set (hybrid NL create: `internal/jobs`, `cmd/pa`, `internal/telegram` tests)  
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 1 | Minor: 0 | Nit: 0 | Suggestion: 1  
**Gate:** Fail

### Summary

Initial review found routing and fallback-order gaps relative to REQ-20.007 / REQ-20.013. Findings were addressed in a follow-up implementation pass before iteration 2.

### Findings (resolved in code before iteration 2)

| Severity | Location | Issue | Resolution |
|----------|----------|-------|--------------|
| Major | `cmd/pa/jobs_runtime.go` | Malformed schedule-intent messages could bypass NL-create handling when intent regex required valid `HH:MM`. | Broadened schedule-intent detection; route to manager for deterministic rejection. |
| Medium | `internal/jobs/manager.go` | Fallback not consistently attempted after strict-parser non-match for explicit intent. | Refactored to strict match first, then single fallback attempt for explicit intent. |
| Suggestion | `cmd/pa/ep020_e2e_test.go` | Strict-template path lacked dedicated E2E. | Added `TestEP020_E2E_StrictTemplateCreateManageRunNowDelivery`. |

---

## Review iteration 2

**Review date:** 2026-04-16  
**Stage 10 iteration:** 2 of max 5  
**Scope:** EP-020 change set after iteration-1 fixes  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 0  
**Gate:** Pass

### Summary

Iteration-1 issues are resolved. Hybrid creation (deterministic-first, native-tool fallback), routing, tests, and AC validation align with EP-020 scope. No open Blocker/Major/Medium/Minor findings.

### Findings

None.

---

## Review iteration 3

**Review date:** 2026-04-16  
**Stage 10 iteration:** 3 of max 5  
**Scope:** Restored LLM malformed-intent fallback enforcement flow (`cmd/pa/jobs_runtime.go`, `internal/jobs/create_context.go`, `internal/jobs/create_scheduled_job_tool.go`) and matching tests/docs  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 0  
**Gate:** Pass

### Summary

Restored fallback enforcement flow is internally consistent: malformed explicit schedule-intent requests escalate to base LLM prompts that enforce `create_scheduled_job`, context wiring for actor/chat is present, and AC trace coverage remains complete.

### Findings

None.
