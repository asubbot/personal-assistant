# ai-sdlc — directory index

This folder holds the **agentic SDLC process definition** for this repository: where the pipeline is specified, where per-stage agent instructions live, and where supporting tools live. It does **not** store pipeline execution outputs (artefacts).

**If you are an agent:** repo-wide behaviour (permissions, commits, secrets, language, **`make check`**) is in the root **[AGENTS.md](../AGENTS.md)**. **Pipeline rules and expectations** (stages, delegation, validation commands) are in **[specification/pipeline.spec.md](specification/pipeline.spec.md)** and the skills it references.

---

## Layout (what is where)

| Path | Role |
|------|------|
| **[specification/pipeline.spec.md](specification/pipeline.spec.md)** | **Normative process:** stage order, inputs/outputs, stage→skill mapping, Human-in-the-loop, delegated execution, artefact naming, traceability, **agent execution expectations** (single process, AC validation, etc.). |
| **[specification/skills/](specification/skills/)** | **Per-stage agent instructions** (`01-` … `11-` plus optional cross-cutting skills). Each skill defines workflow and artefact structure for its stage. |
| **[specification/skills/README.md](specification/skills/README.md)** | Index and **common behaviour** across skills. |
| **[tools/validate/](tools/validate/)** | AC↔test coverage checker (`./bin/validate`); see [VALIDATION.md](tools/validate/VALIDATION.md) and [README](tools/validate/README.md). |

---

## Artefacts (outputs of the pipeline)

Execution results (scope, strategy, epic documents, saved reviews, audit reports, etc.) are stored in **`ai-sdlc-artefacts/`** at the repository root (`../ai-sdlc-artefacts/` from here), not under `ai-sdlc/`. Paths and filenames are defined in **pipeline.spec.md** §4 and in the relevant skills.
