# Plan: Section 6 — Scheduler and tools (EP-104)

**Source:** [implementation-plan.md §6](implementation-plan.md#6-scheduler-and-tools)  
**Requirements:** REQ-009, REQ-010, REQ-011  
**Acceptance criteria:** AC-020, AC-021, AC-022, AC-023, AC-024

This plan is the single source of truth for scope and order of work. Each step includes a verification block; a step is not done until verification passes.

---

## Scope

**In scope**

- Tool contract: interface (Name, Description, ParamsSchema, Run), registry at startup, config can enable/parameterise tools.
- Scheduler: robfig/cron/v3; load tasks from `paths.scheduled_tasks_path` (JSON); schedule cron or `@every`; execution invokes registered tool or sends Telegram notification; within security model (allowlist for node commands, validation for tool params).
- Wire tools and scheduler into core: core (or main) holds registry, scheduler runs in background; when task fires, resolve action to tool or notify, validate input, run.
- New node/tool via config without image rebuild: after restart, new scheduled task and new node are loaded from config (nodes already work; tasks from file).
- Unit and integration tests for tools and scheduler.

**Out of scope for this section**

- LLM tool-calling (assistant choosing tools from conversation): not required for AC-020–AC-024; can be a later extension. Scheduler invokes tools by name from task config.
- Hot-reload of tasks: MVP uses restart to pick up new tasks.
- Plugin/extension loading from disk: MVP tools are code-registered; config only enables or parameterises them and defines scheduled task entries.

---

## Dependencies

- Existing: `internal/noderunner`, `internal/config` (Paths.ScheduledTasksPath), `internal/telegram` (adapter for notify).
- Config: `paths.scheduled_tasks_path` already in config struct; add loader for scheduled tasks JSON (or keep loading in scheduler).
- No change to LLM or memory for this section.

---

## Steps and verification

### Step 6.1 — Tool contract and registry

**Goal:** Define the tool interface and a registry; at least one concrete tool (e.g. run_on_node) to validate the contract.

**Tasks**

1. **Interface** (e.g. in `internal/tools/tools.go`):
   - `Tool` interface: `Name() string`, `Description() string`, `ParamsSchema() string` (JSON Schema or minimal key list for MVP), `Run(ctx context.Context, params map[string]any) (result string, err error)`.
   - Params: map or struct; schema used to validate before Run (AC-023).

2. **Registry:**
   - `Registry` type: `Register(tool Tool)`, `Get(name string) (Tool, bool)`, `List() []Tool` (or names only).
   - Built at startup; no concurrency requirements for MVP (read-only after init).

3. **Config:**
   - Optional: config section or env to enable/disable tools or pass default params. If omitted for MVP, all registered tools are available; schedule file references tool by name and passes params.

4. **First tool (run_on_node):**
   - Name e.g. `run_on_node`; params: `node_id` (string), `command` (string). Run calls `NodeRunner.RunOnNode(ctx, node_id, command)`. NodeRunner injected into tool (constructor or registry with dependencies). Validates AC-022 (single contract, validated input → result) and allows AC-021 (task that asks for disallowed command is rejected by allowlist inside RunOnNode).

**Verification**

- Unit test: register a tool, Get returns it, List includes it; Run with valid params returns result; Run with invalid/missing params returns validation error (tool not run) — AC-023.
- Unit test: run_on_node with valid node and allowlisted command returns stdout; with disallowed command returns error — AC-022, AC-007/AC-008 already cover allowlist.

---

### Step 6.2 — Scheduler (cron)

**Goal:** Load scheduled tasks from JSON file; run tasks at schedule by invoking tool or notify.

**Tasks**

1. **Task file format** (as in implementation plan):
   - JSON array of `{ "schedule": "0 9 * * *" | "@every 1h", "action": "tool_name" | "notify", "params": { ... } }`.
   - Load from `paths.scheduled_tasks_path`; if path empty or file missing, no scheduler (or empty schedule).

2. **Scheduler:**
   - Use `github.com/robfig/cron/v3`.
   - Parse each task’s `schedule` (cron expr or `@every`); add cron entries that call an executor.
   - Executor: given task, resolve `action` → if tool name, get tool from registry, validate `params` against tool’s schema, call `Run(ctx, params)`; if `notify`, send Telegram notification (see 6.3 for how to get adapter or sender).
   - Run scheduler in a goroutine; stop when context is cancelled.

3. **Security (AC-021):**
   - Task that references unknown tool → log and skip (do not run).
   - Task with params that fail validation → log and skip (do not run).
   - Task that invokes run_on_node with a command not on the node’s allowlist → RunOnNode returns error, scheduler logs it (no separate “violating task” path beyond validation + allowlist).

**Verification**

- Unit test: load task file (valid JSON), parse schedules, next run times are correct.
- Unit test: task with unknown action → not executed; task with invalid params → not executed — AC-021.
- Integration test (or unit with mock): scheduler fires at a time (or mock time), task with action = registered tool and valid params → tool Run called; result logged or returned — AC-020.

---

### Step 6.3 — Wire tools and scheduler into core

**Goal:** Main (or core) builds registry, registers tools (run_on_node + optionally notify), creates scheduler with registry and task file path, starts scheduler alongside adapter; tasks run in background.

**Tasks**

1. **Main flow:**
   - After setup(), build tool registry.
   - Register run_on_node tool (with nodeRunner); register notify “tool” (or built-in action) that needs a way to send Telegram messages — e.g. pass a notifier interface (SendMessage(ctx, text) error) implemented by telegram adapter or a small wrapper.
   - If `cfg.Paths.ScheduledTasksPath != ""`, load tasks, create scheduler with registry and notifier, start scheduler.Stop() on ctx.Done().
   - Call core.Run(ctx, ...) as today; Run starts adapter; scheduler runs in parallel (same process).

2. **Notify action:**
   - Either: notify is a special action in the scheduler (not a tool); scheduler has optional Notifier interface; when action == "notify", scheduler calls Notifier.SendMessage(ctx, task.Params["message"] or similar). Telegram adapter implements Notifier (or we add a minimal sender that uses the same token).
   - Or: notify is a registered tool that receives params and calls an injected notifier. Prefer one of these for simplicity.

3. **Core.Run:**
   - Core does not need to know about the scheduler if the scheduler is started in main and only invokes registry + notifier. So “wire into core” = wire into the process: main creates registry, passes it (and notifier) to scheduler; core.Run only needs adapter, LLM, memory, vector, nodeRunner as today. Handler does not call tools from conversation yet (no LLM tool-calling in this section).

**Verification**

- Integration test: start main with a test config (scheduled_tasks_path pointing to a file with one task in the past or use a short @every), mock or real tool; assert task runs (e.g. tool Run called or log shows execution) — AC-020.
- Manual or integration: task with action "notify" and params → message appears in Telegram (or mock notifier receives it).

---

### Step 6.4 — Add node/tool via config without image rebuild

**Goal:** Ensure new node and new scheduled task (and tool usage) are loaded after restart without rebuilding the image.

**Tasks**

1. **Nodes:** Already done: new node in config → after restart, node is in cfg.Nodes and NodeRunner uses it. No change.

2. **Tools:** “New tool” in config for MVP = (a) new scheduled task entry that references an existing tool name (e.g. run_on_node) with new params, or (b) config section that enables a built-in tool with params. No plugin loading. After restart, new task file content is loaded → new task runs when schedule fires. Document that adding a new task = edit scheduled_tasks JSON and restart.

3. **AC-024:** Integration test: add a new task to the task file (or a new node to config), restart process (or load config + scheduler in test), assert new task is scheduled or new node is used — validates “without rebuild”.

**Verification**

- Integration test: two runs — first with task file A, second with task file B (or same file with one more task); assert second run has the new task (e.g. scheduler entry count or next run for new task).
- Document in README or implementation plan: “To add a scheduled task, edit scheduled_tasks_path file and restart.”

---

### Step 6.5 — Unit and integration tests for tools and scheduler

**Goal:** Cover AC-020, AC-021, AC-022, AC-023 with unit and integration tests.

**Tasks**

1. **Unit (tools):**
   - Registry: Register, Get, List; Get unknown returns false.
   - Tool Run: valid params → result; invalid params (missing required, wrong type) → validation error, Run not called or returns error — AC-022, AC-023.
   - run_on_node: with mock NodeRunner, valid (node_id, command) → RunOnNode called, result returned; invalid params → error.

2. **Unit (scheduler):**
   - Parse task file: valid JSON → tasks with schedule and action; invalid JSON or invalid schedule → error or skip.
   - Task with unknown action → not executed (AC-021).
   - Task with invalid params for tool → validation error, not executed (AC-021).

3. **Integration:**
   - Main (or test harness) with config that has scheduled_tasks_path; task file with one task (e.g. @every 1s or fixed time in past); mock tool or run_on_node with mock NodeRunner; assert tool Run called within timeout — AC-020.
   - Task that would “violate” (e.g. unknown action or invalid params) → Run not called, no panic — AC-021.

**Verification**

- `make test` and `make test-integration` pass.
- Coverage: registry, tool Run (valid/invalid), scheduler parse and execute/skip, one integration path that runs a scheduled task.

---

## Order of execution

1. **6.1** — Tool contract and registry (interface, registry, run_on_node tool).
2. **6.2** — Scheduler (load task file, cron, executor that calls registry + notify).
3. **6.3** — Wire into main (build registry, register tools and notify, start scheduler with adapter).
4. **6.4** — Confirm config-only add (nodes + tasks) and document; add integration test for AC-024 if not already covered.
5. **6.5** — Add/finish unit and integration tests for all of the above.

Checkpoint 7 (ensure all tests pass) after 6.5.

---

## Notes

- **ParamsSchema:** For MVP, a simple map of param name → type (string/bool/number) or a JSON Schema string is enough. Validation before Run: check required keys present, types match.
- **Notify:** Prefer scheduler having an optional `Notifier` interface; telegram adapter implements it; main passes adapter (or a thin wrapper) to scheduler so action "notify" sends a message without being a separate tool.
- **Errors in Run:** Scheduler should log tool Run errors and continue; do not stop the scheduler for one task failure.
