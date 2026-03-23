# EP-009 Dynamic Tool Creation with Docker Sandbox — Implementation plan

**Pipeline:** Stage 7 — see `ai-sdlc/specification/pipeline.spec.md` in the repo (not under `ai-sdlc-artefacts/`; no link per implementation-planning skill).  
**Test strategy:** [../../strategy.md](../../strategy.md)

**Related artefacts**

| Document | Path |
|----------|------|
| Scope | [ep-scope.md](ep-scope.md) |
| Requirements | [ep-requirements.md](ep-requirements.md) |
| Acceptance criteria | [ep-acceptance-criteria.md](ep-acceptance-criteria.md) |
| System design | [ep-system-design.md](ep-system-design.md) |

**Purpose:** Ordered, verifiable coding tasks for EP-009. Each task maps to requirements and acceptance criteria where applicable. **Product code changes** require explicit user approval (see `AGENTS.md` at repo root) before implementation.

**Contents**

- [Checkpoints](#checkpoints)
- [1. Config and secret patterns](#1-config-and-secret-patterns)
- [2. Tool catalog: append and validation helpers](#2-tool-catalog-append-and-validation-helpers)
- [3. Native create_tool tool](#3-native-create_tool-tool)
- [4. Core wiring and tool index](#4-core-wiring-and-tool-index)
- [5. Sandbox template hardening (optional)](#5-sandbox-template-hardening-optional)
- [6. Unit tests and coverage gate](#6-unit-tests-and-coverage-gate)
- [7. Integration tests (SSH / Docker)](#7-integration-tests-ssh--docker)
- [8. Documentation](#8-documentation)

---

## Checkpoints

- **CP-A:** After §1–§2, run `go test ./internal/config/... ./internal/toolcatalog/...` (packages touched)—no full `make check` required yet.
- **CP-B:** After §3–§4, run **`make check`** before merging feature; fix failures.
- **CP-C:** User approval before starting **product code** work if not already granted for EP-009.

---

## 1. Config and secret patterns

- [ ] **1.1** Extend `ToolsConfig` (or nested struct) with optional `create_tool_secret_patterns []string` (exact JSON field name aligned with [ep-system-design.md](ep-system-design.md#config-extension-secret-detection)).
  - Compile each pattern with `regexp.Compile` at config load; on any compile error, return error from `config.Load` / validation (fail fast) — [REQ-09.017](ep-requirements.md#non-functional-requirements), [AC-09.017](ep-acceptance-criteria.md#ac-09-017).
  - **Verification:** Unit test: valid patterns load; invalid regex rejects config.
  - _Requirements:_ [REQ-09.017](ep-requirements.md#non-functional-requirements)
  - _Acceptance Criteria:_ [AC-09.017](ep-acceptance-criteria.md#ac-09-017)

---

## 2. Tool catalog: append and validation helpers

- [ ] **2.1** Add pure functions (package `toolcatalog` or subpackage):
  - `ValidateCreateToolTemplatePrefix(template string) error` — must start with `docker run --rm --network bridge` or `docker run --rm --network none` — [REQ-09.009](ep-requirements.md#tool-creation), [AC-09.009](ep-acceptance-criteria.md#ac-09-009).
  - Optional: `ValidateSandboxResourceSubstrings(template string) error` — require `--memory="256m"`, `--cpus="0.5"`, and a 30s bound (document exact substring rules in code comment) — [REQ-09.002](ep-requirements.md#docker-sandbox-execution)–[REQ-09.004](ep-requirements.md#docker-sandbox-execution), [AC-09.002](ep-acceptance-criteria.md#ac-09-002)–[AC-09.004](ep-acceptance-criteria.md#ac-09-004).
  - **Verification:** Unit tests: accept/reject cases.
  - _Requirements:_ [REQ-09.009](ep-requirements.md#tool-creation)
  - _Acceptance Criteria:_ [AC-09.009](ep-acceptance-criteria.md#ac-09-009)

- [ ] **2.2** Implement `AppendToolToCatalogFile(absPath string, tool *Tool) error`:
  - Read existing YAML, append one `tools:` list item, **atomic write** (temp file in same dir + rename) — [ep-system-design.md](ep-system-design.md#concurrency-and-catalog-writes), [REQ-09.011](ep-requirements.md#tool-creation), [AC-09.011](ep-acceptance-criteria.md#ac-09-011).
  - **Verification:** Unit test with temp dir: append twice; file valid YAML; concurrent append test with external mutex (caller holds mutex in 3.x).
  - _Requirements:_ [REQ-09.011](ep-requirements.md#tool-creation)
  - _Acceptance Criteria:_ [AC-09.011](ep-acceptance-criteria.md#ac-09-011)

- [ ] **2.3** Add helper to parse `arguments` from `create_tool` params (YAML or JSON fragment) into `[]ArgumentRule` — must match existing catalog schema — [REQ-09.008](ep-requirements.md#tool-creation), [AC-09.008](ep-acceptance-criteria.md#ac-09-008).
  - **Verification:** Unit test round-trip.
  - _Requirements:_ [REQ-09.008](ep-requirements.md#tool-creation)
  - _Acceptance Criteria:_ [AC-09.008](ep-acceptance-criteria.md#ac-09-008)

---

## 3. Native create_tool tool

- [ ] **3.1** Implement `internal/tools/create_tool.go` (name may differ): `Tool` with `Name() == "create_tool"`, params: `id`, `index_text`, `template`, `node_id`, optional `arguments`, `system_prompt` — [REQ-09.008](ep-requirements.md#tool-creation), [AC-09.008](ep-acceptance-criteria.md#ac-09-008).

- [ ] **3.2** Constructor accepts: `*toolcatalog.Catalog` (or interface), **absolute path** to catalog file, `*config.Config` (for nodes + secret patterns), `sync.Locker` or internal mutex for create critical section.
  - Flow: acquire lock → validate prefix → optional resource substring validation (if 5.x done) → duplicate `id` → secret regex on concatenated fields → `AppendToolToCatalogFile` → `Catalog.Tools[id]=tool` → release lock → return success string with tool id — [REQ-09.009](ep-requirements.md#tool-creation)–[REQ-09.013](ep-requirements.md#tool-creation), [AC-09.009](ep-acceptance-criteria.md#ac-09-009)–[AC-09.013](ep-acceptance-criteria.md#ac-09-013), [REQ-09.017](ep-requirements.md#non-functional-requirements).
  - On duplicate id — [REQ-09.010](ep-requirements.md#tool-creation), [AC-09.010](ep-acceptance-criteria.md#ac-09-010).
  - **Verification:** Unit tests with mocked filesystem or temp dir + fake catalog.
  - _Requirements:_ [REQ-09.008](ep-requirements.md#tool-creation)–[REQ-09.013](ep-requirements.md#tool-creation), [REQ-09.017](ep-requirements.md#non-functional-requirements)
  - _Acceptance Criteria:_ [AC-09.008](ep-acceptance-criteria.md#ac-09-008)–[AC-09.013](ep-acceptance-criteria.md#ac-09-013), [AC-09.017](ep-acceptance-criteria.md#ac-09-017)

---

## 4. Core wiring and tool index

- [ ] **4.1** In `cmd/pa/main.go`, register `create_tool` after catalog load: pass pointer to `cfg.ToolCatalog`, resolved absolute path to `tool_catalog_path`, config, shared mutex — [REQ-09.008](ep-requirements.md#tool-creation).
  - **Verification:** `go build ./...`; integration smoke: binary starts with feature flag / config (optional empty patterns).
  - _Requirements:_ [REQ-09.008](ep-requirements.md#tool-creation)
  - _Acceptance Criteria:_ [AC-09.008](ep-acceptance-criteria.md#ac-09-008)

- [ ] **4.2** After successful create (in core or via callback from tool): **best-effort** `toolindex` embed for new `id` + `index_text`; log error on failure; optional single retry — [ep-system-design.md](ep-system-design.md#tool-vector-index-after-create), [REQ-09.012](ep-requirements.md#tool-creation) (vector lag acceptable), [REQ-09.014](ep-requirements.md#non-functional-requirements) (startup time NFR for pre-selection path).
  - **Verification:** Unit test with mock embedder; or log assertion in integration.
  - _Requirements:_ [REQ-09.012](ep-requirements.md#tool-creation), [REQ-09.014](ep-requirements.md#non-functional-requirements)
  - _Acceptance Criteria:_ [AC-09.012](ep-acceptance-criteria.md#ac-09-012), [AC-09.014](ep-acceptance-criteria.md#ac-09-014)

- [ ] **4.3** Measure `create_tool` path duration in tests or benchmark hook for [REQ-09.015](ep-requirements.md#non-functional-requirements), [AC-09.015](ep-acceptance-criteria.md#ac-09-015) (1s budget on CI hardware—may be soft threshold with flake allowance documented).

---

## 5. Sandbox template hardening (optional)

- [ ] **5.1** If not done in 2.1: enforce resource substrings before persist — ties to [AC-09.002](ep-acceptance-criteria.md#ac-09-002)–[AC-09.004](ep-acceptance-criteria.md#ac-09-004).
  - **Verification:** Unit tests.
  - _Requirements:_ [REQ-09.002](ep-requirements.md#docker-sandbox-execution), [REQ-09.003](ep-requirements.md#docker-sandbox-execution), [REQ-09.004](ep-requirements.md#docker-sandbox-execution)
  - _Acceptance Criteria:_ [AC-09.002](ep-acceptance-criteria.md#ac-09-002), [AC-09.003](ep-acceptance-criteria.md#ac-09-003), [AC-09.004](ep-acceptance-criteria.md#ac-09-004)

---

## 6. Unit tests and coverage gate

- [ ] **6.1** Add/extend tests so combined coverage of `create_tool` + template validation packages is **≥ 70%** — [REQ-09.016](ep-requirements.md#non-functional-requirements), [AC-09.016](ep-acceptance-criteria.md#ac-09-016).

- [ ] **6.2** Extend `Makefile` `check` (or add target) to enforce threshold on selected packages, e.g. `go test -cover -coverprofile=... ./internal/tools/...` with minimum percentage, or document manual gate until automation stable.
  - **Verification:** `make check` passes locally; CI matches.
  - _Requirements:_ [REQ-09.016](ep-requirements.md#non-functional-requirements)
  - _Acceptance Criteria:_ [AC-09.016](ep-acceptance-criteria.md#ac-09-016)

---

## 7. Integration tests (SSH / Docker)

- [ ] **7.1** Extend existing SSH integration testbed (see `tests/integration/`) or add EP-009 test file:
  - Execute a catalog tool whose template uses `docker run ... --network bridge` and assert remote command contains expected flags — [AC-09.001](ep-acceptance-criteria.md#ac-09-001)–[AC-09.004](ep-acceptance-criteria.md#ac-09-004).
  - Execute template with `--network none` and run probe inside container; outbound to public endpoint **fails** — [REQ-09.018](ep-requirements.md#docker-sandbox-execution), [AC-09.018](ep-acceptance-criteria.md#ac-09-018).
  - **Prerequisites:** `pa-sandbox:*` images on test node or skip with `t.Skip` + env gate — [REQ-09.005](ep-requirements.md#docker-sandbox-execution)–[REQ-09.007](ep-requirements.md#docker-sandbox-execution), [AC-09.005](ep-acceptance-criteria.md#ac-09-005)–[AC-09.007](ep-acceptance-criteria.md#ac-09-007).

- [ ] **7.2** Optional: end-to-end `create_tool` → file append → invoke new tool (requires LLM mock or direct handler test)—covers [AC-09.011](ep-acceptance-criteria.md#ac-09-011)–[AC-09.013](ep-acceptance-criteria.md#ac-09-013).

- [ ] **7.3** Timing assertions for [REQ-09.014](ep-requirements.md#non-functional-requirements), [AC-09.014](ep-acceptance-criteria.md#ac-09-014) (5s sandbox start with cached image)—best-effort; document environment variance.

---

## 8. Documentation

- [ ] **8.1** Update `docs/configuration.md` at repo root (when code ships): `create_tool_secret_patterns`, example, fail-fast behaviour — supports [REQ-09.017](ep-requirements.md#non-functional-requirements).

- [ ] **8.2** Add operator section: building/tagging `pa-sandbox:python`, `:node`, `:base`; extending node **allowlist** for `docker run` lines; link to [ep-scope.md](ep-scope.md) — [REQ-09.005](ep-requirements.md#docker-sandbox-execution)–[REQ-09.007](ep-requirements.md#docker-sandbox-execution).

- [ ] **8.3** Regenerate [diagrams/c4-container.png](diagrams/c4-container.png) if [c4-container.puml](diagrams/c4-container.puml) changes — [ep-system-design.md](ep-system-design.md#documentation-and-diagram-maintenance).

---

## Traceability summary

| REQ | Tasks (primary) |
|-----|-----------------|
| REQ-09.001–004, 018 | 2.1, 5.1, 7.1 |
| REQ-09.005–007 | 7.1, 8.2 |
| REQ-09.008–013 | 2.3, 3.x, 4.1–4.2, 7.2 |
| REQ-09.014–015 | 4.2–4.3, 7.3 |
| REQ-09.016 | 6.1–6.2 |
| REQ-09.017 | 1.1, 3.2, 8.1 |

---

## Notes

- **Allowlist:** Operators must update per-node `command_allowlist` to permit new `docker run` command shapes—out of code path but required for execution ([ep-system-design.md](ep-system-design.md)).
- **Hermes / native tools:** Ensure `create_tool` appears in registry only when EP-009 config enabled if product decision adds a feature flag (optional; default ON if not specified).
