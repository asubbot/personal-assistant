---
artefact: ep-system-design-review
epic_id: EP-039
status: draft
source_of_truth: true
gate: pass
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
next_action: proceed_to_stage_8
updated_at: 2026-05-31
---

# Architecture Review — EP-039 Config surface simplification

**Reviewer:** AI Agent (delegated pipeline stage 7)

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Open findings: None (Nit/Suggestion items below do not block the gate)
Next action: Proceed to stage 8

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

The system design is implementation-ready for a config-shape refactor: it names concrete structs, legacy pre-unmarshal rejection, merge resolvers, phased fixture migration, and full REQ-39.001–039.025 traceability. Verification against the branch baseline confirms the documented pain points (legacy flat `vector_search_tools`, duplicate SQLite reliability blocks, phantom `tool_output_artifacts`, hardcoded `maxToolResultPromptBytes = 8 << 10`) and proposes the correct integration points (`configRootJSONKeys`, `VectorSearchToolSettings`, `newRunConversationHandler`). Structural sections, C4 container diagram, error-handling patterns, and risks/trade-offs meet the stage 7 checklist. No Blocker/Major/Medium/Minor findings; stage 8 may proceed.

**Verdict:** Pass gate

### Strengths

- **Accurate as-is baseline:** `ToolsConfig` has no `ToolOutputArtifacts` field while `allowedToolsKeys` whitelists `tool_output_artifacts` (`internal/config/load.go:145-151`, `config.go:166-175`); operator `.config/config.json` carries a full artifact block that is dropped on unmarshal — matches REQ-39.007–011 and design Overview.
- **Correct truncation target:** `truncateToolResultForPrompt` in `handler_llm.go` uses package constant `maxToolResultPromptBytes` (`handler.go:24`, `handler_llm.go:139-143`); design correctly replaces this with config-driven handler state (REQ-39.009, REQ-39.023).
- **Legacy rejection pattern:** Pre-unmarshal scans (`rejectLegacyVectorSearchToolsShape`, `rejectLegacySQLiteReliabilityShape`) follow established EP-034/037 fail-fast style; breaking-change trade-off is documented.
- **SQLite DRY model:** `sqlite_store_defaults` + per-store overrides with `foreign_keys` only in overrides aligns with `SQLiteStoreReliabilityConfig.ToPolicy()` consumers and REQ-39.012–015.
- **Vector merge contract:** `VectorSearchToolSettings(toolID)` signature preserved; merge via `defaults` + per-tool overrides satisfies REQ-39.005–006 without `internal/tools` algorithm changes.
- **Phased implementation:** Six phases (legacy reject → SQLite → vector → artifacts → docs → verify) reduce risk of a single large testdata diff breaking `make check`.
- **Full traceability:** Requirement table maps all 25 REQs; testing strategy names new test files and parity/negative fixtures (REQ-39.019–020).

### Findings

| Id | Severity | Description | Evidence | Recommendation |
|----|----------|-------------|----------|----------------|
| N-001 | Nit | Components table says “All **11** operator fields” for `ToolOutputArtifactsConfig`; operator config and Data models list **12** fields. | `ep-system-design.md` Components table; `.config/config.json` `tools.tool_output_artifacts` (lines 38–50); design Data models lists all 12. | Fix field count to 12 in stage 6 polish or stage 8 checklist (no behavioural impact). |
| N-002 | Nit | Truncation wiring cites `newRunConversationHandler` but not the package-level `truncateToolResultForPrompt` refactor. | `handler_llm.go:139-153` is a standalone function using `maxToolResultPromptBytes`; `handler_test.go:838,861` references the constant. | Stage 8: add `toolResultPromptBytes int` on `conversationHandler`, set in `newRunConversationHandler`, convert `truncateToolResultForPrompt` to a method or pass limit; update tests. |
| N-003 | Nit | `ep-requirements.md` heading typo `#### REQ-39-013` (dash) breaks canonical anchor consistency. | `ep-requirements.md` line 178 vs `#### REQ-39.012` / `#### REQ-39.014`. | Rename to `#### REQ-39.013 — …` before relying on REQ-heading tooling. |
| S-001 | Suggestion | `tools.vector_search_tools` nil behaviour not stated when the block is omitted. | `validateTools` returns early when `VectorSearchTools == nil` (`load.go:376-377`); `VectorSearchToolSettings` falls back to package defaults (`vector_search_tools.go:48-60`). | One sentence in Data models: omitted block keeps EP-032 implicit defaults via `VectorSearchToolSettings`. |
| S-002 | Suggestion | `sqlite_store_defaults` insert position in sorted `configRootJSONKeys`. | `root_keys.go:13-33` requires sorted keys; design says “add” without position. | Insert alphabetically between `runtime_skills` and `telegram`; extend `TestConfigRootJSONKeys_*` if present. |
| S-003 | Suggestion | `ep-acceptance-criteria.md` index lists AC-39.003, 005, 007, 009, 010 but omits their `### AC-…` bodies (only 001, 002, 004, 006, 008, 011–015 written). | AC index vs sections in `ep-acceptance-criteria.md`. | Stage 8 plan should map tests to all indexed ACs; optional stage 5 fix for missing AC bodies. |
| S-004 | Suggestion | REQ-39.010 directory wiring deferred to `ArtifactDirectory(cfg)` helper. | No `tool_artifacts` / directory references under `internal/` today; design Decision accepts resolver-only. | Add one unit test row in Testing strategy: `ArtifactDirectory` when `enabled` true resolves path relative to configured roots. |
| S-005 | Suggestion | Bulk fixture migration (60+ JSON files). | Risks table; grep shows duplicate full reliability blocks across `internal/config/testdata/`. | Stage 8: scripted migrate or phase 2+3 with `make check` after each phase to limit diff churn. |

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Breaking schema (no dual-read) | Fail-fast explicit JSON; migration doc + parity tests (REQ-39.002, REQ-39.015, REQ-39.017). |
| Pre-unmarshal legacy rejection | Detect old shapes before partial unmarshal; consistent with EP-037 removed-key scans. |
| Artifact fields validated, truncation wired first | KISS; addresses phantom config and REQ-39.009; other fields ready for future tool paths (REQ-39.008, REQ-39.010). |
| Preserve `VectorSearchToolSettings` API | `internal/tools` unchanged; merge internal to config package (REQ-39.006, scope guard). |
| `sqlite_store_defaults` as new root key | Explicit JSON rule; every config gains one key (AGENTS.md); per-store blocks shrink to overrides. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-39.021 `make check` | Implementation sequencing phase 6 | OK |
| REQ-39.022 explicit JSON | Overview + `configRootJSONKeys` + nested artifact whitelist | OK |
| REQ-39.023–024 scope guards | Module boundaries; no `tools.selection` change | OK |
| REQ-39.025 operator config | Migration table + manual AC-39.015 | OK (post-migrate) |
| Security | No weakening of allowlist/secret patterns; config validation only | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ DRY schemas + minimal core wiring; no new subsystems |
| Fail fast | ✅ Legacy rejection; nested key whitelist for artifacts |
| Security | ✅ Typed validation; no silent drop of operator artifact settings |
| Testability | ✅ Named unit tests, parity tables, negative fixtures |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md) — all REQ-39.001–039.025 mapped in design § Requirement traceability
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — design testing strategy covers AC themes (see S-003 for missing AC bodies)
- **Scope:** [ep-scope.md](ep-scope.md)

### Requirement traceability verification (this iteration)

| REQ range | Design coverage | Code baseline aligned |
|-----------|-----------------|------------------------|
| REQ-39.001–006 | Data models + merge resolver + legacy reject | Flat triple `VectorSearchToolsConfig` (`vector_search_tools.go:21-26`) |
| REQ-39.007–011 | `ToolOutputArtifactsConfig` + nested validation + core wire | Whitelist-only phantom (`load.go:150`, no struct field) |
| REQ-39.012–015 | `sqlite_store_defaults` + merge + legacy reject | Full duplicate store blocks (`config.go:47-50`, operator config 223–234) |
| REQ-39.016–020 | Phasing + testing strategy | Testdata still legacy shape |
| REQ-39.021–025 | Risks + verification | `tools.selection` unchanged; core limited to wiring |

---

**Signal:** `STAGE_7_COMPLETE: ai-sdlc-artefacts/epics/EP-039/ep-system-design-review.md [gate=pass, iteration 1, blocker:0 major:0 medium:0 minor:0]`
