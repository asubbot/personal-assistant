# Acceptance criteria — US-10

**Story:** [08-user-stories.md](../../08-user-stories.md#us-10--log-destination-and-format)

---

## AC-018 ([US-10](../../08-user-stories.md#us-10--log-destination-and-format))

**Given** operator configuration for log destination (e.g. file path or directory), **When** the logging subsystem writes logs, **Then** entries are written to that destination in a defined, parseable format.

---

## AC-019 ([US-10](../../08-user-stories.md#us-10--log-destination-and-format))

**Given** the log destination is configured but unavailable (e.g. path not writable or disk full), **When** the logging subsystem attempts to write a log entry, **Then** the system handles the error (e.g. fail-safe or fallback) according to documented behaviour.
