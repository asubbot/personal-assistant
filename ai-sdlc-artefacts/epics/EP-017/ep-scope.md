# Epic scope — EP-017 Intent Classifier for Prompt Optimization

| Field | Content |
|-------|---------|
| **ID** | EP-017 |
| **Status** | DONE |
| **Title** | Intent Classifier for Prompt Optimization |
| **Description** | Add a two-stage intent classification step (fast heuristic → small model fallback) before LLM prompt construction so that simple messages are served with a minimal prompt — no tools, no RAG context, no dynamic tail — significantly reducing input token consumption on the main model. |
| **First version date** | 2026-04-15 |

## Glossary

- **Intent classifier**: A two-stage component that analyses the incoming user message and assigns it to a complexity tier before the main LLM call. Stage 1 is a fast heuristic (patterns, keywords); stage 2 is a cheap model call for ambiguous cases.
- **Complexity tier**: A category that determines which prompt components are included. E.g. `simple` (no tools, no RAG), `full` (tools + RAG + skills).
- **Heuristic stage**: The first classification stage — pattern matching (regex, keyword lists) that resolves obvious cases (greetings, pings, single-word confirmations) with zero latency and no external calls.
- **Model stage**: The second classification stage — a small/cheap LLM call (via existing provider infrastructure, e.g. a lightweight Ollama model) invoked only when the heuristic stage cannot decide. Spends ~100–500 tokens of a cheap model to save ~3000–5000 tokens on the main model.
- **Dynamic tail**: The variable portion of the system prompt built by `buildDynamicTailString` — tool instructions, Hermes block, retrieved context, runtime skills — bounded by `max_dynamic_system_runes`.
- **Prompt budget**: The total input token count sent to the LLM provider per turn, as reported in the API response `usage.prompt_tokens`.

## Scope (features/capabilities)

- **Tier definition**: Define at least two complexity tiers (e.g. `simple`, `full`) with clear rules for which prompt components each tier includes (system prompt sections, tools, RAG chunks, session history depth, runtime skills).
- **Two-stage classification**:
  - **Stage 1 — Heuristic**: Fast pattern-based classifier (keyword lists, regex, short-message heuristics) that resolves obvious cases instantly. Configurable patterns.
  - **Stage 2 — Model fallback**: When the heuristic returns "ambiguous", a small/cheap model call via existing LLM provider infrastructure classifies the message. The model name and provider are configurable separately from the main conversation model.
  - **Cascade logic**: Heuristic → if confident, use its result; if not, call model stage. If model stage also fails or is disabled, default to `full` tier.
- **Prompt assembly integration**: Modify the prompt construction path in `HandleMessage` so that tier determines which components are assembled: a `simple` tier skips tool selection, RAG retrieval (`gatherRetrievedChunkTexts`), dynamic tail building, and `CompletionOptions.Tools`.
- **Configuration**: Tiers, heuristic patterns, model-stage provider/model name, and enable/disable flags are configurable via the existing configuration system (`config.yaml` / environment variables) without code changes.
- **Observability**: Log the assigned tier and which classification stage decided (heuristic vs model) per turn at INFO level. The existing usage footer continues to report actual main-model usage. Classification model usage (if invoked) is logged separately.
- **Fallback safety**: If both classification stages are disabled or fail, the system falls back to current behaviour (full prompt with all components) — no degradation of existing functionality.

## Out of scope (deferred to future epics)

- **Provider-level prompt caching** (OpenAI / Anthropic cache API) — separate concern, independent of classification.
- **Dynamic tool selection refinement** — narrowing the tool set per message beyond tier-level on/off.
- **System prompt text compression** — shortening TrustPolicy, MarkerSupplement, or tool descriptions.
- **Session history summarisation** — replacing raw sliding-window exchanges with summaries.

## Success criteria

- A message like "ты здесь?" or "привет" is classified as `simple` (via heuristic stage) and served without tools or RAG context, resulting in main-model prompt token count at least 50 % lower than the current baseline for the same message.
- Ambiguous short messages (e.g. "погода") that could require tools are routed to the model stage and classified correctly.
- Messages that require tools or memory (e.g. "напомни что я говорил вчера") are classified as `full` and prompt construction is unchanged from current behaviour.
- Disabling the classifier via configuration restores exact current behaviour (zero-change fallback).
- All existing tests pass; new unit tests cover heuristic patterns, model-stage integration, cascade logic, and prompt assembly per tier.

## Traceability

- **Scope:** Aligns with the PersonalAssistant system goal of reliability and efficiency on constrained hardware (Synology DS220+) as described in [scope.md](../../scope.md). Reducing unnecessary token consumption improves cost efficiency and response latency.
- **Strategy:** Fits within the MVP increment (v0.01) optimisation track — the core conversation path (Telegram → core → LLM → reply) is preserved; this epic optimises its resource usage without changing external behaviour.
