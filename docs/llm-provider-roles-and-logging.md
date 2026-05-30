# LLM provider pool, runtime roles, and application logging

This guide explains how the ordered `llm_providers` list in `config.json` maps to PersonalAssistant runtime roles (main conversation with transport fallback, summarization, and the optional intent classifier), and how `PA_LOG_LEVEL` / `PA_ENV` relate to **application** logs (`slog` on stderr). It does not replace the dedicated JSONL audit stream under `paths.llm_log_dir` when that path is configured.

See also [configuration.md](configuration.md) for environment variables and the full configuration overview.

## Provider pool (ordered list)

- The **provider pool** is the `llm_providers` JSON array. Indices are **zero-based** and stable for a given configuration file.
- The process builds one Go `llm.Provider` per array entry, in order. Labels used in logs derive from each entry’s `type` and `model`.

## Main conversation (chat replies)

- The **main conversation** path uses `internal/llmrouter.Router` over the same provider instances passed to `core.Run`.
- **Starting index for each new user turn:** the router always starts at index **0** (the first `llm_providers` entry).
- **Transport fallback:** On retryable completion errors (network failures, timeouts, HTTP 5xx from the provider API), the router may advance to the **next** pool index and retry the same `Complete` call, up to an internal attempt cap. Tool execution failures do **not** change the active provider index.
- Reordering or inserting entries in `llm_providers` therefore changes which model answers first and which providers participate in transport fallback.

## Summarization (memory jobs and `-summarize`)

- Summarization uses a dedicated adapter built from the **same** `llm_providers` slice.
- Each summarization `Complete` also starts at index **0** and uses the same transport fallback rules as main chat.

## Intent classifier (optional, EP-017 / EP-036)

- When `intent_classifier` is enabled, classification is **heuristic-only** (regex patterns and length guard). No extra LLM client is created for intent classification.
- Ambiguous messages default to the **`full`** tier without calling any provider from `llm_providers`.

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

Main chat and summarization both use index **0** only. Transport fallback has no alternate provider to switch to.

## Example: multi-provider pool with transport fallback

```json
"llm_providers": [
  { "type": "ollama", "endpoint": "http://localhost:11434/v1", "model": "llama3:8b", "http_timeout": "120s" },
  { "type": "openai", "endpoint": "https://api.openai.com/v1", "model": "gpt-4o-mini", "http_timeout": "120s" }
]
```

Each new turn starts at provider **0**. If a `Complete` call fails with a retryable transport error, the router may retry on provider **1**. Tool failures during a turn do not advance the provider index.
