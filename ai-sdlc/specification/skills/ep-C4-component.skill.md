---
name: ep-c4-component.skill
description: Add or update a C4 Level 3 (components) PlantUML diagram for Go packages in an epic system design (ep-system-design.md). Use for "C4 C3", "component diagram for Go", "internal packages diagram", or after EP-006-style escalation/core wiring changes.
---

# Epic C4 C3 — Go component diagram

**Pipeline:** Optional extension of stage 6 ([06-system-design.skill.md](06-system-design.skill.md)).  
**Primary references:** [pipeline.spec.md](../pipeline.spec.md), epic `ep-system-design.md` under `ai-sdlc-artefacts/epics/<epic-id>/`.

---

## 1. Context and goal

**Goal:** Produce a **C4 component (C3)** view of the **Go codebase** relevant to the epic—packages, key dependencies, and relationships—so operators and implementers see how `internal/*` (and `cmd/pa` if needed) connect. Complements the mandatory **C2 container** diagram (`c4-container.puml`); does not replace it.

**Typical inputs:**

- Current Go tree under `internal/` and `cmd/` (read imports and call flow; do not invent packages).
- Existing epic [ep-system-design.md](../../../ai-sdlc-artefacts/epics/EP-006/ep-system-design.md) as a pattern (adjust epic id).
- Epic requirements/design text for scope of what to include (keep the diagram readable; omit tangential packages or collapse with a note in the doc).

**Output (per epic):**

- Source: `ai-sdlc-artefacts/epics/<epic-id>/diagrams/c4-component-go.puml`
- Rendered: `ai-sdlc-artefacts/epics/<epic-id>/diagrams/c4-component-go.png`
- Markdown: new subsection under **Architecture** in `ep-system-design.md`, with **Contents (TOC)** entry and GitHub-compatible anchor.

**Rules:** Use **English** for all skill and diagram text. Links in the epic document only under `ai-sdlc-artefacts/`. Do not write or overwrite files until the user approves, unless they explicitly asked to save (align with [06-system-design.skill.md](06-system-design.skill.md) and project [AGENTS.md](../../../AGENTS.md)).

---

## 2. Naming and labeling conventions

| Topic | Convention |
|-------|----------------|
| **Go package `core`** | Component **title** (first label): `Core`. Put the module path in the **description** (third line), e.g. `Package pa/internal/core — …`. |
| **Boundary** | Prefer `PersonalAssistant — Core component (<epic-id> focus)` for the `Container_Boundary` title (or user-agreed wording). Distinguishes the bounded “core app” from the word **Core** as the `internal/core` package. |
| **Section title** | Use a consistent heading, e.g. `### C4 C3 — Core components (PlantUML)`, so the doc matches the diagram intent. |
| **Stable PNG name** | First line: `@startuml c4-component-go` (or `c4-component-<epic>`) so `plantuml -tpng` emits `c4-component-go.png` without spaces in the filename. |
| **TOC anchor** | After changing the heading, recompute the fragment for GitHub-style slugs (em dash `—` yields `--` in the id). Example: `C4 C3 — Core components (PlantUML)` → `#c4-c3--core-components-plantuml`. |

---

## 3. PlantUML template (C4-PlantUML)

Use the same include and styling approach as `c4-container.puml` for visual consistency:

```plantuml
@startuml c4-component-go

!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Component.puml

SetDefaultLegendEntries("")
' Optional: UpdateElementStyle(...) to match sibling diagrams

LAYOUT_TOP_DOWN()
HIDE_STEREOTYPE()

title PersonalAssistant — C4 C3 — Core components (PlantUML) — <short epic tag>

Container_Boundary(go_app, "PersonalAssistant — Core component (<epic-id> focus)") {
    Component(..., "Core", "Go", "Package pa/internal/core — …")
    ' Other components: one per significant package or logical group
}

System_Ext(llm_api, "LLM HTTP API", "…")

' Relationships: Rel / Rel_D as needed
Rel_D(llm_pkg, llm_api, "HTTPS", "")
Lay_D(go_app, llm_api)

@enduml
```

**External system below the boundary:** Use `Rel_D(from, external, …)` from the package that calls HTTP, plus `Lay_D(go_app, external)` so the external box sits **under** the application boundary (not to the right).

**Legend:** Do **not** call `SHOW_LEGEND()` unless the user wants a legend; it adds a **Legend** caption and extra vertical space. `SetDefaultLegendEntries("")` alone does not remove the legend title if `SHOW_LEGEND()` is present.

---

## 4. Workflow

1. **Scope** — List packages and edges from code (`import` graph, handler → router → llm, policy packages, etc.). Confirm with the user if the epic touches only a subset.
2. **Draft .puml** — Add or edit `diagrams/c4-component-go.puml` following §3 and project boundaries (no false dependencies).
3. **Render** — From the epic `diagrams/` directory: `plantuml -tpng c4-component-go.puml` (or the chosen stem). Ensure the PNG filename matches the markdown `src`.
4. **Markdown** — In `ep-system-design.md`: add/update **Contents**, `###` heading, short intro paragraph, centered image, **Source:** link to `.puml` and regeneration command (same pattern as C2).
5. **Verify** — `plantuml` exits 0; relative image path loads from the epic folder; TOC anchor matches the heading slug.

---

## 5. Done when

- [ ] `c4-component-go.puml` reflects the codebase (no orphan or misleading components).
- [ ] `c4-component-go.png` is regenerated and committed with the `.puml` when the user approves.
- [ ] `ep-system-design.md` includes TOC entry, section, image, and source/regeneration line; anchors are valid for GitHub-style rendering.
- [ ] No `SHOW_LEGEND()` unless explicitly requested.

---

**Traceability:** Optional cross-link from [06-system-design.skill.md](06-system-design.skill.md) Architecture section when C3 is added for an epic.
