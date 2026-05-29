---
artefact: ep-context
epic_id: EP-034
status: approved
source_of_truth: false
updated_at: 2026-05-29
---

# Epic Context — EP-034 Remove tool-path LLM escalation

## Purpose

Simplify LLM routing by removing EP-006 tool-path escalation while keeping transport fallback for provider outages.

## Current Scope

Remove `escalationpolicy`, `toolfailure`, `tools.llm_escalation`, and handler/router escalation logic. Keep multi-provider transport retry. Start all paths at provider index 0.

## Key Requirements

16 REQs (11 FR, 5 NFR): remove escalation packages and APIs; keep transport fallback; reject `tools.llm_escalation`; update docs. See [ep-requirements.md](ep-requirements.md).

## Acceptance Signals

16 ACs covering no tool index change, transport fallback, config rejection, docs, tests, `make check`, validate. See [ep-acceptance-criteria.md](ep-acceptance-criteria.md).

## Design Decisions

- Delete `escalationpolicy` and `toolfailure`.
- `llmrouter`: transport fallback only; start index 0.
- Reject `tools.llm_escalation` at config load.
- Supersedes EP-006 tool-path escalation scope.

## Interfaces / Contracts

- `llmrouter.Router.Complete` — transport fallback only.
- Tool errors — plain `error`; no escalation typing.

## Current Gate Summary

| Gate | Status |
|------|--------|
| Stage 3 ep-scope | DONE |
| Stage 7 design review | pass (iteration 1) |
| Stage 8 impl plan | complete |
| Stage 10 code review | pass (iteration 2) |
| Stage 11 audit | pass |

## Open Questions

None — HOTL defaults applied for config removal and index-0 start.

## Links

- [ep-scope.md](ep-scope.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [ep-system-design.md](ep-system-design.md)
- [ep-system-design-review.md](ep-system-design-review.md)
- [ep-implementation-plan.md](ep-implementation-plan.md)
- [strategy.md](../../strategy.md) — Refactoring 0.02
- [EP-006](../EP-006/ep-scope.md) — superseded escalation scope
