# Research: EP-104 PersonalAssistant MVP

**Purpose:** Technical discovery — options, comparison, recommendations, risks, and mitigations to inform architecture and delivery strategy.  
**Pipeline:** [PIPELINE.SPEC.md](PIPELINE.SPEC.md)  
**Previous:** [01-02-requirements.md](01-02-requirements.md)  
**Next:** [04-system-design.md](04-system-design.md)  
**Related:** [04-system-design.md](04-system-design.md), [05-delivery-strategy.md](05-delivery-strategy.md)

---

## Table of contents

1. [Introduction](#1-introduction)
2. [Section 1: Repository and components](#2-section-1-repository-and-components)
3. [Section 2: As-Is architecture](#3-section-2-as-is-architecture)
4. [Section 3: Readiness for target state](#4-section-3-readiness-for-target-state)
5. [Section 4: Proposed design (To-Be)](#5-section-4-proposed-design-to-be)  
   - [4.1 Vector store options](#41-vector-store-options-req-007-pluggable)  
   - [4.2 Deep analysis: three options (decades, DS220+)](#42-deep-analysis-three-vector-store-options-decades-long-retention-target-hardware)
6. [Delivery strategy](#6-delivery-strategy)
7. [Risks and mitigations](#7-risks-and-mitigations)
8. [Sources](#8-sources)

---

## 1. Introduction

**Objective:** Choose technical options for the PersonalAssistant MVP (EP-104): Telegram bot, Go core in Docker on Synology DS220+, SSH to nodes with an explicit security model, long-term memory in markdown, vector search, swappable LLMs, scheduler, extensible tools, and LLM request/response logging.

**Context:** Requirements are in [01-02-requirements.md](01-02-requirements.md) ([REQ-001–REQ-020](01-02-requirements.md#requirement-index)). This research follows the epic-researcher workflow (PROMPT-008); target platform: DS220+ (x86_64, limited CPU/RAM).

---

## 2. Section 1: Repository and components

**Current state:** The PersonalAssistant repo contains only the requirements spec and `.gitignore`. No code or prior architecture — green field. The epic and [Glossary in 01-02-requirements.md](01-02-requirements.md#glossary) define the system boundary: Telegram adapter, core (orchestration, LLM, tools), MD store, vector index, scheduler, SSH client, LLM providers, logging subsystem.

---

## 3. Section 2: As-Is architecture

No existing architecture. Target behaviour is defined by the epic description and C4 diagrams; requirements [REQ-001–REQ-020](01-02-requirements.md#requirement-index) are the source of truth (see [Requirements](01-02-requirements.md#requirements) and [Requirement index](01-02-requirements.md#requirement-index)).

---

## 4. Section 3: Readiness for target state

| Area | Options | Readiness |
|------|---------|-----------|
| Telegram | go-telegram/bot, mr-linch/go-tg, telebot, go-telegram-bot-api | Mature libraries, Bot API 7.x+ |
| SSH + security | stdlib `golang.org/x/crypto/ssh`, allowlist by pattern/regex | Exec with args, command whitelist validation |
| Vector search | chromem-go, vecgo (HNSW), SQLite+sqlite-vec (optional), pgvector (optional) | Pluggable (Glossary, REQ-012); default: in-process, no extra container |
| LLM | llmhub (multi-provider), go-ollama, llmx | Single interface, swap provider via config |
| Scheduler | robfig/cron v3 | Cron expressions and @every |
| Tools | In-process registry (interface), no go plugin | Config + code; add tools without image rebuild via config-driven registry |
| LLM logging | Structured logs (JSON Lines), configurable path | Standard approach |
| Platform | DS220+ (Intel Celeron J4025, x86_64) | Standard linux/amd64 Docker image |

---

## 5. Section 4: Proposed design (To-Be)

Three stack-level options; **Option B is recommended**.

**Option A — Minimal deps, hand-rolled**  
Telegram: low-level HTTP to Bot API. SSH: stdlib only. Vector: simple in-memory index (e.g. slice + cosine). LLM: per-provider clients behind a small interface. Pros: full control, small binary. Cons: more custom code, slower to MVP.

**Option B — Proven libraries, single binary (recommended)**  
Telegram: [go-telegram/bot](https://github.com/go-telegram/bot). SSH: `golang.org/x/crypto/ssh`, allowlist per node (patterns/regex in config). Vector: [chromem-go](https://github.com/philippgille/chromem-go) or [vecgo](https://github.com/hupe1980/vecgo). LLM: [llmhub](https://pkg.go.dev/github.com/smhanov/llmhub) or similar with a single interface (OpenAI-compatible + Ollama). Scheduler: [robfig/cron/v3](https://github.com/robfig/cron). Tools: in-process registry (interface + Register), config for name/params; no runtime plugin. Logging: JSON Lines to a configurable path (request_id, messages, model, response, tokens, duration). Pros: fast MVP, less custom code, single x86_64 build. Cons: dependency on chosen libraries (all open and maintained).

**Option C — Microservices**  
Separate containers for bot, core, vector DB. Pros: scale/replace independently. Cons: heavier deploy and config on DS220+, more resource use; overkill for MVP ([REQ-002](01-02-requirements.md#interface-and-deployment) expects a single core image).

**Choice:** Option B best fits MVP, [REQ-012](01-02-requirements.md#extensibility-and-architecture) (clear separation), and DS220+ constraints.

### 4.1 Vector store options ([REQ-007](01-02-requirements.md#memory-and-indexing), pluggable)

[REQ-007](01-02-requirements.md#memory-and-indexing) requires a vector index and semantic search over long-term memory. The [Glossary](01-02-requirements.md#glossary) defines the vector store as **pluggable** (in-memory, file, or DB). The following fits DS220+ (single container, limited RAM), [REQ-012](01-02-requirements.md#extensibility-and-architecture) (replaceable component), and MVI goal of `CGO_ENABLED=0` where possible.

| Option | Persistence | CGO | Scale (order) | Pluggable fit | Note |
|--------|-------------|-----|---------------|---------------|------|
| **chromem-go** | Optional (file/export) | No | ~10²–10⁵ docs | ✓ Default candidate | Chroma-like API, single binary, low latency. |
| **vecgo (HNSW)** | Gob serialisation | No | ~10⁴–10⁶ vectors | ✓ Default candidate | Better ANN quality; reload on startup. |
| **SQLite + sqlite-vec** | Native (one .sqlite file) | Yes (C extension) | ~10⁵–10⁶ | ✓ Alternative | Single file, vector + FTS in one DB; optional build tag. |
| **PostgreSQL + pgvector** | Native, ACID | No (client only) | ~10⁶+ | ✓ Alternative | Extra container; use only if Postgres already in stack. |
| External vector DB (Qdrant, etc.) | External | No | 10⁷+ | ✓ Future | Overkill for MVP; more ops and resources. |

### 4.2 Deep analysis: three vector-store options (decades-long retention, target hardware)

**Target hardware (DS220+):** Synology DS220+ uses Intel Celeron J4025 (2 cores, 2.0–2.7 GHz), typically 2–6 GB RAM shared with DSM and other containers, and disk (HDD or SSD). No separate vector DB container is desired; the core runs in one Docker image. Build target: `linux/amd64`, preferably `CGO_ENABLED=0` for a single static binary.

**Retention horizon:** Memory is expected to be kept for **decades**. That implies: durable persistence, recoverability from a single copy (e.g. one file or one DB), format longevity, and minimal reliance on “reload from canonical source only” without a durable index.

**Abbreviations used:**

| Term | Meaning |
|------|--------|
| **ANN** | Approximate Nearest Neighbor — algorithms that find vectors *approximately* closest to a query vector, trading some accuracy for speed. Essential for large vector sets. |
| **HNSW** | Hierarchical Navigable Small World — an ANN algorithm (graph-based) with high recall and good query speed; used by vecgo. |
| **FTS** | Full-Text Search — keyword search over text (e.g. SQLite FTS5). Combined with vector search it enables hybrid (semantic + keyword) retrieval. |
| **CGO** | Go’s mechanism to call C code. `CGO_ENABLED=0` yields a pure Go binary; CGO requires a C toolchain and can complicate cross-build and portability. |
| **Gob** | Go’s binary serialisation format (`encoding/gob`). Used by vecgo to save/load the in-memory index. |
| **ACID** | Atomicity, Consistency, Isolation, Durability — database properties that guarantee reliable, recoverable writes. |

---

#### Option 1: chromem-go

| Aspect | Detail |
|--------|--------|
| **Persistence** | Optional: export/import to file or `io.Writer`. No automatic flush; durability depends on explicit save and backup. |
| **CGO** | No. Pure Go, single binary, easy cross-compile for DS220+. |
| **Scale** | ~10²–10⁵ documents. Sufficient for personal memory over many years if chunking is moderate. |
| **Search** | In-memory similarity (no HNSW); for large N search can become slower than ANN. |

**Pros:** Chroma-like API; zero external deps; very low latency; trivial deploy (one binary); no C toolchain.  
**Cons:** Persistence is opt-in and manual — if the process exits without save, the index is lost; decades-long retention requires disciplined export/backup and a clear “where is the single source of truth” (the index file). Full index in RAM — on DS220+ with 2 GB usable, a large index competes with the rest of the stack.  
**Decades:** Not ideal by itself. Needs a strict policy: periodic export to a durable volume, versioning/backup of that file, and possibly rebuild-from-MD path if the export is lost. Without that, risk of data loss is high.

---

#### Option 2: vecgo (HNSW)

| Aspect | Detail |
|--------|--------|
| **Persistence** | Gob serialisation: the HNSW index is saved to/loaded from a file. More “first-class” than chromem-go’s export. |
| **CGO** | No. Pure Go, single binary. |
| **Scale** | ~10⁴–10⁶ vectors. Better suited than chromem-go to growth over decades (millions of chunks). |
| **Search** | HNSW (ANN) — better recall/speed trade-off than brute force at scale. |

**Pros:** Better ANN quality; integrated save/load via Gob; no CGO; single binary; scales to millions of vectors.  
**Cons:** Full index in RAM; “reload on startup” — for a large index (e.g. 10⁶ vectors), startup time and RAM spike on DS220+ can be noticeable. Gob format is Go-specific — long-term (decades) may require a migration path if Go’s Gob changes. No hybrid FTS in the library itself.  
**Decades:** Viable if: (1) Gob file is treated as a first-class asset (backed up, checksummed), (2) index size and startup time are monitored, (3) there is a documented path to rebuild index from markdown if the Gob file is lost or corrupted. Format longevity is the main uncertainty over decades.

---

#### Option 3: SQLite + sqlite-vec

| Aspect | Detail |
|--------|--------|
| **Persistence** | Native: one `.sqlite` file; all data and vector index on disk. Writes are transactional (ACID). |
| **CGO** | Yes. The sqlite-vec extension is C; linking it typically requires CGO (or a separate C build and loading the extension at runtime). Complicates `CGO_ENABLED=0` and cross-compile. |
| **Scale** | ~10⁵–10⁶ vectors; SIMD (AVX/NEON) accelerates KNN. Fits “lifetime” personal memory with growth. |
| **Search** | Vector similarity (e.g. cosine) plus optional FTS5 in the same DB — hybrid vector + keyword in one place (OpenClaw-style). |

**Pros:** One file to backup/migrate; ACID and proven durability; vector + FTS in one DB; no need to hold the full index in RAM — SQLite pages from disk; format is widely supported and likely readable for decades.  
**Cons:** C extension and CGO — build and possibly runtime depend on C toolchain and correct extension for the platform; on DS220+ (linux/amd64) usually solvable but adds complexity. Query latency is disk I/O–bound; on HDD it can be tens of ms vs in-memory.  
**Decades:** Best fit for “store once, keep decades”. Single file, standard format, transactional, easy to backup and restore. Main long-term risk is availability of sqlite-vec (or a compatible replacement) for future SQLite versions; the data itself in the `.sqlite` file remains accessible.

---

#### Summary and recommendation for decades-long retention

- **If the priority is decades-long retention and minimal operational risk:** Prefer **SQLite + sqlite-vec** as the pluggable implementation (accept CGO or an optional build tag). Single file, ACID, vector+FTS, and format longevity make it the most robust for a “memory for life” scenario on DS220+.
- **If the priority is zero CGO and simplest deploy:** Use **vecgo** as the default: persistence via Gob, better ANN than chromem-go, and a clear “backup the Gob file + optionally rebuild from MD” policy. Monitor index size and startup time on DS220+.
- **chromem-go** is acceptable for MVP or small corpora if persistence and backup are strictly enforced; for decades, it is the weakest of the three without a durable, versioned export strategy.

---

## 6. Delivery strategy

Delivery strategy (named increments, MVP stack, iteration plan, and success criteria) is defined in **[05-delivery-strategy.md](05-delivery-strategy.md)**. It is informed by the technology options and recommendations in this research (e.g. [§4 Proposed design](#5-section-4-proposed-design-to-be)).

---

## 7. Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Bot API spec changes | Pin go-telegram/bot version; test typical flows. |
| Secret leakage (tokens, SSH keys) | Secrets from env or mounted files; exclude secret config from repo. |
| Arbitrary command execution on node | Strict allowlist per node; no shell; exec with args only; review and tests. |
| Disk fill from LLM logs | Rotation by size/date; configurable retention; optional compression. |
| High RAM from vector index | Cap index size or use persistent backend with disk (chromem-go/vecgo); see §4.1. |
| CGO if sqlite-vec chosen | Optional vector backend; document build with CGO or keep default pure-Go. |
| DS220+ cannot run heavy model | Document: use light model or remote LLM; monitor container memory. |

---

## 8. Sources

- Telegram: [go-telegram/bot](https://pkg.go.dev/github.com/go-telegram/bot), [mr-linch/go-tg](https://pkg.go.dev/github.com/mr-linch/go-tg).
- SSH: [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh), [ssh-shield](https://github.com/morriswinkler/ssh-shield), [PrivX command restrictions](https://privx.docs.ssh.com/docs/ssh-command-restrictions).
- Vector: [chromem-go](https://github.com/philippgille/chromem-go), [vecgo](https://github.com/hupe1980/vecgo), [comet](https://github.com/wizenheimer/comet); [sqlite-vec](https://github.com/asg017/sqlite-vec), [pgvector](https://github.com/pgvector/pgvector).
- LLM: [llmhub](https://pkg.go.dev/github.com/smhanov/llmhub), [go-ollama](https://pkg.go.dev/github.com/eslider/go-ollama), [llmx](https://pkg.go.dev/github.com/llmx-ai/llmx).
- Scheduler: [robfig/cron/v3](https://github.com/robfig/cron).
- Plugins: [go plugin](https://pkg.go.dev/plugin), [HashiCorp go-plugin](https://pkg.go.dev/github.com/hashicorp/go-plugin), [modular plugin system](https://peerdh.com/blogs/programming-insights/building-a-modular-plugin-system-in-go).
- Synology: [DS220+ (Intel Celeron J4025, x86)](https://www.synology.com/products/refurbished/DS220%2B), [Compiling Go for Synology](https://www.afox.dev/posts/compiling-go-for-synology-nas).
