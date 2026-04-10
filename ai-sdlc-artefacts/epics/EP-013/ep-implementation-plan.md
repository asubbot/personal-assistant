# EP-013 — Implementation plan

**Pipeline:** Stage 8.

**Inputs:** [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-system-design.md](ep-system-design.md), [strategy.md](../../strategy.md)

---

## Task order

- [ ] 1. **Prompt markers and system prompt wrappers** — Add `internal/promptmarkers` (canonical lines, `TextContainsForbiddenMarkerLine`) and `internal/systemprompt` (trust policy B, wrap helpers).  
  - _Requirements:_ [REQ-13.007](ep-requirements.md#load-and-validation), [REQ-13.014](ep-requirements.md#prompt-assembly), [REQ-13.015](ep-requirements.md#prompt-assembly), [REQ-13.018](ep-requirements.md#memory-indexing)  
  - _Acceptance Criteria:_ — (foundation)  
  - **Verification:** `go test ./internal/promptmarkers/... ./internal/systemprompt/...`

- [ ] 2. **Runtime skill loader** — Add `internal/runtimeskills` (parse `SKILL.md`, `LoadDir`, `ValidateToolRefs`, marker scan).  
  - _Requirements:_ [REQ-13.004](ep-requirements.md#load-and-validation)–[REQ-13.007](ep-requirements.md#load-and-validation)  
  - _Acceptance Criteria:_ [AC-13.001](ep-acceptance-criteria.md#ac-13-001), [AC-13.002](ep-acceptance-criteria.md#ac-13-002), [AC-13.011](ep-acceptance-criteria.md#ac-13-011)  
  - **Verification:** `go test ./internal/runtimeskills/...`

- [ ] 3. **vec_skills store and skillindex** — Add `sqlite.TableSkills`, `internal/skillindex` (Build, Search, Index type).  
  - _Requirements:_ [REQ-13.008](ep-requirements.md#vec_skills-index), [REQ-13.009](ep-requirements.md#vec_skills-index)  
  - _Acceptance Criteria:_ [AC-13.010](ep-acceptance-criteria.md#ac-13-010)  
  - **Verification:** `go test ./internal/skillindex/...`

- [ ] 4. **Config and load validation** — Extend `Paths`, add `runtime_skills` and `RuntimeSkillPackages`; validate `always_include`, caps, `skills_dir` when enabled; `AllowedNativeToolIDs`.  
  - _Requirements:_ [REQ-13.001](ep-requirements.md#configuration-and-paths)–[REQ-13.003](ep-requirements.md#configuration-and-paths)  
  - _Acceptance Criteria:_ [AC-13.003](ep-acceptance-criteria.md#ac-13-003)  
  - **Verification:** `go test ./internal/config/...`

- [ ] 5. **Core handler: tool union, skills selection, marked system** — Extend `conversationHandler`, `buildSystemContent` / `gatherContext`, `buildToolOptions`, `appendToolBlocksToSystem`, `HandleMessage`, `indexTurn`; add `core.SkillIndex` and `core.Run` parameter.  
  - _Requirements:_ [REQ-13.010](ep-requirements.md#selection-and-tool-union)–[REQ-13.018](ep-requirements.md#memory-indexing)  
  - _Acceptance Criteria:_ [AC-13.004](ep-acceptance-criteria.md#ac-13-004)–[AC-13.009](ep-acceptance-criteria.md#ac-13-009), [AC-13.012](ep-acceptance-criteria.md#ac-13-012), [AC-13.014](ep-acceptance-criteria.md#ac-13-014)  
  - **Verification:** `go test ./internal/core/...`

- [ ] 6. **cmd/pa wiring** — `setup` opens `vec_skills`, builds skill index, closes on shutdown; pass skill index into `core.Run`.  
  - _Requirements:_ [REQ-13.009](ep-requirements.md#vec_skills-index)  
  - _Acceptance Criteria:_ [AC-13.013](ep-acceptance-criteria.md#ac-13-013)  
  - **Verification:** `go test ./cmd/pa/...` if tests exist; else `go build -o /dev/null ./cmd/pa`

- [ ] 7. **Sample config / docs** — Document `paths.skills_dir` and `runtime_skills` in README or config example (minimal comment in `.config/config.json` optional).  
  - _Requirements:_ —  
  - _Acceptance Criteria:_ —  
  - **Verification:** reviewer skim

---

## Checkpoints

- After tasks 1–3: `make check` (or `go test ./...` for touched packages).  
- After task 5: run `./bin/validate EP-013` (requires `make build`).  
- If design review open points affect ordering, pause and update ep-system-design before merging.

---

## Verification (epic complete)

- `make check`  
- `./bin/validate EP-013`  
- Update [ep-scope.md](ep-scope.md) **Status** to DONE when released.
