# Epic scope — EP-021 Scheduler routing without a separate gate (runtime skill + main handler)

| Field | Content |
|-------|---------|
| **ID** | EP-021 |
| **Status** | DONE |
| **Title** | Scheduler routing without a separate gate (main handler + optional runtime skill) |
| **Description** | Remove dedicated schedule-creation routing in the Telegram `jobsCommandHandler` wrapper: no heuristic/regex gate and no `runLLMCreateFallback` path. Scheduling relies on the existing **tier → tool merge → model** path: the native **`create_scheduled_job`** tool (explicit parameters) is sufficient for multilingual NL create **without** any runtime skill. Operators may optionally deploy an **example runtime skill** (`config.examples/skills/scheduled-jobs/`) to add playbook text and vector selection hints when `runtime_skills` is enabled — it is **not** required for correct behaviour. **The static system prompt (system head and base personality) MUST NOT change.** |
| **First version date** | 2026-04-16 |

## Pipeline (SDLC)

This document is **stage 3 (Epic planning)** per [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md). Later stages for this epic: requirements → acceptance criteria → system design → design review → implementation plan → task execution → code review → audit ([specification/skills/](../../../ai-sdlc/specification/skills/)).

## Glossary

- **Legacy separate gate:** Logic in [`cmd/pa/jobs_runtime.go`](../../../cmd/pa/jobs_runtime.go) that used text signals (`LooksLikeNaturalLanguageCreateRequest` and related) to choose between plain `base.HandleMessage` and a forced LLM fallback prompt for schedule creation.
- **Main handler:** [`internal/core`](../../../internal/core) — intent tier classification, prompt assembly, catalog + native tool merge, optional runtime skill selection on **`full`** tier only, tool-call loop.
- **Runtime skill (EP-013, optional):** A `SKILL.md` package under `paths.skills_dir`, indexed by `skillindex`, contributing playbook text to the dynamic tail when selected; may list native tool ids for merge. **Schedule creation works without any skill** if the model calls `create_scheduled_job` from tool schema and description alone.
- **Native tool `create_scheduled_job`:** Registered in [`cmd/pa/main.go`](../../../cmd/pa/main.go); implemented under `internal/jobs`; appended to main completion tool defs on `full` and `full_lite` tiers via [`completionOptionsMergedCatalogNative`](../../../internal/core/handler.go).

## Scope (features / capabilities)

- **Thin jobs wrapper:** When `jobs_db_path` is set, the outer handler serves **`/jobs` …** commands (and scheduler readiness messages) and still attaches `jobs.WithCreateContext` for actor/delivery ids. **All** other user messages are passed **directly** to `base.HandleMessage` with that context — no `handleNaturalLanguageCreate` regex path, no `runLLMCreateFallback`, no synthetic user prompt replacing the original text.
- **Create tool contract:** `create_scheduled_job` exposes **explicit** parameters (`instruction`, `hour`, `minute`, optional `timezone`, optional ids) with no free-text regex parsing inside the tool; description and schema tell the model when to call the tool and how to fill fields.
- **Optional example skill for scheduling:** Ship a repository template under `config.examples/skills/scheduled-jobs/` (embedding-friendly description, **English** playbook, `tools: [create_scheduled_job]`) for operators who enable **`runtime_skills`**; validation when `jobs_db_path` is set. **Product behaviour for NL create does not depend on this skill.**
- **Configuration documentation:** Document that the example skill is **optional**; using it requires `runtime_skills.enabled` and `paths.skills_dir`; **no** edits to `systemStaticHead` / TrustPolicy / MarkerSupplement.
- **Tests and traceability:** Update tests for the new flow; run `make check` and `./bin/validate EP-021` after AC wiring.

## Out of scope (deferred)

- Changing the **static system prompt** (`systemStaticHead` and related static blocks, base personality line).
- A dedicated mini-LLM classifier for “schedule intent” separate from tier.
- New UI beyond Telegram; EP-019 persistence/delivery semantics unchanged except integration points with the tool and wrapper.

## Success criteria

- With jobs enabled, a user describes a daily task and time; the **main** conversation path (no `runLLMCreateFallback`) can invoke `create_scheduled_job` with valid explicit fields; the job appears in `/jobs list`.
- Non-scheduling chat messages go through the same `base` path without a forced create-only system prompt.
- `/jobs` management commands behave as today.
- Static system prompt code is **unchanged** from the pre-epic baseline (default: no exceptions).
- `make check` and `./bin/validate EP-021` pass after stages 5–9.

## Risks (for design / AC)

- **`simple` tier:** No tools are attached; very short scheduling phrases might miss `create_scheduled_job`. Mitigation: EP-017/018 tuning or agreed fixture expectations in AC.

## Traceability

- **Project scope:** [scope.md](../../scope.md) — assistant goals and LLM/tool usage.
- **Strategy:** [strategy.md](../../strategy.md) — MVP delivery and configuration stability.
- **Prior epic:** Supersedes the NL routing model in [EP-020](../EP-020/ep-scope.md) (NL + hybrid gate); EP-021 does **not** remove `/jobs` management or EP-019 runtime, but **replaces** the Telegram-only gate with the **main handler + native tool** path (skill optional).
