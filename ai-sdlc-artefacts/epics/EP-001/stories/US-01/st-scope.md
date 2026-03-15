# Story scope — US-01 Telegram bot

**Story:** US-01  
**Title:** Telegram bot — receive and reply to messages

---

## Formulation

As a user, I want to send text messages to the assistant via a Telegram bot and receive text replies, so that I can interact without installing a separate app.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-001](../../ep-acceptance-criteria.md#ac-001) | [REQ-001](../../ep-requirements.md#interface-and-deployment) | User sends text message → receives reply within timeout |
| [AC-002](../../ep-acceptance-criteria.md#ac-002) | [REQ-001](../../ep-requirements.md#interface-and-deployment) | Empty or over-length message → reject or truncate with clear message |
