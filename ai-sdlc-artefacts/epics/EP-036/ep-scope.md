---
artefact: ep-scope
epic_id: EP-036
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# Epic scope — EP-036 Simplify intent classification (drop model stage, two tiers)

| Field | Content |
|-------|---------|
| **ID** | EP-036 |
| **Status** | DONE |
| **Title** | Simplify intent classification (drop model stage, two tiers) |
| **Description** | Remove the optional intent-classifier model stage and the `full_lite` tier so classification is heuristic-only with two outcomes (`simple`, `full`). Reduces per-message LLM cost, latency, and branching in core prompt assembly as part of Refactoring increment 0.02. |
| **First version date** | 2026-05-30 |

## Glossary

- **Intent classifier cascade:** Pre-main-LLM step that assigns a complexity tier from the user message (EP-017/018). After this epic: heuristic patterns → default `full` when ambiguous; no separate classification LLM call.
- **Model stage:** Optional cheap-LLM stage (`internal/intent/model.go`, `intent_classifier.model_stage` config) that resolved ambiguous heuristic results via an extra `Complete` per message. Removed by this epic.
- **`full_lite` tier:** Middle complexity tier (EP-018): session history and tools, but no RAG or runtime-skills tail. Removed; messages formerly classified `full_lite` use the `full` path (safer, richer prompt).
- **Explicit JSON configuration:** Product `config.json` lists every documented top-level key; unknown keys rejected; optional blocks disabled with JSON `null`; invalid values fail at load (AGENTS.md). Schema keys may be removed when documented; remaining keys stay strictly validated.

## Scope (features/capabilities)

- **Remove model stage:** Delete `internal/intent/model.go` and `model_test.go`; remove `ModelClassifier` wiring from `cmd/pa/main.go` (`buildIntentClassifier`); simplify `internal/intent/cascade.go` to heuristic → default `full` (no model branch). Cascade `Result.Stage` values become `heuristic` or `default` only.
- **Two tiers only:** Remove `TierFullLite` / `full_lite` from `internal/intent/tier.go` and all references in `internal/intent/` (heuristic order becomes length → simple → full → ambiguous).
- **Core prompt assembly:** Remove `buildTierFullLiteMainPrompt` and `TierFullLite` branches in `internal/core/handler_tier_main_prompt.go` and related tests; `assembleTierMainLLMParams` handles only `simple` and `full`. No change to `simple` or `full` assembly logic beyond deleting the `full_lite` path.
- **Config schema shrink (strict validation preserved):** Remove `intent_classifier.model_stage` (`ClassificationModelConfig`) and `intent_classifier.heuristic.full_lite_patterns` from `internal/config` structs, load/validate, and `resolve.go` path resolution. Config load **rejects** legacy keys if present (same fail-fast pattern as EP-034 for removed blocks). Top-level `intent_classifier` remains a documented key (`null` or enabled object with `heuristic` only).
- **Config and docs alignment:** Update `config.examples/config.example.json`, live `.config/config.json`, `internal/config/testdata/` fixtures that embed removed fields, `docs/configuration.md`, and `docs/llm-provider-roles-and-logging.md` so documented schema matches code and example configs still load. Operator-facing prose describes two-tier heuristic-only classification.
- **Tests and observability:** Update or remove tests covering model-stage parsing, three-way classification, and `full_lite` prompt/token deltas (`handler_ep018_test.go`, cascade/heuristic tests, `cmd/pa/ep024_operator_logging_test.go`, config validation tests). Add/adjust tests proving ambiguous messages default to `full` without an LLM call and that removed config keys fail load.
- **Behavioural intent (explicit):** Messages that would have been `full_lite` (heuristic or model) now run the existing `full` path. Ambiguous heuristic results default to `full` instead of invoking the model stage. `simple` and `full` paths are unchanged in their own assembly rules.

## Out of scope

- Retuning operator heuristic patterns or adding new tiers.
- Changes to `tools.dynamic_selection` semantics for the `full` tier (only remove `TierFullLite`-specific references in comments/tests).
- Broader `conversationHandler` refactors (e.g. further tier-builder extraction from EP-026).
- Removing the intent classifier entirely (`intent_classifier: null` remains valid).
- Performance/load testing or token-budget re-benchmarking beyond updated unit tests.

## Success criteria

- No production code imports or constructs `ModelClassifier`; no per-message classification LLM call in the intent pipeline.
- Only `simple` and `full` tiers exist in `intent.Tier` and core tier dispatch.
- Config load succeeds for updated example and live configs; config load **fails** if `intent_classifier.model_stage` or `intent_classifier.heuristic.full_lite_patterns` is present (removed keys).
- `intent_classifier` top-level key still listed in root-key validation; enabled configs validate `heuristic` regexes and `max_simple_len` as today (minus removed fields).
- Operator docs describe two-tier heuristic cascade only; no model-stage setup instructions.
- `make check` passes.
- No regression in `simple` or `full` tier prompt assembly behaviour (verified by existing or updated tier tests).

## Traceability

- **Scope:** Reduces **Core** orchestration complexity while preserving reliability and security posture ([scope.md](../../scope.md)).
- **Strategy:** Maps to **Refactoring 0.02** — remove extra architecture complexity; direction **C** (simplify intent tiers / drop classifier model stage) from [pa-architecture-review.md](../../pa-architecture-review.md) ([strategy.md](../../strategy.md)).
- **Supersedes (partial):** Model-stage scope from [EP-017](../EP-017/ep-scope.md); `full_lite` tier scope from [EP-018](../EP-018/ep-scope.md). EP-017/018 epics remain DONE; behaviour intentionally simplified.
