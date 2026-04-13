# Awesome Harness Engineering — PA briefing note

| Field | Content |
|-------|---------|
| **Type** | External curated resource (awesome-list), not an official PA specification |
| **Source** | [github.com/ai-boost/awesome-harness-engineering](https://github.com/ai-boost/awesome-harness-engineering) |
| **List license** | CC0 (README content is a link index; each linked item has its own license upstream) |
| **Analysis date** | 2026-04-13 |
| **Branch / snapshot** | `main` (no commit SHA pinned here; pin a SHA in an epic or PR for deep comparisons) |

---

## Agent quick index

**Keywords:** `harness engineering`, agent loop, tool design, MCP, A2A, context compaction, verification, evals, observability, HITL, sandbox, permissions, memory, orchestration.

**Use this doc when:** you need to explain or design **scaffolding** around the LLM in PA (not “which model to pick”, but context, tools, checks, logs, rights); you want external essays and references on harness primitives; you want a **prioritized idea list for PA** (see §4).

**PA touchpoints:** `AGENTS.md`, `internal/core`, `internal/tools`, `internal/toolcatalog`, `internal/llmrouter`, `internal/llmlog`, `internal/allowlist`, `internal/httpsafety`, `internal/memory`, `internal/vector`, `internal/scheduler`, `ai-sdlc/specification/skills/`, epics EP-001, EP-004, EP-006, EP-007.

---

## 1. What this resource is

This is **not a single article**; it is a public **awesome-list**: curated links to essays, product docs, frameworks, and tools on **harness engineering** — designing **scaffolding** around an agent.

The list authors define the discipline as: *context, tool interfaces, planning, verification, memory, sandboxes* — what models often lack on real tasks without support. The focus is on the **harness**, not model weights; the list assumes that as models improve, **some** harness components may simplify or disappear.

The repository also includes `templates/`, `assets/`, and example `AGENTS.md` / `CLAUDE.md` files illustrating “instructions live in the repo”.

---

## 2. Upstream README outline (section map)

Below are **logical blocks** of the upstream README table of contents (exact section titles may shift across commits).

| Section | What it covers |
|---------|----------------|
| **Foundations** | Why harness matters as a discipline: OpenAI (Codex loop, long-horizon), Anthropic (agents, tools, evals), Google ADK, Martin Fowler (context + architectural constraints + “entropy” management), LangChain, case studies such as Azure SRE agent, Red Hat, and others. |
| **Design primitives → Agent loop** | ReAct, loop iteration breakdown, LangGraph, middleware, extended thinking in APIs, etc. |
| **Planning & task decomposition** | Plan-and-execute, planning artifacts, multi-agent topologies, handoffs as distributed systems. |
| **Context delivery & compaction** | Context engineering, API compaction, caching, RAG as tool calls rather than preprocessing only. |
| **Tool design** | Schemas, errors, structured output; MCP annotations as a risk vocabulary; skill evals. |
| **Skills & MCP** | MCP, A2A, AG-UI, skills frameworks, MCP transports, integrations. |
| **Permissions & authorization** | Least privilege, OWASP LLM06, SDK permissions, enterprise policy fabrics, IETF drafts. |
| **Memory & state** | MemGPT-style stacks, mem0, Zep, memory policy, graphs, recoverability. |
| **Task runners & orchestration** | LangGraph, ADK, LiteLLM, Temporal, Vercel AI SDK, and similar. |
| **Verification & CI integration** | Evals in CI, promptfoo, AgentBench, regressions for non-deterministic behaviour. |
| **Observability & tracing** | OTel GenAI, Langfuse, Phoenix, logging of call chains. |
| **Debugging & DX** | Trace analysis, agent failure taxonomies. |
| **Human-in-the-loop** | Interrupt / approve patterns, scaling the human role to “on the loop”. |
| **Reference implementations** | Tutorial repos, adjacent awesome lists. |
| **Security, sandbox, evals, templates** | Separate blocks with links and templates. |

---

## 3. Mapping to PersonalAssistant

The list is **not about PA**; the table below shows **where list ideas already show up in PA** and reasonable next directions.

| Harness primitive (list vocabulary) | In PA today | Notes |
|-------------------------------------|-------------|--------|
| **Encoded instructions** | `AGENTS.md`, `ai-sdlc/specification/skills/`, requirements/design under `ai-sdlc-artefacts/epics/` | SDLC skills ≠ runtime skills; see [pa-runtime-skills-tools](../pa-runtime-skills-tools/pa-runtime-skills-tools.md). |
| **Tool surface + schemas** | `internal/tools` (`Tool`, `ValidateParams`), `internal/toolcatalog`, runtime skills roadmap in that same note | Narrow surface and explicit contracts are core harness design. |
| **Permissions / least privilege** | `internal/allowlist`, `internal/httpsafety`, escalation policies (EP-006) | Not a full enterprise PDP/PEP — deliberate minimalism. |
| **Memory & retrieval** | `internal/memory`, `internal/vector`, tool pre-selection | No separate “graph memory” layer as in some linked work. |
| **Orchestration** | `internal/core` (conversation, LLM, tools), `internal/scheduler`, `internal/llmrouter` | Single-process orchestrator, not a mesh of independent agents. |
| **Verification / CI** | `make check`, `.github/workflows/ci.yml`, `./bin/validate`, integration tests | Aligns with the list’s emphasis on tests as a trust lever. |
| **Observability** | `slog`, `internal/llmlog`, redaction; EP-007 (correlation, metrics) | Partial overlap with the awesome-list Observability section. |
| **MCP / A2A** | Not a universal bus in the PA product; list links are for comparison when integrating | “Whether PA needs MCP” is a separate decision from reading the list. |

---

## 4. Ideas worth applying in PA

These items synthesize themes from the [upstream awesome-list](https://github.com/ai-boost/awesome-harness-engineering) and related harness literature it points to. They are **suggestions**, not backlog commitments. Prioritize with existing epics and KISS.

### 4.1 High priority (strong fit or high leverage)

| # | Idea | Rationale / PA anchor |
|---|------|------------------------|
| 1 | **End-to-end observability for one user turn** | Harness literature stresses traceable prompt → tool → outcome paths for nondeterministic models. PA: complete **EP-007** (correlation ID in `slog` + `internal/llmlog`, consistent field semantics). |
| 2 | **Test volume, thresholds, and run reliability** | Same sources emphasize tests over model swaps; flaky tests erode trust in any automated loop. PA: keep `make check` / CI deterministic; avoid expanding integration tests without isolation. |
| 3 | **Tool design as “agent UX”** | Clear names, schemas, and **actionable error returns** so the model can recover. PA: enforce on every new tool in `internal/tools` / catalog (`ValidateParams`, safe error strings, no secrets in errors). |
| 4 | **Explicit permission boundaries** | OWASP LLM06 / “Beyond permission prompts” style thinking. PA: extend **narrow** surfaces first (`internal/allowlist`, `internal/httpsafety`) before new high-risk tools. |
| 5 | **Encoded instructions in the repo** | Fowler-style: judgment lives in versioned artifacts, not chat. PA: maintain `AGENTS.md`, SDLC skills, epics; avoid duplicating rules only in conversation. |

### 4.2 Medium priority (selective adoption)

| # | Idea | Rationale / PA anchor |
|---|------|------------------------|
| 6 | **Middleware-style cross-cutting hooks** | Intercept before/after model and tool calls for policy, redaction, retries, logging without tangling `internal/core`. PA: thin wrappers or a small hook API if `handler` branches grow. |
| 7 | **Eval / regression strategy for agent behaviour** | Separate **capability** probes (low pass rate, for improvement) from **regression** gates (near 100%, for protection); prefer deterministic checks before expensive LLM-as-judge. PA: `./bin/validate` + integration tests; add targeted harness only where needed. |
| 8 | **Progressive disclosure for runtime skills (optional variants)** | Canonical AgentSkills flow: always-on **catalog** (name + description) then full `SKILL.md` on demand. PA **already** does **harness-driven** disclosure: vector search + `max_skills_per_turn` + full playbook for selected packages (`selectSkillPackages` in `internal/core/handler.go`). **Further ideas:** optional compact catalog block in system prompt; on-demand `references/`; model-requested activation—only if product needs them. |
| 9 | **Temporal context in the harness** | Deadlines and “current time” in loop context improve time-sensitive behaviour. PA: consider explicit time/deadline lines in system or tool context for scheduler- or calendar-heavy flows. |

### 4.3 Lower priority or product-scope change

| # | Idea | When it matters |
|---|------|-----------------|
| 10 | **MCP (or A2A) as a primary integration bus** | If PA must plug into many external tool servers or cross-agent topologies; otherwise optional. |
| 11 | **Heavy orchestration (Temporal, large multi-agent mesh, graph memory stores)** | If PA grows into a distributed agent platform; overkill for a single-operator assistant today. |
| 12 | **Self-tuning closed loops on metrics** (e.g. auto-adjust gates from dashboards) | Risk of optimizing proxy metrics; PA already has simpler automation (CI, provider escalation). Add only with governance. |

### 4.4 Checklist when adding a risky capability

Derived from the list’s security / tool / verification themes: (1) tool schema and validation, (2) clear error contract, (3) least privilege and allowlists where applicable, (4) logs without secrets, (5) tests + epic traceability, (6) operator docs or `AGENTS.md` update.

---

## 5. How to use the list in practice

1. Treat it as a **topic menu**, not a mandatory stack: pick a subtopic (e.g. compaction or evals in CI) and follow the **primary source** linked from the upstream README.
2. Do **not** blindly copy scale from third-party case studies (e.g. dozens of nightly suites); adopt **principles** (feedback loops, deterministic test runs).
3. When adding a risky capability to PA, run a short checklist: tool schema, error surfaces, permissions, logs without secrets, tests, epic documentation.

---

## 6. Limits of this briefing

- The awesome-list **changes** with commits; this file captures **concepts** and a **map**, not a mirror of every URL.
- Upstream README links go to **third parties**; verify license, freshness, and security before adoption.
- This document does **not** replace PA epics and skills; it is external reference for harness design only.

---

## 7. Related material in this repository

- [Analytics README](../README.md) — report index and external project tables.
- [PA runtime skills vs tools](../pa-runtime-skills-tools/pa-runtime-skills-tools.md) — runtime vs SDLC skills, Hermes, tools (note: that note is partly in Russian).
- Epics: EP-001 (logging), EP-004 (tools/traces), EP-006 (escalation), EP-007 (observability).
