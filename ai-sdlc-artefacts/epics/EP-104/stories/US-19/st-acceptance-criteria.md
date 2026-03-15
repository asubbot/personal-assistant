# Acceptance criteria — US-19

**Story:** [08-user-stories.md](../../08-user-stories.md#us-19--startup-validation)

---

## AC-033 ([US-19](../../08-user-stories.md#us-19--startup-validation))

**Given** the configuration is invalid or incomplete (e.g. config file missing or invalid JSON, Telegram token_path missing or token file empty, users file missing or invalid, LLM or embedding provider unsupported type or missing API key file), **When** the core starts, **Then** the system refuses to start or reports a clear error listing the validation failure.
