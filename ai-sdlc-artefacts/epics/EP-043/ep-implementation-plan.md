---
artefact: ep-implementation-plan
epic_id: EP-043
updated_at: 2026-05-31
---

# EP-043 — Implementation plan

- [x] **1.1** Split `handler_test.go` into domain files; target ≤600 LOC in handler_test.go.
- [x] **1.2** Add `config_test_helpers.go`; migrate ≥10 inline JSON to testdata.
- [x] **1.3** Create `architecture_guards_test.go`; merge ep034/ep038/ep040 traceability tests; preserve `// Covers AC-xx.xxx`.
- [x] **1.4** `make check` (coverage within 0.5% of baseline ~76%); commit `test(EP-043): organize handler and config tests`.
