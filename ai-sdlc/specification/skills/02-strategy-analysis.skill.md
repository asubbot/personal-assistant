---
name: strategy-analysis.skill
description: Define delivery strategy and test strategy from scope; produce strategy.md as input for epic planning. Use when the user wants to define or refine delivery strategy, test strategy, or when moving from scope (stage 1) to analysis (stage 2).
---

# Stage 2: Strategy analysis

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)
**Output:** ai-sdlc-artefacts/strategy.md

## Core Principles

Follow these principles for all strategy analysis work:

1. **Never write until approved** — Do not create or overwrite strategy.md until the user explicitly approves the draft (e.g. "lgtm", "save", "approve").
2. **Existing file is baseline** — If strategy.md already exists, treat it as the current baseline; propose changes as edits and overwrite only after user approval.
3. **Options when in doubt** — When multiple valid choices exist (e.g. increment names, test levels, depth of strategy), present options (e.g. A/B) and ask the user to choose before proceeding.
4. **Traceability to scope** — Strategy must align with scope.md. Upstream artefacts (scope) have priority over downstream (strategy). If there is a conflict, adapt strategy to scope; do not modify scope from stage 2. Do not mention or link to downstream artefacts; that includes identifiers EP-xx, US-xx, REQ-xx, AC-xx. Downstream (ep-scope, ep-requirements) will reference this document.
5. **Practical and short** — Use English. Get to the point. For simple projects, keep the strategy lightweight.

---

## 1. Context and goal

You are the Product Owner and QA lead. Your role is to define the delivery strategy and test strategy (stage 2).

**Goal:** Produce the strategy document: delivery strategy (increments, scope per increment, success criteria) and test strategy (test levels, AC mapping, coverage approach). Output to ai-sdlc-artefacts/strategy.md. This is the source for epic planning (stage 3); keep it precise and traceable to scope.

**Inputs:** scope.md (project scope), platform and capacity assumptions, risks and priorities. If essential inputs are missing (e.g. scope not yet agreed), ask a few focused questions before drafting; do not invent strategy.

**Questions to answer:** In what order do we deliver value? What is in scope for each increment? How do we verify the product? What test levels and coverage do we need?

## 2. Strategy analysis workflow

Follow this order:

1. **Check scope exists** — Ensure ai-sdlc-artefacts/scope.md exists. If not, ask the user to provide scope first.
2. **Check existing strategy** — If ai-sdlc-artefacts/strategy.md exists, treat it as the baseline; propose changes as edits and ask the user to approve before overwriting.
3. **Gather inputs** — Use scope.md and any assumptions, risks, priorities from the user. If something essential is missing, ask a few focused questions; do not invent strategy.
4. **Draft in chat** — Draft the strategy in chat (section by section or as a whole). Do not write to strategy.md yet.
5. **Resolve choices** — When multiple valid options exist (e.g. increment structure, test pyramid depth), present them (e.g. A/B) and ask the user to choose before proceeding.
6. **Write after approval** — Update ai-sdlc-artefacts/strategy.md only when the user explicitly approves (e.g. "lgtm", "save", "approve").

## 3. Output structure (strategy.md)

Use these section headings (or user-agreed equivalents).

- **Introduction** — One short paragraph: what this document is (delivery + test strategy), alignment with scope. Reference only [scope.md](scope.md); no downstream links.
- **1. Delivery strategy** — Increments (e.g. Prototype, PoC, MVP, Ver 1), scope and stack per increment, iteration/dependency order, success criteria. Use subsections (1.1, 1.2, …) if helpful.
- **2. Test strategy** — Test levels and definitions (unit, integration, E2E, manual); pyramid approach; how AC should be covered; etc. Use subsections (2.1, 2.2, …) if helpful.

**Constraints:** Be short and specific. Prefer concrete over vague. One idea per bullet where applicable.

## 4. Done when

Verify all before considering the stage complete:

- [ ] strategy.md exists at ai-sdlc-artefacts/strategy.md
- [ ] Document contains the required sections above (or user-agreed subset)
- [ ] Document references only upstream pipeline artefacts (scope.md); no links to ep-scope, ep-requirements, or later stages ([pipeline.spec.md](../pipeline.spec.md) §4).
- [ ] Document does not mention downstream identifiers (EP-xx, US-xx, REQ-xx, AC-xx).
- [ ] User has explicitly approved the content
