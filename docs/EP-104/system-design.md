# System Design: EP-104 PersonalAssistant MVP

**Epic:** EP-104 (Spexus)  
**Requirements:** [REQUIREMENTS.md](REQUIREMENTS.md)  
**Research:** [research.md](research.md) — technology choices, MVI, iteration plan, risks  
**Version:** 1.0  

---

## Table of contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Components and interfaces](#3-components-and-interfaces)
4. [Data models](#4-data-models)
5. [Error handling](#5-error-handling)
6. [Testing strategy](#6-testing-strategy)

---

## 1. Overview

Single-binary Go application (Option B from [research §5 Proposed design](research.md#5-section-4-proposed-design-to-be)): one process with clear boundaries between Telegram adapter, core orchestration, memory store, vector index, LLM abstraction, scheduler, tools, SSH client, and LLM logging. Target: Synology DS220+ (Intel Celeron J4025, x86_64), deployed as one Docker container ([REQ-002](REQUIREMENTS.md#interface-and-deployment)). Config-driven: nodes, LLM providers (ordered list, fallback on failure), memory/log paths, path to scheduled tasks file, and per-node allowlist file path are validated at startup ([REQ-003](REQUIREMENTS.md#nodes-and-ssh)).

---

## 2. Architecture

- **Ingestion:** Telegram bot ([go-telegram/bot](https://github.com/go-telegram/bot), polling for MVP) → core. See [REQ-001](REQUIREMENTS.md#interface-and-deployment).
- **Core:** Receives messages, loads config, calls LLM via provider interface ([REQ-008](REQUIREMENTS.md#llm-and-logging)), reads/writes long-term memory ([REQ-006](REQUIREMENTS.md#memory-and-indexing)), runs semantic search over vector index ([REQ-007](REQUIREMENTS.md#memory-and-indexing)), invokes tools ([REQ-010](REQUIREMENTS.md#scheduler-and-tools)) and scheduler ([REQ-009](REQUIREMENTS.md#scheduler-and-tools)), and (when needed) runs allowed commands on nodes via SSH ([REQ-004](REQUIREMENTS.md#nodes-and-ssh), [REQ-005](REQUIREMENTS.md#nodes-and-ssh), [REQ-013](REQUIREMENTS.md#nodes-and-ssh)).
- **Storage:** Long-term memory = the assistant’s single store: directory of markdown files in calendar structure year/month/day, with hierarchical summarization (day → month → year) from LLM logs, tool results, and scheduler events ([REQ-019](REQUIREMENTS.md#memory-and-indexing), [REQ-020](REQUIREMENTS.md#memory-and-indexing)); not partitioned by interlocutor ([REQ-018](REQUIREMENTS.md#memory-and-indexing)). Vector store = pluggable interface; default in-process implementation (see [§3 Components](#3-components-and-interfaces) and [research §4.1 Vector store options](research.md#41-vector-store-options-req-007-pluggable)).
- **Outbound:** SSH client (`golang.org/x/crypto/ssh`) to nodes as dedicated PA user only; commands restricted by per-node allowlist (pattern/regex). No shell with untrusted input; exec-style args only.
- **Observability:** LLM logging subsystem writes request/response to configurable path in JSON Lines ([REQ-014](REQUIREMENTS.md#llm-and-logging), [REQ-015](REQUIREMENTS.md#llm-and-logging)).

C4: see [REQUIREMENTS.md — C4 Diagrams](REQUIREMENTS.md#c4-diagrams). No extra containers for MVP ([REQ-012](REQUIREMENTS.md#extensibility-and-architecture)).

---

## 3. Components and interfaces

| Component | Responsibility | Key interface / tech |
|-----------|----------------|----------------------|
| **Config** | Load and validate JSON (nodes, LLM providers, paths, allowlist paths, scheduled_tasks_path). | Validated structs; fail startup on error ([REQ-003](REQUIREMENTS.md#nodes-and-ssh)). |
| **Telegram adapter** | Poll Bot API, map updates to core messages, send replies. | go-telegram/bot; config: token, path to users file (user_id, role: user or admin). |
| **Core** | Orchestrate conversation: memory read, vector search, LLM call, tool/scheduler dispatch, SSH when needed. | Single entry per user message; uses all below. |
| **Memory (MD store)** | Read/write the assistant’s single memory store (markdown files in calendar layout year/month/day). Hierarchical summarization (day → month → year) from LLM logs, tool results, scheduler events. | File system; calendar structure ([REQ-019](REQUIREMENTS.md#memory-and-indexing)); summary sources ([REQ-020](REQUIREMENTS.md#memory-and-indexing)); not partitioned by interlocutor ([REQ-006](REQUIREMENTS.md#memory-and-indexing), [REQ-018](REQUIREMENTS.md#memory-and-indexing)). |
| **Vector store** | Pluggable: index embeddings from memory (and optionally conversation), semantic search. | Interface with default impl; see [research §4.1](research.md#41-vector-store-options-req-007-pluggable) and [§4.2](research.md#42-deep-analysis-three-vector-store-options-decades-long-retention-target-hardware). |
| **LLM provider** | Abstract completion: e.g. `Complete(ctx, messages, opts) (response, usage, err)`. | Ordered list in config; first available used; fallback to next on failure ([REQ-008](REQUIREMENTS.md#llm-and-logging)). |
| **Scheduler** | Run tasks at times defined in a separate tasks file. | [robfig/cron/v3](https://github.com/robfig/cron); tasks loaded from path in config (JSON array); execution calls tools or Telegram notification ([REQ-009](REQUIREMENTS.md#scheduler-and-tools)). |
| **Tools** | Extensible: Name, Description, ParamsSchema, Run(ctx, params). | In-process registry at startup; config can enable/parameterise ([REQ-010](REQUIREMENTS.md#scheduler-and-tools), [REQ-011](REQUIREMENTS.md#extensibility-and-architecture)). |
| **SSH client** | Connect to nodes as dedicated PA user; execute only allowlisted commands. | `golang.org/x/crypto/ssh`; one identity per node ([REQ-013](REQUIREMENTS.md#nodes-and-ssh)); allowlist loaded from file path per node ([REQ-005](REQUIREMENTS.md#nodes-and-ssh)). |
| **LLM logging** | On each LLM call, write request and response to configured path. | JSON Lines; configurable destination and parseable format ([REQ-014](REQUIREMENTS.md#llm-and-logging), [REQ-015](REQUIREMENTS.md#llm-and-logging)). |

### Vector store choice (pluggable, [REQ-007](REQUIREMENTS.md#memory-and-indexing))

- **Default (with CGO):** SQLite + sqlite-vec for single-file, durable, vector+FTS storage when decades-long retention is priority; use optional build tag or separate build ([research §4.2 summary](research.md#summary-and-recommendation-for-decades-long-retention)).
- **Optional (no CGO):** [vecgo](https://github.com/hupe1980/vecgo) (HNSW) or [chromem-go](https://github.com/philippgille/chromem-go). Persistence: vecgo via Gob file; chromem-go via explicit export/import. Index built from MD content; embeddings from LLM provider. For long-term retention: backup index file and document rebuild-from-MD if needed. See [research §4.2](research.md#42-deep-analysis-three-vector-store-options-decades-long-retention-target-hardware).

---

## 4. Data models

- **Config:** Nodes (host, dedicated_user, auth, command_allowlist_path), llm_providers (ordered list), paths (memory_dir, log_path, vector_index_path, scheduled_tasks_path). Scheduled tasks in separate JSON file. See [implementation plan — Config file](implementation-plan.md#config-file-json) and [research §6 MVI](research.md#6-section-5-minimum-viable-increment-mvi).
- **Long-term memory:** The assistant’s single store: markdown files under memory_dir in a calendar structure year/month/day ([REQ-019](REQUIREMENTS.md#memory-and-indexing)); not subdivided by interlocutor ([REQ-018](REQUIREMENTS.md#memory-and-indexing)). Hierarchical summarization: day summary (from that day’s activity), month summary (from day summaries), year summary (from month summaries). Summary inputs include LLM logs, tool execution results, and scheduler events ([REQ-020](REQUIREMENTS.md#memory-and-indexing)). Optional approval by owner before persisting summaries. Content and chunking strategy are part of implementation ([REQ-006](REQUIREMENTS.md#memory-and-indexing)).
- **LLM log entry:** request_id, timestamp, direction (request|response), payload (messages, model, response, usage, duration) in JSON Lines ([REQ-015](REQUIREMENTS.md#llm-and-logging)).
- **Tool:** name, description, params_schema (e.g. JSON Schema), Run(ctx, params) ([REQ-010](REQUIREMENTS.md#scheduler-and-tools)).

---

## 5. Error handling

- **Config validation:** On load failure or invalid allowlist/node/LLM config → log clear error, exit non-zero; do not start serving ([REQ-003](REQUIREMENTS.md#nodes-and-ssh)).
- **LLM/Telegram/SSH errors:** Log; return user-facing error or retry per policy; do not expose internal details to the user.
- **Vector store:** If index load fails, optionally start with empty index and log; or fail startup if persistence is required by config.
- **SSH:** Only allowlisted commands; exec with args (no shell). On connection or exec failure, log and report to core; no fallback to other users ([REQ-013](REQUIREMENTS.md#nodes-and-ssh)). See [research §8 Risks](research.md#8-section-7-risks-and-mitigations).

---

## 6. Testing strategy

- **Unit:** Config validation, allowlist matching, tool registry, LLM logging format, memory path resolution.
- **Integration:** Core + one LLM provider (e.g. mock or local Ollama), core + in-memory vector store, Telegram adapter with test bot or mock.
- **E2E (optional for MVP):** Single flow: message in → memory/vector + LLM → reply out; optional SSH to test node with allowlisted command.
- **Deploy:** Build linux/amd64 Docker image; run on target or equivalent (e.g. DS220+ or same arch); verify config load, one conversation, and log output.

Iteration plan and risks: [research §7 Iteration plan](research.md#7-section-6-iteration-plan), [research §8 Risks and mitigations](research.md#8-section-7-risks-and-mitigations).
