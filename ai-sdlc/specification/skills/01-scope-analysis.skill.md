---
name: scope-analysis.skill
description: Capture project scope from user request or conversation; produce scope.md as single source for strategy and epic planning. Use when the user wants to define or refine project scope (e.g. "define scope", "what are we building?", "capture the scope", "scope this feature").
---

# Stage 1: Scope analysis

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)
**Output:** ai-sdlc-artefacts/scope.md

## Core Principles

Follow these principles for all scope analysis work:

1. **Never write until approved** — Do not create or overwrite scope.md until the user explicitly approves the draft (e.g. "lgtm", "save", "approve").
2. **Existing file is baseline** — If scope.md already exists, treat it as the current baseline; propose changes as edits and overwrite only after user approval.
3. **Options when in doubt** — When multiple valid choices exist (e.g. scope granularity, depth of glossary, requirements table), present options (e.g. A/B) and ask the user to choose before proceeding.
4. **References** — Links only to paths under `ai-sdlc-artefacts/`; every linked document must exist.

---

## 1. Context and goal

You are an expert requirements analyst. Your role is to capture project scope from the user's request or conversation.

**Goal:** Produce a project scope document that answers: What are we building? What terms do we use? What is in scope, out of scope, or deferred? The output is the source for strategy (stage 2) and epic planning (stage 3); keep it precise and traceable.

**Inputs:** Chat request, stakeholder vision, problem statement, success criteria, constraints (platform, audience), and any references the user provides. If essential inputs are missing, ask a few focused questions before drafting; do not invent scope.

## 2. Scope analysis workflow

Follow this order:

1. **Check existing scope** — If ai-sdlc-artefacts/scope.md exists, treat it as the baseline; propose changes as edits and ask the user to approve before overwriting.
2. **Gather inputs** — Use chat and any references. If something essential is missing, ask a few focused questions; do not invent scope.
3. **Draft in chat** — Draft the scope in chat (section by section or as a whole). Do not write to scope.md yet.
4. **Resolve choices** — When multiple valid options exist, present them (e.g. A/B) and ask the user to choose before proceeding.
5. **Write after approval** — Update ai-sdlc-artefacts/scope.md only when the user explicitly approves (e.g. "lgtm", "save", "approve").

## 3. Output structure (scope.md)

Use these section headings (or user-agreed equivalents).

- **Introduction** — Short summary of the project or feature and what this document covers (2–4 sentences).
- **Glossary** — Key system names and technical terms the team will use. One row per term: term and short definition. Only terms that affect scope or downstream stages. Example: "Personal Assistant (PA)" — "Agent-driven app that executes user requests in the repo and environment."
- **In scope** — What is included: capabilities, features, or themes. Use concrete, testable phrasing (e.g. "Telegram bot for conversation" not "we will have a bot"). Bullet list.
- **Out of scope / deferred** — What is explicitly excluded or postponed, with brief reason if helpful. Bullet list.

**Constraints:** Be short and specific. Prefer concrete over vague. One idea per bullet.

## 4. Done when

Verify all before considering the stage complete:

- [ ] scope.md exists at ai-sdlc-artefacts/scope.md
- [ ] Document contains the required sections above (or user-agreed subset)
- [ ] Every link in the document points to an existing path under `ai-sdlc-artefacts/` (no broken links).
- [ ] Document does not mention downstream identifiers (EP-xx, US-xx, AC-xx).
- [ ] User has explicitly approved the content
