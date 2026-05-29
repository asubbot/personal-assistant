---
artefact: ep-scope
epic_id: EP-034
status: approved
source_of_truth: true
updated_at: 2026-05-29
---

# Epic scope — EP-034 Remove tool-path LLM escalation

| Field | Content |
|-------|---------|
| **ID** | EP-034 |
| **Status** | DONE |
| **Title** | Remove tool-path LLM escalation |
| **Description** | Remove EP-006 tool-path escalation (switching to a stronger LLM provider after qualifying tool failures) while keeping transport fallback in `llmrouter`. Simplify core handler, config, and docs as part of the Refactoring 0.02 increment. |
| **First version date** | 2026-05-29 |

## Glossary

- **Tool-path escalation:** Mid-turn switch to the next `llm_providers` entry after a qualifying tool failure (EP-006). Removed by this epic.
- **Transport fallback:** Retry on the next provider when a `Complete` call fails with a retryable transport error (timeout, network, 5xx). Retained by this epic.
- **Primary provider:** The first entry in `llm_providers` (index 0) used to start each conversation `Complete` and non-chat router paths after this epic.
- **Qualifying tool failure:** Typed tool-path outcome that previously allowed escalation (`toolfailure.MayEscalate`, `escalationpolicy`). Removed with escalation.

## Scope (features/capabilities)

- Remove packages `internal/escalationpolicy` and `internal/core/toolfailure`.
- Remove `llmrouter` tool-path escalation API (`OnQualifyingFailure`, `DecideToolFailure`, `ActionEscalatePolicy`, `State.EscUsed`).
- Simplify `conversationHandler`: no per-turn provider state for escalation; no `maybeEscalate` after tool rounds.
- Remove `tools.llm_escalation` from product config schema, examples, and validation; reject unknown legacy keys per existing fail-fast rules.
- Keep `llmrouter.Router.Complete` transport fallback across ordered `llm_providers`.
- Start all router paths at provider index **0** (remove `baseline_index` behaviour).
- Remove or rewrite EP-006 escalation-specific tests; add tests proving tool failures do not change provider index.
- Update operator docs (`configuration.md`, `llm-provider-roles-and-logging.md`, `operations.md`, `troubleshooting.md`) and [threat-model.md](../../threat-model.md) where they describe tool-path escalation.
- Record in epic artefacts that EP-034 **supersedes** the tool-path escalation portion of EP-006 (transport fallback remains).

## Out of scope

- Intent tier system (EP-017/018).
- Runtime skills, tool vector pre-selection, scheduler, memory summarization.
- Removing multi-provider pools or transport fallback.
- Changing SSH allowlist, `cmdsafe`, or tool catalog validation behaviour (errors still return to the model; only escalation is removed).

## Success criteria

- No code path advances `llm_providers` index because of a tool execution failure.
- Transport fallback still switches provider on simulated retryable `Complete` errors when a second provider is configured.
- Config load rejects `tools.llm_escalation` (removed key) and example configs contain no escalation block.
- `make check` and `./bin/validate EP-034` pass.
- Operator docs no longer describe tool-path escalation or `baseline_index`.

## Traceability

- **Scope:** Aligns with project goal to evolve architecture without radical redesign; reduces internal complexity in Core LLM routing ([scope.md](../../scope.md)).
- **Strategy:** Maps to **Refactoring 0.02** — remove extra architecture complexity ([strategy.md](../../strategy.md)).
- **Supersedes:** Tool-path escalation scope in [EP-006](../EP-006/ep-scope.md) (EP-006 status remains DONE; behaviour intentionally removed).
