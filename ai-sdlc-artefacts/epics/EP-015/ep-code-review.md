## Review iteration 1

**Review date:** 2026-04-14  
**Stage 10 iteration:** 1 of max 5  
**Scope:** Current branch change set for EP-015 — `internal/core/handler.go`, `internal/core/usage_turn_accum.go`, `internal/core/handler_ep015_test.go`, `internal/core/usage_turn_accum_test.go`, `internal/telegram/outbound_chunk.go`, `internal/telegram/outbound_chunk_test.go`, and epic artefacts under `ai-sdlc-artefacts/epics/EP-015/` (cross-check vs [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-implementation-plan.md](ep-implementation-plan.md), [ep-system-design.md](ep-system-design.md)).  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 3 | Nit: 1 | Suggestion: 1  
**Gate (§2.2):** Pass — Blocker, Major, and Medium open counts are all zero.

**Iteration open counts (Blocker / Major / Medium):** 0 / 0 / 0

### Summary

The implementation matches the agreed system design: the core accumulates `CompletionResult.Usage` only after successful `Complete` calls, appends a single plain-text footer when sums are non-zero and the assistant body is non-empty after trim, keeps session memory on the body without the footer line, and the Telegram layer splits Markdown source on the body only then attaches the footer to the last chunk (or a final short chunk when the last body chunk is full), skipping sends when the body is empty and only a footer would remain. Unit tests and `./bin/validate EP-015` provide traceability for AC-15.001–AC-15.007. `make check` completed successfully in this review environment. Remaining notes are minor hygiene (plan checkboxes, API visibility, edge-case UX), not merge blockers under the §2.2 gate.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|----------------|
| Minor | [ep-implementation-plan.md](ep-implementation-plan.md) | All five tasks remain marked `- [ ]` even though the described code and tests appear delivered, which misleads stage 11 / operators scanning the plan. | In a follow-up artefact edit (operator-approved), mark completed tasks `[x]` or add a short status note aligned with the branch. |
| Minor | `internal/core/handler.go` (`HandleMessage` / `finishAfterFirstLLM`) | When the model path returns a fixed Hermes parse error string (`invalidReply`) with non-zero accumulated usage, the handler still appends the token footer to that string. This is not covered by an AC and may be unintended UX. | Confirm product intent; if footers should only accompany normal assistant replies, gate footer append on the same conditions as success-path replies or add an explicit AC and test. |
| Minor | `internal/telegram/outbound_chunk.go` | `SplitTokenFooterSuffix` is exported, widening the package API without an external consumer requirement in the epic. | Prefer unexported `splitTokenFooterSuffix` unless another package must call it; keep behaviour unchanged. |
| Nit | `internal/telegram/outbound_chunk_test.go` | Non–EP-015 tests in the same file still trace `// Covers AC-01.001`, which is noisy when reading EP-015 coverage. | Optional: align trace comments with the owning epic or a project-wide neutral trace tag per validation rules. |
| Suggestion | `internal/telegram/outbound_chunk.go` | Suffix splitting is only exercised indirectly via `sendLongOutboundText`; false-positive suffix collisions are documented as an accepted risk. | Optional: small table-driven tests for `SplitTokenFooterSuffix` (no match, match at end, trimming) to lock behaviour and document intent. |

### Test / verification

- `make check` — **pass** (exit code 0).
- `make build && ./bin/validate EP-015` — **pass** (exit code 0); all seven in-scope ACs traced.

### Residual risks / follow-ups

- Model-generated text that exactly matches the strict end-anchored footer pattern could still be split incorrectly; this is acknowledged in [ep-system-design.md](ep-system-design.md) risks and is acceptable for the stated scope.
