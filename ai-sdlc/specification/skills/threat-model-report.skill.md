---
name: threat-model-report
description: >-
  Produce a threat model report for PersonalAssistant (or a scoped subsystem) grounded in
  source code and operator docs. Use when the user asks for a threat model, STRIDE analysis,
  security architecture from code, or "модель угроз" derived from the repository.
---

# Threat model report (code-grounded)

**Primary output (default):** `docs/threat-model.md` (operator-facing; add an index row in [docs/README.md](../../../docs/README.md) when the user approves saving).  
**PersonalAssistant project artefact (SDLC root):** `ai-sdlc-artefacts/threat-model.md` — use when the user asks for the threat model **at project artefacts level** (alongside [audit-report.md](../../../ai-sdlc-artefacts/audit-report.md)); update [ai-sdlc-artefacts/README.md](../../../ai-sdlc-artefacts/README.md) if a new top-level artefact type is introduced.  
**Alternative output:** `ai-sdlc-artefacts/analytics/<short-topic-slug>/threat-model.md` — use if the user asks to file under `analytics/` only.

---

## 1. Goal and scope

Produce a **single markdown report** that:

- Describes **assets**, **trust boundaries**, **threat actors**, and **representative threats** for the product **as implemented**, inferred from **`cmd/`**, **`internal/`**, **`docs/`**, config contracts, and tests — **not** from wishlist prose alone.
- Maps **existing controls** in code and configuration to threats (with **file/symbol references** where practical).
- Lists **gaps and residual risks** explicitly (missing rate limits, DEBUG behaviour, third-party trust, etc.).
- Stays **factual**: do not claim controls that are not visible in code or documented operator behaviour; label assumptions as assumptions.

**When to use:** User asks for a threat model, STRIDE / attack-surface analysis from code, security baseline document, or equivalent in another language (e.g. Russian request → still write the **skill artefact in English** per project [AGENTS.md](../../../AGENTS.md); the user may ask for a Russian summary in chat only).

---

## 2. Inputs

- **Scope:** Whole product (default) or named subsystem (e.g. “SSH tools only”, “Telegram + core”, “config load path”). If ambiguous, present options and ask the user to choose.
- **Codebase:** The workspace repository (PersonalAssistant). Read **entrypoints** (`cmd/`), **security-relevant packages** (config, adapters, core handler, remote exec, logging, redaction), and **`docs/`** (`configuration.md`, `operations.md`, `docker.md`, `troubleshooting.md`).
- **Optional design context:** If [ep-system-design.md](../../../ai-sdlc-artefacts/epics/<epic-id>/ep-system-design.md) or security-related requirements exist for the scoped area, **cross-check** for intent vs implementation and note **drift**.
- **Revision:** Record `git rev-parse HEAD` at analysis time in the report header.

---

## 3. Workflow

1. **Confirm scope** — Whole repo vs subsystem; default output path (`docs/threat-model.md` vs `ai-sdlc-artefacts/analytics/...`). Ask if the user did not specify and both are plausible.
2. **Map the system from code** — Identify:
   - Process entrypoints and long-running services;
   - External systems (messengers, LLM APIs, SSH targets, filesystem paths);
   - Secret and credential **storage mechanism** (files, env bases, Docker secrets);
   - Execution paths that can lead to **command execution**, **file I/O**, or **network egress**.
3. **Define trust boundaries** — Draw or describe boundaries (e.g. untrusted user input → core → node). Prefer a **Mermaid** `flowchart` or `graph` in the report.
4. **Enumerate assets and threat actors** — At minimum: operator secrets, user messages, node integrity, logs, config integrity, provider-visible data.
5. **Analyse threats** — Use **STRIDE** (or equivalent) per trust boundary: Spoofing, Tampering, Repudiation, Information disclosure, Denial of service, Elevation of privilege. For each relevant cell, tie to **concrete code paths** or config knobs (e.g. allowlist, `cmdsafe`, redactor, `PA_LOG_LEVEL`).
6. **Controls vs gaps table** — Threat or category | In-code / in-doc control | Gap or residual risk | Severity (informal: Critical / High / Medium / Low) if appropriate.
7. **Draft in chat** — Full report using [§4 output structure](#4-output-structure).
8. **Write file** — Save only after explicit user approval (“save”, “lgtm”, “write to docs”). If the file exists, ask whether to **replace** or **version** (e.g. `threat-model-2026-03-20.md`).

**Constraints:** Do not weaken security in product code as part of this skill. Do not commit secrets or real tokens into the report. **Language:** Report body and section titles in **English**; code comments in examples in English.

---

## 4. Output structure

| Section | Content |
|--------|--------|
| **Header** | Title; **date**; **git revision** analysed; **scope** (whole product / subsystem); **output path**; pointer to `docs/` or analytics. |
| **1. System overview** | One paragraph: what the binary does, main external dependencies (from code + README). |
| **2. Architecture and trust boundaries** | Mermaid diagram + narrative; list **trust zones** (e.g. Telegram user, PA process, nodes, cloud APIs). |
| **3. Assets** | Table: asset, location / flow, sensitivity. |
| **4. Threat actors** | Short list with motivation and reach. |
| **5. STRIDE analysis** | Table or subsections per category; each row references **implementation** (`package`, `function`, or `docs/...`) when a control exists. |
| **6. Security controls (evidence-based)** | Bullet list grouped by theme (authentication, authorization, input validation, exec safety, secrets, logging, availability); each bullet should cite **where** in repo (path). |
| **7. Gaps and residual risks** | Honest list: e.g. no per-user rate limit, DEBUG logging, tool error strings to LLM, dependency/CVE posture. |
| **8. Recommendations** | Prioritised for **operator** (config hardening) vs **future development** (features); must not contradict [AGENTS.md](../../../AGENTS.md) (no scope creep without user choice). |
| **9. References** | Links to `docs/configuration.md`, `docs/operations.md`, relevant epic artefacts (relative paths from the saved file). |

**Diagrams:** Mermaid in fenced blocks; avoid reserved participant names that break `sequenceDiagram` (e.g. use `User` not `Loop`).

---

## 5. Conventions

- **Evidence over claims:** Prefer “`internal/noderunner` enforces allowlist via …” over “SSH is secure”.
- **Relative links:** From `docs/threat-model.md`, link to `./configuration.md`. From `ai-sdlc-artefacts/analytics/foo/threat-model.md`, link to `../../epics/...` as needed.
- **Cooperation:** If the user wants STRIDE only vs full report, or a different path, ask once and proceed.
- **Not a pentest:** State in the header that the document is a **design-time threat model**, not an executed penetration test.

---

## 6. Reference

- Operator context: [docs/configuration.md](../../../docs/configuration.md), [docs/docker.md](../../../docs/docker.md).  
- Project rules: [AGENTS.md](../../../AGENTS.md).  
- Similar structured outputs: [10-code-review.skill.md](10-code-review.skill.md), [project-comparison-report.skill.md](project-comparison-report.skill.md).
