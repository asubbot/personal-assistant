# EP-014 Sliding session memory window — Acceptance Criteria

This document defines acceptance criteria for EP-014 in Gherkin form. Each AC maps to one or more requirements in [ep-requirements.md](ep-requirements.md).

**Total: 14 acceptance criteria**

---

## Index (AC → REQ)

| AC id | REQ ids | Short name |
|-------|---------|------------|
| AC-14.001 | REQ-14.001 | Config keys when section present |
| AC-14.002 | REQ-14.002 | Invalid cap fails load |
| AC-14.003 | REQ-14.003 | Telegram supplies session id |
| AC-14.004 | REQ-14.004 | In-memory store per session |
| AC-14.005 | REQ-14.005 | Concurrent updates safe |
| AC-14.006 | REQ-14.006 | Message order with history |
| AC-14.007 | REQ-14.007 | Disabled matches legacy shape |
| AC-14.008 | REQ-14.008 | Append after successful turn |
| AC-14.009 | REQ-14.009 | No append on early reject |
| AC-14.010 | REQ-14.010 | Chronological order |
| AC-14.011 | REQ-14.011 | Vector + session both possible |
| AC-14.012 | REQ-14.012 | Automated tests |
| AC-14.013 | REQ-14.013 | Redaction in logs |
| AC-14.014 | REQ-14.014 | Operator docs |

---

## AC-14.001 — Config keys when section present

**Maps to:** REQ-14.001

```gherkin
Feature: Session memory configuration

  Scenario: Operator enables session memory with a positive cap
    Given a valid configuration file includes conversation_session with enabled true
    And max_session_exchanges is a positive integer
    When the application loads configuration
    Then session memory is enabled with the configured cap
```

---

## AC-14.002 — Invalid cap fails load

**Maps to:** REQ-14.002

```gherkin
Feature: Session memory configuration validation

  Scenario: Enabled session memory with zero cap rejects startup
    Given a configuration file enables session memory
    And max_session_exchanges is less than 1
    When the application loads configuration
    Then configuration load fails with an error that references the invalid field
```

---

## AC-14.003 — Telegram supplies session id

**Maps to:** REQ-14.003

```gherkin
Feature: Telegram session threading

  Scenario: Private chat maps to one session per chat
    Given a user sends a message in a private Telegram chat
    When the Telegram adapter invokes the core handler
    Then the handler receives a session identifier derived from that chat

  Scenario: Group chat maps to a distinct session from private
    Given a user sends a message in a Telegram group
    When the Telegram adapter invokes the core handler
    Then the session identifier for the group differs from a private chat with the same user
```

---

## AC-14.004 — In-memory store per session

**Maps to:** REQ-14.004

```gherkin
Feature: Session store lifecycle

  Scenario: Session data does not survive process restart in MVP
    Given session memory is enabled
    And two exchanges were stored for session S
    When the process restarts
    Then session S has no stored exchanges until new turns occur
```

---

## AC-14.005 — Concurrent updates safe

**Maps to:** REQ-14.005

```gherkin
Feature: Concurrent session updates

  Scenario: Parallel messages for same session do not corrupt the window
    Given session memory is enabled for session S
    When two goroutines append exchanges for S concurrently under test
    Then the resulting window contains exactly the expected exchanges with no lost updates
```

---

## AC-14.006 — Message order with history

**Maps to:** REQ-14.006

```gherkin
Feature: LLM message assembly

  Scenario: Prior exchanges appear between system and current user
    Given session memory is enabled
    And one prior user and assistant exchange exists for the session
    When the handler builds messages for a new user message
    Then the first message is system
    And the next messages are user then assistant for the prior exchange
    And the last message is user with the current text
```

---

## AC-14.007 — Disabled matches legacy shape

**Maps to:** REQ-14.007

```gherkin
Feature: Backward compatible message shape

  Scenario: Session memory off yields two-message list
    Given session memory is disabled
    When the handler builds messages for any user text
    Then the message list has exactly system then one user message
```

---

## AC-14.008 — Append after successful turn

**Maps to:** REQ-14.008

```gherkin
Feature: Window update after reply

  Scenario: Successful completion adds one exchange
    Given session memory is enabled and the window is empty
    When a user turn completes with a non-empty assistant reply
    Then the window contains one exchange with that user text and reply text

  Scenario: Cap evicts oldest
    Given session memory is enabled with max_session_exchanges equal to N
    And the window already holds N exchanges
    When another successful turn completes
    Then the window still holds N exchanges
    And the oldest prior exchange is no longer present
```

---

## AC-14.009 — No append on early reject

**Maps to:** REQ-14.009

```gherkin
Feature: No window pollution on reject

  Scenario: Empty user message does not append
    Given session memory is enabled
    When the handler rejects an empty user message before LLM
    Then the session window is unchanged

  Scenario: Over-length user message does not append
    Given session memory is enabled
    When the handler rejects a message exceeding configured max length before LLM
    Then the session window is unchanged
```

---

## AC-14.010 — Chronological order

**Maps to:** REQ-14.010

```gherkin
Feature: Temporal ordering

  Scenario: Multiple exchanges inject oldest first
    Given session memory is enabled
    And three exchanges exist in order A then B then C
    When messages are built for a new turn
    Then injected pairs appear as A user/assistant, then B, then C, before the current user
```

---

## AC-14.011 — Vector + session both possible

**Maps to:** REQ-14.011

```gherkin
Feature: Coexistence with vector memory

  Scenario: System message may contain retrieval while history injects separately
    Given vector retrieval returns chunks for the merged system message
    And session memory injects at least one prior exchange
    When the LLM request is assembled
    Then the system message still contains retrieval markers or chunks per existing rules
    And additional user or assistant messages carry session history
```

---

## AC-14.012 — Automated tests

**Maps to:** REQ-14.012

```gherkin
Feature: Test coverage

  Scenario: Unit tests cover store behaviour
    Given the sliding window implementation exists
    When tests run
    Then unit tests verify cap enforcement and eviction order

  Scenario: Integration asserts multi-turn structure
    Given a test harness can observe built LLM messages
    When two user turns run in sequence for one session with session memory on
    Then the second request includes the first turn as prior user or assistant messages
```

---

## AC-14.013 — Redaction in logs

**Maps to:** REQ-14.013

```gherkin
Feature: Safe logging

  Scenario: Debug logs with session text use redaction
    Given debug logging of LLM or session content is enabled in tests
    When session window text would appear in a log line
    Then the same redaction pipeline as other user content applies
```

---

## AC-14.014 — Operator docs

**Maps to:** REQ-14.014

```gherkin
Feature: Operator documentation

  Scenario: README or config doc describes session memory
    Given operator documentation is updated for this epic
    When an operator reads configuration documentation
    Then the documentation explains enable flag, max_session_exchanges, and restart behaviour
```
