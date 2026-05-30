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

Not yet defined (stage 4).

## Acceptance Signals

Not yet defined (stage 5).

## Design Decisions

- Ambiguous heuristic → `full` (no model stage).
- Former `full_lite` → `full` (richer path; acceptable token/latency trade-off).
- Removed nested config keys rejected at load; top-level `intent_classifier` key retained per explicit-JSON rules.

## Interfaces / Contracts

- `intent.Classifier.Classify` → `Result{Tier: simple|full, Stage: heuristic|default}`.
- Enabled `intent_classifier`: `{ "enabled": true, "heuristic": { "simple_patterns", "full_patterns", "max_simple_len" } }` only.

## Current Gate Summary

Stage 3 draft ep-scope complete; downstream gates not started.

## Open Questions

- None.

## Links

- [ep-scope.md](ep-scope.md)
- [scope.md](../../scope.md)
- [strategy.md](../../strategy.md)
- [pa-architecture-review.md](../../pa-architecture-review.md)
