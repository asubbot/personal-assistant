# Acceptance criteria — US-02

**Story:** [08-user-stories.md](../../08-user-stories.md#us-02--docker-deploy)

---

## AC-003 ([US-02](../../08-user-stories.md#us-02--docker-deploy))

**Given** a valid Docker image of the PersonalAssistant core, **When** the operator runs the container on an x86_64 host (e.g. Synology DS220+), **Then** the core starts and exposes or uses the configured interfaces (e.g. Telegram webhook, config mount).

**Given** the core is invoked with invalid wiring (e.g. nil adapter, nil provider, or nil handler passed to the Telegram adapter), **When** Run is called, **Then** the core (or adapter) returns an error and does not start serving.

---

## AC-004 ([US-02](../../08-user-stories.md#us-02--docker-deploy))

**Given** the Dockerfile or build instructions, **When** the operator builds the image, **Then** the resulting image runs on Synology DS220+ (or equivalent x86_64) without requiring code changes.
