# Acceptance criteria — US-01

**Story:** [08-user-stories.md](../../08-user-stories.md#us-01--telegram-bot)

---

## AC-001 ([US-01](../../08-user-stories.md#us-01--telegram-bot))

**Given** the bot is running and the user is allowed, **When** the user sends a text message to the bot, **Then** the user receives a text reply from the assistant within a defined timeout.

---

## AC-002 ([US-01](../../08-user-stories.md#us-01--telegram-bot))

**Given** the bot is running, **When** the user sends an empty message or a message exceeding the maximum length, **Then** the system either rejects the input with a clear message or truncates according to documented behaviour.
