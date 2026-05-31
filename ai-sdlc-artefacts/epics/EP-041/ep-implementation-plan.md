---
artefact: ep-implementation-plan
epic_id: EP-041
status: draft
updated_at: 2026-05-31
---

# EP-041 — Implementation plan

- [x] **1.1** Create `handler_full_tier_pipeline.go` with `fullTierAssembler` and five step methods.
- [x] **1.2** Wire `buildTierFullMainPrompt` through assembler; remove duplicate logic from `mergeTailMergedToolsAndOptions` (keep as thin delegate or inline comments).
- [x] **1.3** Run handler tier tests; add `// Covers AC-41.003` on existing parity tests if needed; `make check`.
- [x] **1.4** Add `ep041_traceability_test.go` documenting step order constants.
