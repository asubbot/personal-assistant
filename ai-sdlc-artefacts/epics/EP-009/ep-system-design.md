# EP-009 Dynamic Tool Creation with Docker Sandbox — System design

**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) (stage 6)  
**Inputs:** [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-scope.md](ep-scope.md)

**Contents**

- [Overview](#overview)
- [Architecture](#architecture)
- [Module boundaries](#module-boundaries)
- [Components and interfaces](#components-and-interfaces)
- [Data models](#data-models)
- [Error handling](#error-handling)
- [Testing strategy](#testing-strategy)
- [Risks and trade-offs](#risks-and-trade-offs)
- [Requirement traceability](#requirement-traceability)

---

## Overview

EP-009 extends the existing PersonalAssistant monolith ([EP-004](../EP-004/ep-scope.md) baseline) with a **native `create_tool`** path, **runtime updates** to the tool catalog file and in-memory catalog, **template whitelist** and **secret-pattern** checks before persistence, and **Docker sandbox execution** on nodes via the existing **SSH + `run_on_node`** pipeline. The LLM calls `create_tool` like any other native tool; sandbox workloads are ordinary catalog tools whose `template` expands to a `docker run` command meeting [Docker Sandbox Execution](ep-requirements.md#docker-sandbox-execution) requirements.

Design scope covers: [REQ-09.001](ep-requirements.md#docker-sandbox-execution)–[REQ-09.007](ep-requirements.md#docker-sandbox-execution), [REQ-09.018](ep-requirements.md#docker-sandbox-execution), [REQ-09.008](ep-requirements.md#tool-creation)–[REQ-09.013](ep-requirements.md#tool-creation), [REQ-09.014](ep-requirements.md#non-functional-requirements)–[REQ-09.017](ep-requirements.md#non-functional-requirements). Acceptance tests align with [ep-acceptance-criteria.md](ep-acceptance-criteria.md).

---

## Architecture

PersonalAssistant remains a **single deployable process** (see C2 below). EP-009 adds logical components for **create_tool validation**, **YAML append**, and relies on **noderunner** + **SSH** to run `docker` on the node. No new network service on the node beyond existing SSH and Docker Engine.

<p align="center"><img src="diagrams/c4-container.png" alt="C4 C2 — Containers EP-009" /></p>

**Source:** [c4-container.puml](diagrams/c4-container.puml) (C4-PlantUML). To regenerate PNG: `plantuml -tpng diagrams/c4-container.puml` from this directory.

---

## Module boundaries

| Layer / package (proposed) | Responsibility | Depends on |
|----------------------------|----------------|------------|
| `internal/tools` | `Tool` interface; `create_tool` implementation; register with `Registry` in `cmd/pa` | `toolcatalog`, `config`, filesystem |
| `internal/toolcatalog` (extend) | Parse single tool; append list item to YAML; thread-safe or single-writer update of in-memory `Catalog` | `os`, `yaml` |
| `internal/noderunner` | Unchanged contract: after substitution, `cmdsafe` + allowlist + SSH `Exec` | `internal/ssh`, `internal/allowlist` |
| `internal/config` (extend) | Load optional **secret detection patterns** for [REQ-09.017](ep-requirements.md#non-functional-requirements) | existing validation |
| `internal/core` | Wire `create_tool`; refresh tool list / pre-selection subset after successful create (implementation detail in stage 7) | `tools`, `toolcatalog`, `toolindex` |

**Rule:** `create_tool` MUST NOT bypass `noderunner` for remote execution: new tools still use `run_on_node` with the same allowlist model (operator extends allowlist for expected `docker run` lines).

---

## Components and interfaces

| Component | Responsibility | Key interface / contract | Requirements |
|-----------|----------------|---------------------------|--------------|
| **Core conversation handler** | Invokes native tools by name; passes catalog-derived tool defs to LLM | Existing `executeOneToolCall` path | [REQ-09.008](ep-requirements.md#tool-creation), [REQ-09.013](ep-requirements.md#tool-creation) |
| **Tools registry** | Registers `create_tool` + `run_on_node` | `tools.Tool` / `Registry` | [REQ-09.008](ep-requirements.md#tool-creation) |
| **CreateTool handler** | Validates params; whitelist; duplicate id; secret patterns; append YAML; update runtime `Catalog` | `Run(ctx, params) (string, error)` | [REQ-09.009](ep-requirements.md#tool-creation)–[REQ-09.012](ep-requirements.md#tool-creation), [REQ-09.017](ep-requirements.md#non-functional-requirements) |
| **Template whitelist** | Prefix check: `docker run --rm --network bridge` or `docker run --rm --network none` | Pure function or small package | [REQ-09.009](ep-requirements.md#tool-creation) |
| **YAML append** | Atomic append of one tool list entry to `paths.tool_catalog_path` | Serialize `toolcatalog.Tool` to YAML | [REQ-09.011](ep-requirements.md#tool-creation) |
| **Runtime catalog** | `Catalog.Tools[id] = t` after successful write | Mutex if accessed from async paths | [REQ-09.012](ep-requirements.md#tool-creation) |
| **Node runner** | Executes substituted template via SSH | `RunOnNode(ctx, nodeID, command)` | [REQ-09.001](ep-requirements.md#docker-sandbox-execution)–[REQ-09.004](ep-requirements.md#docker-sandbox-execution), [REQ-09.018](ep-requirements.md#docker-sandbox-execution) |
| **SSH client** | `session.Run(command)` | Existing `internal/ssh` | [REQ-09.001](ep-requirements.md#docker-sandbox-execution), [REQ-09.018](ep-requirements.md#docker-sandbox-execution) |
| **Docker on node** | Interprets CLI; applies `--network`, `--memory`, `--cpus`; enforces isolation | Operator-built `pa-sandbox:*` images | [REQ-09.005](ep-requirements.md#docker-sandbox-execution)–[REQ-09.007](ep-requirements.md#docker-sandbox-execution) |
| **Tool index (optional refresh)** | After create, new tool available for pre-selection | Embed new `index_text` + id or defer to restart | [REQ-09.012](ep-requirements.md#tool-creation), [REQ-09.014](ep-requirements.md#non-functional-requirements) (performance) |

Sandbox **resource flags** ([REQ-09.002](ep-requirements.md#docker-sandbox-execution)–[REQ-09.004](ep-requirements.md#docker-sandbox-execution)) are part of the **catalog tool template** (or enforced by a thin wrapper): the design assumes templates include `--memory="256m"`, `--cpus="0.5"`, and a 30s bound via `timeout` or Docker’s stop timeout—exact enforcement is specified in implementation plan so integration tests can assert the remote command ([AC-09.002](ep-acceptance-criteria.md#ac-09-002)–[AC-09.004](ep-acceptance-criteria.md#ac-09-004)).

---

## Data models

### Tool definition (append)

Same schema as existing [tool catalog](ep-requirements.md#tool-creation): `id`, `index_text`, `template`, `node_id`, `arguments`, optional `system_prompt` / `hermes_prompt` / `triggers`. Persisted as a new list element under `tools:` in `tools.yaml`.

### create_tool parameters (native tool)

| Field | Type | Required | Notes |
|-------|------|----------|--------|
| `id` | string | yes | Must not collide with existing catalog id |
| `index_text` | string | yes | For vector index and LLM description |
| `template` | string | yes | Whitelisted `docker run` prefix |
| `node_id` | string | yes | Must exist in `config.Nodes` |
| `arguments` | array (optional) | no | Serialized per `toolcatalog.ArgumentRule` |
| `system_prompt` | string | no | |

### Config extension (secret detection)

| Field | Purpose |
|-------|---------|
| `tools.create_tool_secret_patterns` (or under `log_redaction`-style list) | List of regex strings; if any matches concatenated tool fields, reject ([REQ-09.017](ep-requirements.md#non-functional-requirements)) |

Exact JSON shape is fixed in implementation planning; must fail fast at load if invalid.

### State transition (create_tool)

1. Validate params → whitelist → duplicate id → secret patterns.  
2. Append YAML → update `Catalog.Tools`.  
3. Return success string to LLM.  
4. (Optional) Queue tool index embedding for new id.

---

## Error handling

| Failure | Behaviour | Requirement |
|---------|-----------|-------------|
| Invalid whitelist prefix | Error string to LLM; no file write | [REQ-09.009](ep-requirements.md#tool-creation) |
| Duplicate `id` | Typed error; no file write | [REQ-09.010](ep-requirements.md#tool-creation) |
| Secret pattern match | Error; no file write | [REQ-09.017](ep-requirements.md#non-functional-requirements) |
| YAML write failure | Error; in-memory catalog unchanged (transactional discipline) | [REQ-09.011](ep-requirements.md#tool-creation), [REQ-09.012](ep-requirements.md#tool-creation) |
| Remote `docker run` failure | Existing noderunner / escalation behaviour; stdout/stderr surfaced per existing policy | [REQ-09.001](ep-requirements.md#docker-sandbox-execution)–[REQ-09.004](ep-requirements.md#docker-sandbox-execution) |
| Network-none probe fails in test | Pass [AC-09.018](ep-acceptance-criteria.md#ac-09-018); if unexpected pass, fail build | [REQ-09.018](ep-requirements.md#docker-sandbox-execution) |

---

## Testing strategy

Aligned with [strategy.md](../../strategy.md):

- **Unit:** whitelist parser, duplicate check, secret regex, YAML append (temp dir), coverage per [REQ-09.016](ep-requirements.md#non-functional-requirements).
- **Integration:** SSH testbed (existing pattern): run tool with `--network bridge` vs `--network none`; assert command flags; for none, run probe command inside container ([AC-09.018](ep-acceptance-criteria.md#ac-09-018)).
- **Manual / operator:** verify `pa-sandbox:python`, `:node`, `:base` images on NAS per [REQ-09.005](ep-requirements.md#docker-sandbox-execution)–[REQ-09.007](ep-requirements.md#docker-sandbox-execution).

---

## Risks and trade-offs

| Risk | Mitigation |
|------|------------|
| LLM emits dangerous but whitelisted `docker run` | Prefix whitelist + allowlist on node + secret patterns; operator controls allowlist file |
| YAML append corruption | Atomic write (write temp + rename) or file lock; implementation in stage 7 |
| Tool index stale after create | Explicit refresh or document “available next turn after embed”; meets minimum [REQ-09.012](ep-requirements.md#tool-creation) for in-memory catalog |
| `network none` misconfigured on node | Integration test fails; operator fixes Docker |

---

## Requirement traceability

Every requirement in [ep-requirements.md](ep-requirements.md) is covered by the design below.

| ID | Design coverage |
|----|-----------------|
| [REQ-09.001](ep-requirements.md#docker-sandbox-execution) | Bridge templates → SSH command includes `--network bridge`; Core + noderunner |
| [REQ-09.002](ep-requirements.md#docker-sandbox-execution) | Template includes `--memory="256m"` (enforced or validated per implementation plan) |
| [REQ-09.003](ep-requirements.md#docker-sandbox-execution) | Template includes `--cpus="0.5"` |
| [REQ-09.004](ep-requirements.md#docker-sandbox-execution) | Context timeout / `timeout` wrapper in template or runner |
| [REQ-09.005](ep-requirements.md#docker-sandbox-execution) | Operator image `pa-sandbox:python`; documented in deploy artefact |
| [REQ-09.006](ep-requirements.md#docker-sandbox-execution) | Operator image `pa-sandbox:node` |
| [REQ-09.007](ep-requirements.md#docker-sandbox-execution) | Operator image `pa-sandbox:base` |
| [REQ-09.018](ep-requirements.md#docker-sandbox-execution) | Templates with `--network none` + integration probe ([AC-09.018](ep-acceptance-criteria.md#ac-09-018)) |
| [REQ-09.008](ep-requirements.md#tool-creation) | Native tool params schema in `internal/tools` |
| [REQ-09.009](ep-requirements.md#tool-creation) | CreateTool whitelist validation |
| [REQ-09.010](ep-requirements.md#tool-creation) | Duplicate check against `Catalog.Tools` |
| [REQ-09.011](ep-requirements.md#tool-creation) | Append to `tools.yaml` |
| [REQ-09.012](ep-requirements.md#tool-creation) | Mutate in-memory `Catalog` |
| [REQ-09.013](ep-requirements.md#tool-creation) | Return string to LLM from `create_tool.Run` |
| [REQ-09.014](ep-requirements.md#non-functional-requirements) | Measured in integration test environment |
| [REQ-09.015](ep-requirements.md#non-functional-requirements) | Timed integration / benchmark hook |
| [REQ-09.016](ep-requirements.md#non-functional-requirements) | `make check` coverage threshold on new packages |
| [REQ-09.017](ep-requirements.md#non-functional-requirements) | Config-driven regex list in CreateTool path |

---

## Traceability

- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
- **Strategy:** [../../strategy.md](../../strategy.md)
