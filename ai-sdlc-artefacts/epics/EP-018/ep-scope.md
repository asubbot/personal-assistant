# Epic scope — EP-018 Tiered Prompt Cost Reduction

| Field | Content |
|-------|---------|
| **ID** | EP-018 |
| **Status** | NEW |
| **Title** | Tiered Prompt Cost Reduction |
| **Description** | Extend the post-EP-017 prompt path with a third complexity tier (`full_lite`) and per-turn **dynamic tool selection** (bounded subset plus configured `always_include`) so that main-model input tokens drop on conversational turns without sacrificing the existing `full` tier behaviour when dynamic selection for `full` is disabled. |
| **First version date** | 2026-04-15 |

## Glossary

- **`full_lite` tier**: A complexity tier between `simple` and `full`. The main LLM turn includes session history and a **bounded** tool set, but **skips RAG / vector memory retrieval** and **omits runtime skill playbook text** from the dynamic tail. Hermes / tool-format instructions appear only when at least one tool is present in the request.
- **Dynamic tool selection**: After tier assignment, the core narrows which tool definitions are attached to the main completion request using the existing tool-ranking pipeline, configured caps, and `tools.always_include`, instead of sending the full eligible catalog for that tier.
- **Main LLM prompt**: The assembled messages and completion options sent to the primary conversation provider for a user turn, excluding the optional EP-017 classification call.

## Scope (features/capabilities)

- **Third tier (`full_lite`)**: Define rules for which prompt parts are included vs `simple` and vs `full` (RAG off, skills tail off, session history on; when conversation tools are enabled, tools only via dynamic subset; when tools are disabled for conversations, no tool definitions or Hermes block).
- **Classifier extension**: Extend the EP-017 two-stage classifier (heuristic → model) so it can assign `full_lite` in addition to `simple` and `full`; configurable patterns and model prompt text for three-way choice; safe fallback remains `full` on ambiguity or error.
- **Dynamic tool selection**: For `full_lite` (required path when tools are enabled for the assistant), and optionally for `full` when enabled by configuration, merge `always_include` with top-ranked tools from the existing pre-selection path, then enforce `max_tools_for_llm_request` (or equivalent) before building tool definitions and the Hermes block.
- **Prompt assembly integration**: Update `HandleMessage` (and related helpers) so tier and dynamic tool list jointly determine RAG calls, tail sections, tool array, and static system head text (same prose as after EP-017 except where tier rules omit sections).
- **Configuration**: New and changed fields loadable from `config.json` (and path resolution consistent with the rest of the product); invalid combinations rejected at load time.
- **Observability**: Log assigned tier, count of tools attached to the main request, and whether dynamic selection was applied for that turn (INFO).
- **Regression safety**: With classifier disabled or conservative tier defaults, behaviour matches the EP-017 baseline as defined in acceptance criteria.

## Out of scope (deferred)

- **System prompt density / static text compression** — configurable `prompt_density` (`standard` | `compact`), compact-eligible static sections, alternate short wordings for non-tool system head blocks, and optional TrustPolicy shortening flags. (Formerly listed as in-scope for EP-018; deferred to a future epic.)
- **Provider-level prompt caching** (OpenAI / Anthropic cache APIs).
- **Session history summarisation** or sliding-window token accounting beyond existing `conversation_session` settings.
- **New embedding models** or changes to embedding dimensions.
- **Automatic rewriting** of user-visible assistant personality or TrustPolicy prose without an explicit operator-controlled flag.

## Success criteria

- For a fixed **full_lite** fixture message (no tools in ranked output except `always_include`), main-model prompt token count is **measurably lower** than the same message on the `full` tier in an automated test (documented threshold in acceptance criteria, e.g. ≥15% reduction or rune-count proxy agreed in AC).
- For a user turn on **`full` tier** with dynamic selection **disabled**, prompt assembly and tool list match the EP-017 `full` baseline (byte-identical or structurally identical per AC).
- **Dynamic tool selection** never exceeds the configured maximum; `always_include` tools are always present when configured and valid.
- `make check` passes; `./bin/validate EP-018` passes after AC↔test wiring in stage 9.

## Traceability

- **Scope:** Supports efficiency goals for constrained hardware in [scope.md](../../scope.md) (Synology-class deployment, cost-aware LLM usage).
- **Strategy:** Aligns with MVP increment **v0.01** in [strategy.md](../../strategy.md) (optimisation without breaking the core Telegram → core → LLM path).
- **Related epic:** Builds on [EP-017 Intent Classifier](../EP-017/ep-scope.md); EP-018 extends tiers and prompt assembly but does not remove EP-017 safety properties.
