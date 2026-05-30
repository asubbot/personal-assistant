---
artefact: ep-context
epic_id: EP-036
status: draft
source_of_truth: false
updated_at: 2026-05-30
---

# Epic Context — EP-036 Simplify intent classification (drop model stage, two tiers)

## Purpose

Cut intent-classification complexity for increment 0.02: drop the optional model-stage LLM call and the `full_lite` tier so only `simple` and `full` remain, with heuristic → default-`full` cascade.

## Current Scope

- Delete `internal/intent/model.go`; simplify `cascade.go` and `cmd/pa/main.go` wiring.
- Remove `TierFullLite`, `full_lite_patterns`, and core `buildTierFullLiteMainPrompt`.
- Shrink `intent_classifier` config schema; reject removed keys at load; update examples, live config, and docs.
- Refresh tests; keep `simple` / `full` assembly behaviour unchanged.

## Key Requirements

- **REQ-36.001–002:** Two tiers (`simple`, `full`); remove `full_lite` / `TierFullLite`.
- **REQ-36.003–007:** Heuristic-only cascade; ambiguous → `full` / `default`; no `full_lite_patterns`.
- **REQ-36.008–011:** Delete model stage; no classification LLM in `cmd/pa`; stages `heuristic` | `default`.
- **REQ-36.012–015:** Core `simple`/`full` dispatch only; remove `full_lite` builder; tier assembly parity; former `full_lite` → `full`.
- **REQ-36.016–021:** Reject removed config keys; enabled `heuristic` schema; keep `intent_classifier` root key (`null` ok).
- **REQ-36.022–027:** Update configs/docs; regression tests; `make check` and EARS validate.

## Acceptance Signals

- **21 ACs** ([ep-acceptance-criteria.md](ep-acceptance-criteria.md)): two tiers only; heuristic cascade (`heuristic` \| `default`, ambiguous → `full` without classification LLM); model stage and `full_lite` removed from code, config, and docs.
- **Automated:** unit tests for tiers, cascade, config reject/load; integration for per-turn tier and `simple`/`full` assembly parity; former `full_lite` → `full` path.
- **Manual gates:** deleted `model.go` / `cmd/pa` wiring inspection, operator docs review, obsolete-test inventory, `make check`, `./bin/validate ears EP-036`.

## Design Decisions

- Ambiguous heuristic → `full`, `Stage: default` (no model stage).
- Former `full_lite` → `full` path (`buildTierFullMainPrompt`); operator may merge old `full_lite_patterns` into `full_patterns` for confident routing.
- Removed nested keys rejected via raw-JSON map check in `rejectRemovedUnsupportedConfigKeys` (EP-034 pattern), not silent struct drop.
- `NewCascadeClassifier(heuristic, logger)` and `NewHeuristicClassifier(simple, full, maxSimpleLen)` — no model / full_lite parameters.
- Top-level `intent_classifier` key retained; explicit-JSON rules unchanged.

## Interfaces / Contracts

- `intent.Classifier.Classify` → `Result{Tier: simple|full, Stage: heuristic|default, MessageLen}`.
- Enabled `intent_classifier`: `{ "enabled": true, "heuristic": { "simple_patterns", "full_patterns", "max_simple_len" } }` only.
- Load errors if JSON contains `intent_classifier.model_stage` or `intent_classifier.heuristic.full_lite_patterns`.

## Current Gate Summary

Stage 6 system design draft complete ([ep-system-design.md](ep-system-design.md)); stage 7 not started.

## Open Questions

- None.

## Links

- [ep-system-design.md](ep-system-design.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-scope.md](ep-scope.md)
- [scope.md](../../scope.md)
- [strategy.md](../../strategy.md)
- [pa-architecture-review.md](../../pa-architecture-review.md)
