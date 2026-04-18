# LLM provider pool, runtime roles, and application logging

This guide explains how the ordered `llm_providers` list in `config.json` maps to PersonalAssistant runtime roles (main conversation with optional escalation and transport fallback, summarization, and the optional intent classifier), and how `PA_LOG_LEVEL` / `PA_ENV` relate to **application** logs (`slog` on stderr). It does not replace the dedicated JSONL audit stream under `paths.llm_log_dir` when that path is configured.

See also [configuration.md](configuration.md) for environment variables and the full configuration overview.

## Provider pool (ordered list)

- The **provider pool** is the `llm_providers` JSON array. Indices are **zero-based** and stable for a given configuration file.
- The process builds one Go `llm.Provider` per array entry, in order. Labels used in logs derive from each entry’s `type` and `model`.

## Main conversation (chat replies)

- The **main conversation** path uses `internal/llmrouter.Router` over the same provider instances passed to `core.Run`.
- **Starting index for each new user turn:**
  - If `tools.llm_escalation.enabled` is **false** (or the block is absent): the router starts at index **0**.
  - If escalation is **true**: the router starts at `tools.llm_escalation.baseline_index` (validated at config load to lie within the pool).
- **Transport fallback:** On certain retryable completion errors, the router may advance to the **next** pool index and retry, up to an internal attempt cap. This is independent of escalation counters; escalation policies still apply on top of the active index.
- Reordering or inserting entries in `llm_providers` therefore changes which model answers first and which providers participate in fallback.

## Summarization (memory jobs and `-summarize`)

- Summarization uses a dedicated adapter built from the **same** `llm_providers` slice.
- Router configuration follows `SummarizeRouterConfig`: when escalation is enabled, summarization starts at `baseline_index` like the main chat router; otherwise it starts at index **0**.

## Intent classifier (optional, EP-017)

- When `intent_classifier` is enabled and the **model stage** is enabled, the process constructs a **separate** LLM client from `intent_classifier.model_stage` fields (`type`, `endpoint`, `model`, timeouts, and so on).
- That classifier client is not selected by an index into `llm_providers`. Operators configure it explicitly under `intent_classifier.model_stage` (including its own `api_key_path` resolution via `PA_SECRETS_DIR`).

## Application log level (`PA_LOG_LEVEL`) and `PA_ENV`

- `PA_LOG_LEVEL` sets the root `slog` level. At **`debug`**, conversation code paths may log **full** LLM request and response bodies to the application log stream (high sensitivity).
- At **`info`** (the in-process default when unset or invalid), those paths log metadata only.
- **`PA_ENV=development`** (ASCII case-insensitive) suppresses the startup warning that reminds you about sensitive logging when `PA_LOG_LEVEL=debug`. Use it only on trusted diagnostic hosts.

The production `Dockerfile` sets `ENV PA_LOG_LEVEL=info`. Compose uses `PA_LOG_LEVEL=${PA_LOG_LEVEL:-info}` so the default is `info` while still allowing a higher verbosity from `.env` when you intend to.

## Example: single-provider pool

```json
"llm_providers": [
  {
    "type": "ollama",
    "endpoint": "http://localhost:11434/v1",
    "model": "llama3:8b",
    "http_timeout": "120s"
  }
]
```

Main chat and summarization both use index **0** only. Escalation is off unless you add `tools.llm_escalation` with `enabled: true` (which requires at least two providers).

## Example: escalation-enabled pool

```json
"tools": {
  "llm_escalation": {
    "enabled": true,
    "max_per_user_message": 2,
    "baseline_index": 0
  }
},
"llm_providers": [
  { "type": "ollama", "endpoint": "http://localhost:11434/v1", "model": "llama3:8b", "http_timeout": "120s" },
  { "type": "openai", "endpoint": "https://api.openai.com/v1", "model": "gpt-4o-mini", "http_timeout": "120s" }
]
```

Each new turn starts at provider **0**. Failed completions that qualify for transport fallback may move to provider **1**. Per-message escalation limits are configured separately (`max_per_user_message`).

## Example: pool with intent classifier enabled

Keep your main pool as above. Add a minimal `intent_classifier` block (patterns and timeouts omitted here for brevity — see [configuration.md](configuration.md) for required fields):

```json
"intent_classifier": {
  "enabled": true,
  "model_stage": {
    "enabled": true,
    "type": "openai",
    "endpoint": "https://api.openai.com/v1",
    "api_key_path": "openai_api_key.txt",
    "model": "gpt-4o-mini",
    "default_temperature": 0,
    "default_max_tokens": 16,
    "http_timeout": "30s"
  }
}
```

The classifier’s model is configured only under `intent_classifier.model_stage`, not by referencing an `llm_providers` index.
