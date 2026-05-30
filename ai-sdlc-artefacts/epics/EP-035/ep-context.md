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

- Stage 5: [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — 20 ACs (AC-35.001–020), each REQ-35.001–020; grep/build for removed imports; `-race` on relocated concurrent-write test; byte-frozen `TrustPolicy`/markers; `make check` and unchanged `config.json`.

## Design Decisions

- Merged package: **`internal/prompt`** (`pa/internal/prompt`) — files `markers.go`, `wrap.go` + tests; delete `promptmarkers` / `systemprompt`.
- Reliability test: **`tests/integration/concurrent_write_test.go`**, package `integration_test`, `//go:build integration`.
- `lifecyclelog` out of epic (EP-029 contract).

## Interfaces / Contracts

- Frozen: `TrustPolicy`, six marker consts, `ForbiddenMarkerLines`, `TextContainsForbiddenMarkerLine`, three `Wrap*` functions (byte/logic identical).
- Seven importer files → single `pa/internal/prompt` import (see [ep-system-design.md](ep-system-design.md)).

## Current Gate Summary

Stage 6 complete (draft ep-system-design). Next: system design review (stage 7).

## Open Questions

- None for stage 6 (package name `internal/prompt` confirmed in system design).

## Links

- [ep-system-design.md](ep-system-design.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-scope.md](ep-scope.md)
- [scope.md](../../scope.md)
- [strategy.md](../../strategy.md)
- [pa-architecture-review.md](../../pa-architecture-review.md) (package inventory)
