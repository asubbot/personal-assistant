# Code review — EP-033 Memory Summarization Retry

---

## Review iteration 1

**Review date:** 2026-04-21  
**Stage 10 iteration:** 1 of max 5  
**Scope:**  
- `internal/memoryjob/memoryjob.go`  
- `internal/memoryjob/memoryjob_test.go`  
- `internal/memoryjob/builtin_schedule_tick_test.go`  
- `tests/integration/ep033_manual_test.go`  
- `ai-sdlc-artefacts/epics/EP-033/ep-scope.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-requirements.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-acceptance-criteria.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-system-design.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-system-design-review.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-implementation-plan.md`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-context.puml`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-context.png`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-container.puml`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-container.png`  

**Iteration summary — open counts:** Blocker: 1 | Major: 1 | Medium: 1 | Minor: 1 | Nit: 0 | Suggestion: 1  
**Gate (§2.2):** Fail (Blocker/Major/Medium/Minor not all zero)

### Summary

The EP-033 implementation is close and generally aligns with the intended bounded retry architecture (dedupe map, `notBefore`, bounded delays, and passing memoryjob tests).  
However, one correctness bug breaks the same-day-target retry contract, and one dedupe-key mismatch can allow duplicate chains for the same target day under startup/scheduled overlap.  
Merge is blocked until these are fixed.

### Findings

| Severity | Location | Issue | Recommendation |
|---|---|---|---|
| Blocker | `internal/memoryjob/memoryjob.go` (`jobSummarizeYesterday`, `jobCatchUpDay`, `enqueueDayRetry`) | Day target was computed at run time, so delayed retry could drift to another day. | Bind day target at enqueue time and run retries against fixed target day. |
| Major | `internal/memoryjob/memoryjob.go` (`maybeEnqueueDaily`, `enqueueStartup`) | Startup and scheduled paths used different day-key semantics, allowing duplicate retry chains. | Use one target-day key definition for both enqueue paths. |
| Medium | `internal/memoryjob/memoryjob_test.go` | Missing explicit tests for cross-midnight retry target preservation and startup/scheduled dedupe interaction. | Add deterministic tests that advance clock across midnight and verify same-day target + single dedupe chain. |
| Minor | `ai-sdlc-artefacts/epics/EP-033/ep-implementation-plan.md` | Task checklist not marked complete while code changes were implemented. | Update task status/checklist to keep plan traceability accurate. |
| Suggestion | `internal/memoryjob/memoryjob.go` | Retry exhaustion log lacked explicit structured outcome field. | Consider adding `outcome=retry_exhausted` for easier machine parsing. |

### Test / verification

- `make check`: pass
- `./bin/validate EP-033`: pass

---

## Review iteration 2

**Review date:** 2026-04-21  
**Stage 10 iteration:** 2 of max 5  
**Scope:**  
- `internal/memoryjob/memoryjob.go`  
- `internal/memoryjob/memoryjob_test.go`  
- `internal/memoryjob/builtin_schedule_tick_test.go`  
- `tests/integration/ep033_manual_test.go`  
- `ai-sdlc-artefacts/epics/EP-033/ep-scope.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-requirements.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-acceptance-criteria.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-system-design.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-system-design-review.md`  
- `ai-sdlc-artefacts/epics/EP-033/ep-implementation-plan.md`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-context.puml`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-context.png`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-container.puml`  
- `ai-sdlc-artefacts/epics/EP-033/diagrams/c4-container.png`  

**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 1  
**Gate (§2.2):** Pass

### Summary

Implementation now aligns with EP-033 scope and plan: retries are bounded and day-job-only, dedupe is enforced by stable day key under queue mutex, and retries stay in the existing runner loop with preserved user-turn deferral semantics.  
No blocking findings remain.

### Findings

| Severity | Location | Issue | Recommendation |
|---|---|---|---|
| Suggestion | `tests/integration/ep033_manual_test.go` | AC-33.009 remains manual-only via `t.Skip`. | Consider adding deterministic automated assertion for retry structured logs in future hardening pass. |

### Test / verification

- `make check`: pass
- `./bin/validate EP-033`: pass
