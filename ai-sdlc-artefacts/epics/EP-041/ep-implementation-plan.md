---
artefact: ep-implementation-plan
epic_id: EP-041
status: draft
updated_at: 2026-05-31
---

# EP-041 — Implementation plan

- [ ] **1.1** Create `handler_full_tier_pipeline.go` with `fullTierAssembler` and five step methods.
- [ ] **1.2** Wire `buildTierFullMainPrompt` through assembler; remove duplicate logic from `mergeTailMergedToolsAndOptions` (keep as thin delegate or inline comments).
- [ ] **1.3** Run handler tier tests; add `// Covers AC-41.003` on existing parity tests if needed; `make check`.
- [ ] **1.4** Add `ep041_traceability_test.go` documenting step order constants.
