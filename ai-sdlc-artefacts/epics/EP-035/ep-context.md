---
artefact: ep-context
epic_id: EP-035
status: draft
source_of_truth: false
updated_at: 2026-05-30
---

# Epic Context — EP-035 Consolidate small internal packages

## Purpose

Shrink the `internal/` package tree for increment 0.02 by removing empty/stub packages, relocating a misplaced reliability test, and merging EP-013 prompt marker + system-prompt helpers—without config or security contract changes.

## Current Scope

- Remove `internal/logging` (doc-only stub; zero imports).
- Remove `internal/reliability`; move `TestConcurrentWrites_NoBusyErrors` to `tests/integration`.
- Merge `internal/promptmarkers` + `internal/systemprompt` → `internal/prompt`; update six production/test import sites.
- Defer `internal/lifecyclelog` (two importers; EP-029 attribute contract).

## Key Requirements

- Byte-identical `TrustPolicy` and `<<<PA_BEGIN_*>>>` / `<<<PA_END_*>>>` marker constants.
- Equivalent forbidden-marker line validation and wrap helpers.
- No `config.json` schema or validation changes.
- AC-22.010 concurrent-write test still runs with `-race` after relocation.

## Acceptance Signals

- `make check` green; grep shows no imports of removed packages.
- EP-013-related prompt/marker tests still pass.
- `docs/configuration.md` updated if it still cites `internal/reliability`.

## Design Decisions

- Target merge name: `internal/prompt` (replaces two ~46 LOC packages).
- Reliability test destination: `tests/integration` (cross-store, already allowed by EP-022 design).
- `lifecyclelog` out of epic: avoid coupling `cmd/pa` and `memoryjob` or splitting EP-029 constants.

## Interfaces / Contracts

- Frozen strings: `TrustPolicy`, marker consts in `promptmarkers` today.
- Frozen API surface for callers: wrap functions + `TextContainsForbiddenMarkerLine` + `ForbiddenMarkerLines`.

## Current Gate Summary

Stage 3 complete (draft ep-scope + ep-context). Next: requirements (stage 4).

## Open Questions

- Confirm final merged package name (`internal/prompt` vs alternative) at system design if operators care about import ergonomics.

## Links

- [ep-scope.md](ep-scope.md)
- [scope.md](../../scope.md)
- [strategy.md](../../strategy.md)
- [pa-architecture-review.md](../../pa-architecture-review.md) (package inventory)
