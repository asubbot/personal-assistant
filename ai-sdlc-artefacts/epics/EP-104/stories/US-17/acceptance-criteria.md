# Acceptance criteria — US-17

**Story:** [08-user-stories.md](../../08-user-stories.md#us-17--debug-llm-logging)

---

## AC-031 ([US-17](../../08-user-stories.md#us-17--debug-llm-logging))

**Given** the application is started with `PA_LOG_LEVEL=debug` (or equivalent case-insensitive value), **When** a user message is processed and the core calls the LLM provider, **Then** the core logs the full request (messages sent to the provider, including assembled context from memory and vector search; may be truncated at a documented length) and the full response (model output and usage) at DEBUG level.

**Given** the application is started with the default log level (INFO) or with `PA_LOG_LEVEL=info`, **When** a user message is processed and the core calls the LLM provider, **Then** the core logs only metadata (e.g. message count, response length, token usage) and does NOT log full request or response bodies.
