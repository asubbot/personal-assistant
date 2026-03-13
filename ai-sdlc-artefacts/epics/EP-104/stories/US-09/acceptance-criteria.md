# Acceptance criteria — US-09

**Story:** [08-user-stories.md](../../08-user-stories.md#us-09--llm-logging)

---

## AC-017 ([US-09](../../08-user-stories.md#us-09--llm-logging))

**Given** an LLM call, **When** the call completes (or fails), **Then** the logging subsystem records the request (input messages, model parameters, request ID) and the response (output, token counts when available, duration/model identifier).
