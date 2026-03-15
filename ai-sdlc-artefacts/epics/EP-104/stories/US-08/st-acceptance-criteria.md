# Acceptance criteria — US-08

**Story:** [08-user-stories.md](../../08-user-stories.md#us-08--pluggable-llm-provider)

---

## AC-015 ([US-08](../../08-user-stories.md#us-08--pluggable-llm-provider))

**Given** configuration that specifies an LLM provider (e.g. OpenAI-compatible endpoint, Ollama), **When** the core calls the LLM, **Then** the configured provider is used without code change.

---

## AC-016 ([US-08](../../08-user-stories.md#us-08--pluggable-llm-provider))

**Given** a switch of provider in configuration and core restart (or hot-reload), **When** the core calls the LLM, **Then** the new provider is used.

---

## AC-036 ([US-08](../../08-user-stories.md#us-08--pluggable-llm-provider))

**Given** the LLM provider returns an error or invalid response (e.g. 4xx/5xx, empty choices, invalid JSON, context canceled, unreachable server), **When** the core uses the provider, **Then** the system handles the error (e.g. returns error to caller or fallback) and does not crash.
