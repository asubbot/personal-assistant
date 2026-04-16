---
name: Scheduled jobs in chat
description: Daily recurring tasks the user wants the assistant to run automatically at a fixed clock time and deliver results to this Telegram chat. Use when the user asks for a digest, reminder, or agent task every day at a specific hour, or mentions cron-style daily schedules in natural language.
tools:
  - create_scheduled_job
---

## When to use `create_scheduled_job`

- The user wants a **daily** job (same wall-clock time every calendar day) executed by the assistant and delivered here.
- Extract **instruction** (what the job should do each run) and local **hour** (0–23) and **minute** (0–59) from the user message. Pass them as separate tool arguments; do not rely on hidden server-side parsing of a single prose blob.
- If the user gives a timezone name, pass it in `timezone`; otherwise omit it so the server default applies.

## Behaviour

- Call **`create_scheduled_job`** once per requested job with explicit `instruction`, `hour`, and `minute`.
- Prefer the built-in scheduler over suggesting external cron, systemd timers, or third-party job services.
- If time or task details are missing, ask one short clarifying question instead of guessing.

## After creation

- Summarize the tool output (job id, schedule, next run) for the user.
- Remind that they can manage jobs with `/jobs list` and related commands.
