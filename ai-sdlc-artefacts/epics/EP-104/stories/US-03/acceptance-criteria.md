# Acceptance criteria — US-03

**Story:** [08-user-stories.md](../../08-user-stories.md#us-03--node-config)

---

## AC-005 ([US-03](../../08-user-stories.md#us-03--node-config))

**Given** node configuration with invalid host or missing authentication, **When** the core starts, **Then** the core refuses to start or reports a clear error listing the validation failure.

**Given** the main config file is missing, unreadable, or invalid JSON, or a referenced file (e.g. `users_path`) is missing or invalid (e.g. invalid role), **When** the core loads configuration, **Then** the core refuses to start or reports a clear error.

---

## AC-006 ([US-03](../../08-user-stories.md#us-03--node-config))

**Given** valid node configuration, **When** the core is running, **Then** all communication to nodes uses SSH and the credentials from the validated configuration only.
