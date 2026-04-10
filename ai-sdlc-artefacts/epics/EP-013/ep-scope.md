# Epic scope — EP-013 Runtime skills and consolidated system prompt

| Field | Content |
|-------|---------|
| **ID** | EP-013 |
| **Status** | DONE |
| **Title** | Runtime skills and consolidated system prompt |
| **Description** | Introduce AgentSkills-style runtime skill packages (SKILL.md only for MVP), a dedicated `vec_skills` index with vector selection at startup, tool selection as union(skill-declared, always_include, tool_vector_top_k) with volume budgets and fail-fast validation. Extend the single merged `role: system` string with trust policy, canonical PA_BEGIN/PA_END block markers, retrieved-context placement and rules for user turns vs tool rounds, and a bounded RUNTIME_SKILLS block. No execution of skill `scripts/`, no `references/` in prompt or index in this epic. |
| **First version date** | 2026-04-09 |
| **Audit (stage 11)** | [ep-audit-report.md](ep-audit-report.md) — 2026-04-10 (UTC) |

## Glossary

- **Runtime skill:** An operator-deployed package under a configured skills root: subdirectory per skill containing `SKILL.md` with AgentSkills-style YAML frontmatter (`name`, `description` minimum). Distinct from SDLC skills under `ai-sdlc/specification/skills/`.
- **`vec_skills`:** A dedicated sqlite-vec virtual table in the same SQLite database as existing vector tables (`vec_tools`, memory index), same embedder and embedding dimension as tools; rows keyed by stable skill id (default: skill directory name).
- **`always_include`:** Config list of tool ids that must be eligible every turn, validated to exist in `tools.yaml` or native registry.
- **Tool vector top-k:** Existing or configured semantic top-k over the tool index to widen the tool set beyond skill-declared ids.
- **Canonical block markers:** Paired lines `<<<PA_BEGIN_*>>>` / `<<<PA_END_*>>>` wrapping dynamic sections inside the merged system string (retrieved context, tool instructions, Hermes block, runtime skills).

## Scope (features/capabilities)

- Config: path to skills directory; feature flag or equivalent to disable runtime skills; `always_include` tool ids; conservative caps for skill count per turn and tool vector top-k with an upper bound; two independent rune budgets per turn (total selected skill text, total tool instruction aggregate) with whole-skill / whole-tool eviction order per analytics note §7.7.
- Load all valid skill packages at process start; parse and validate frontmatter; enforce variant **D** tool reference rules (every tool id referenced from any skill or from `always_include` must resolve; orphan `tools.yaml` entries remain allowed).
- Fail fast on startup for invalid skills, broken tool references, duplicate skill identity rules, and any `SKILL.md` line that exactly matches a canonical marker line after trim (per §8.5.1).
- Build and maintain `vec_skills`: clear and full rebuild on startup; embed text derived from `SKILL.md` (frontmatter fields + body per epic detail); no hot-reload of skills or index without process restart.
- Per user turn: vector-search skills for the user message; include full bodies of selected skills up to caps; compute final tool id set as union(skill-declared tools, always_include, tool_vector_top_k); apply volume budgets by dropping whole lowest-priority skills then whole tools only from vector top-k not pinned by selected skills or always_include.
- Fallback when skills are disabled, directory empty, or vector returns no matches: retain current tool pre-selection behaviour plus `always_include` (exact fallback rules as in implementation).
- Prompt assembly: insert English trust/injection policy block near the start of merged system (default variant B from analytics §8.4, Hermes-aware tightening where applicable); wrap dynamic sections with canonical markers in fixed order; place retrieved context in `RETRIEVED_CONTEXT` pair at the tail of system; place selected runtime skills in `RUNTIME_SKILLS` pair; do not move retrieval to a separate API message in this epic.
- Multi-turn: rebuild retrieved block on each new user turn; do not re-retrieve between tool rounds inside the same user turn; same rule for native and Hermes histories aside from message shape.
- Memory indexing: reject persistence of user or memory text that contains an exact marker line after trim (same canonical set); align with skill load rule.
- Automated tests: unit coverage for validation, marker collision rejection, prompt layout snapshots or structural checks, and integration paths with mocked embedder or store where practical; no requirement to run real cloud LLM in CI.
- Explicitly out of scope for this epic: MCP as a tool source; executing `scripts/` inside skill packages; loading `references/` into prompt or `vec_skills`; provider-specific `developer` role or multi-system-message redesign (§8.5 stage 3); hot-reload of skills.

## Success criteria

- With a configured non-empty skills directory and valid packages, at least one user message causes the outbound LLM request’s merged system content to include a `RUNTIME_SKILLS` block whose body contains text from a selected `SKILL.md`, and the model receives a tool list consistent with the union rule and caps.
- With skills disabled or zero vector matches, the core still answers and tool selection matches the agreed fallback without startup failure.
- Startup fails with a clear error if any skill references a missing tool id or if `always_include` references a missing id.
- Startup fails if any loaded `SKILL.md` contains a forbidden marker line; saving memory content that contains such a line is rejected without silent truncation.
- Retrieved context block behaviour matches the user-turn vs tool-round rule (verified by tests on constructed conversation traces).
- **E2E:** At least one automated or documented E2E scenario runs the full path **Telegram (or test double equivalent to inbound update) → core → LLM client (real configured provider or mock/stub as agreed in the implementation plan) → outbound reply**, with runtime skills enabled and a skill package present, such that the scenario demonstrates that skills affect the assembled prompt or observable tool surface without crashing the service. The implementation plan names the exact test entry point (e.g. integration test harness or Docker-based check) and whether the LLM is mocked.
- `./bin/validate EP-013` passes once acceptance criteria exist and are traced to tests per repository convention.

## Traceability

- **Scope:** Advances the PersonalAssistant **Core** capability to teach tool use via operator-managed playbooks and tighter, safer prompt structure, consistent with the system description in [scope.md](../../scope.md) (orchestration, tools, vector memory, swappable LLM backends).
- **Strategy:** Supports the **MVP** increment in [strategy.md](../../strategy.md): new behaviour remains testable (unit/integration); aligns with test strategy on traceability and prompt-injection awareness.
- **Source analysis:** Consolidated discussion and decisions in [analytics/pa-runtime-skills-tools/pa-runtime-skills-tools.md](../../analytics/pa-runtime-skills-tools/pa-runtime-skills-tools.md) (policy §7.0, §7.7, §8.2, §8.4–§8.6, Hermes retention §5.5, escalation note §5.6 / EP-006 interaction at implementation).
- **Related epics:** Runtime behaviour interacts with existing tool pre-selection (EP-004) and LLM escalation (EP-006); this epic defines the skills layer and prompt contract without removing those mechanisms.
