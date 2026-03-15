# Acceptance criteria — US-11

**Story:** [08-user-stories.md](../../08-user-stories.md#us-11--scheduled-tasks)

---

## AC-020 ([US-11](../../08-user-stories.md#us-11--scheduled-tasks))

**Given** a task configured with a schedule (time or interval), **When** the scheduled time or interval is reached, **Then** the scheduler executes the task (e.g. invokes the defined tool or notification) within the security model. For tasks with action "notify", the message is sent to the Telegram chat determined by configuration (see [REQ-023](../../01-02-requirements.md#scheduler-and-tools): `telegram.notify_chat_id` or first allowed user).

---

## AC-021 ([US-11](../../08-user-stories.md#us-11--scheduled-tasks))

**Given** a task that would violate the security model, **When** the scheduler would run it, **Then** the system does not execute the violating action (and may log or report).

---

## AC-034 ([US-11](../../08-user-stories.md#us-11--scheduled-tasks))

**Given** the scheduled tasks file is missing, path is empty, JSON is invalid, or task names are duplicate or empty, **When** the core loads tasks, **Then** the system returns an empty list or reports a clear error and does not start invalid tasks.
