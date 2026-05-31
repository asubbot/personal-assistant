---
artefact: ep-implementation-plan
epic_id: EP-042
updated_at: 2026-05-31
---

# EP-042 — Implementation plan

- [ ] **1.1** Create `cmd/pa/wire/` with `Build` moving wiring from `main.go` and related helpers.
- [ ] **1.2** Slim `main.go` to bootstrap only.
- [ ] **1.3** Add explicit jobs state type/docs; extend tests for initializing/ready/failed + readiness JSON.
- [ ] **1.4** Document subsystem insertion checklist in docs.
- [ ] **1.5** `make check`; commit `feat(EP-042): extract cmd/pa/wire composition root`.
