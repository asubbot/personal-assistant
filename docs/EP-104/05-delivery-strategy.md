# Delivery Strategy — EP-104 PersonalAssistant MVP

**Purpose:** Define the sequence of shippable increments (MVP), iteration plan, and success criteria.  
**Pipeline:** [PIPELINE.SPEC.md](PIPELINE.SPEC.md)  
**Previous:** [04-system-design.md](04-system-design.md)  
**Next:** [06-test-strategy.md](06-test-strategy.md)  
**Related:** [01-02-requirements.md](01-02-requirements.md), [03-technical-discovery.md](03-technical-discovery.md), [11-12-implementation-plan.md](11-12-implementation-plan.md)

---

## 1. Named increments

For EP-104 the single planned increment is **MVP** (Minimum Viable Product), aligned with the Minimum Viable Increment (MVI) stack chosen in research.

### MVP — scope and stack

- **Language/runtime:** Go 1.26+; single static binary for linux/amd64 (`CGO_ENABLED=0` for simplicity).
- **Telegram:** [go-telegram/bot](https://github.com/go-telegram/bot) — polling for MVP; config: bot token, path to users file (user_id, role: user/admin).
- **Config:** JSON: nodes (host, dedicated PA user, auth, command_allowlist_path to file), llm_providers (ordered list, fallback), paths (memory_dir, log_path, vector_index_path, llm_log_dir, scheduled_tasks_path), telegram (token, users_path, optional notify_chat_id for scheduler notify [REQ-023](01-02-requirements.md#scheduler-and-tools)). Scheduled tasks in separate file. Validated at startup ([REQ-003](01-02-requirements.md#nodes-and-ssh)).
- **SSH:** `golang.org/x/crypto/ssh`. One user per node ([REQ-013](01-02-requirements.md#nodes-and-ssh)). Execute only commands allowed by that node’s allowlist ([REQ-005](01-02-requirements.md#nodes-and-ssh)); build commands via exec-style args, no untrusted shell.
- **Memory:** The assistant’s single store: directory of markdown files in calendar structure year/month/day ([REQ-019](01-02-requirements.md#memory-and-indexing)); hierarchical summarization (day → month → year) from LLM logs, tool execution results, and scheduler events ([REQ-020](01-02-requirements.md#memory-and-indexing)); not partitioned by interlocutor ([REQ-018](01-02-requirements.md#memory-and-indexing)). Read/write by core only; optional approval before persisting summaries; format/schema is part of design ([REQ-006](01-02-requirements.md#memory-and-indexing)). **Hierarchical memory summarization:** scheduled jobs — end-of-day produce day summary from that day’s inputs (LLM logs, tool results, scheduler events); end-of-month from day summaries; end-of-year from month summaries; timezone/config for boundaries; optional approval workflow before persist (see [implementation-plan §8](11-12-implementation-plan.md#8-hierarchical-memory-summarization)).
- **Vector index:** Pluggable vector store interface; default implementation chromem-go or vecgo (see [research §4.1](03-technical-discovery.md#41-vector-store-options-req-007-pluggable)). Index chosen fields from MD (e.g. paragraphs); embeddings via chosen LLM provider. Persist index to disk where supported to survive restarts and cap RAM. Search for user query to inject context ([REQ-007](01-02-requirements.md#memory-and-indexing)).
- **LLM:** Interface e.g. `Complete(ctx, messages, opts) (response, usage, err)`. Implementations: OpenAI-compatible HTTP (Ollama, local servers), provider selected from config ([REQ-008](01-02-requirements.md#llm-and-logging)).
- **Scheduler:** robfig/cron/v3; tasks loaded from file at path in config (cron or @every); execution = call registered tools or send Telegram notification in-process ([REQ-009](01-02-requirements.md#scheduler-and-tools)).
- **Tools:** Interface `Tool` (Name, Description, ParamsSchema, Run(ctx, params)); registry at startup; new tools = new code + config, image rebuild ([REQ-010](01-02-requirements.md#scheduler-and-tools), [REQ-011](01-02-requirements.md#extensibility-and-architecture)).
- **LLM logging ([REQ-014](01-02-requirements.md#llm-and-logging), [REQ-015](01-02-requirements.md#llm-and-logging)):** Dedicated component: on each LLM call write to configurable path (file or directory with rotation) in JSON Lines: request_id, timestamp, direction (request/response), payload (messages, model, response, usage, duration).
- **Deploy:** Dockerfile multi-stage, final image Alpine or distroless; docker-compose with single core service, volumes for config, memory, logs. Target: DS220+, x86_64 (Intel Celeron J4025).

---

## 2. Iteration plan (dependency order)

Delivery follows this sequence; each step builds on the previous. Detailed tasks and verification are in [11-12-implementation-plan.md](11-12-implementation-plan.md).

1. **Skeleton:** Packages (cmd, config, telegram, core, memory, vector, llm, scheduler, tools, ssh, logging), config load and validate, minimal main.
2. **Config and node security:** Load and validate ([REQ-003](01-02-requirements.md#nodes-and-ssh)), allowlist model per node ([REQ-005](01-02-requirements.md#nodes-and-ssh), [REQ-013](01-02-requirements.md#nodes-and-ssh)).
3. **Telegram + core:** Receive messages, call LLM (one provider), reply in chat ([REQ-001](01-02-requirements.md#interface-and-deployment), [REQ-008](01-02-requirements.md#llm-and-logging)).
4. **Memory and vector:** Read/write MD, index, semantic search and context injection ([REQ-006](01-02-requirements.md#memory-and-indexing), [REQ-007](01-02-requirements.md#memory-and-indexing)).
5. **SSH nodes:** Connect as dedicated user, run only allowed commands ([REQ-004](01-02-requirements.md#nodes-and-ssh), [REQ-013](01-02-requirements.md#nodes-and-ssh)).
6. **Scheduler and tools:** Cron jobs, tool registry, invoke from core ([REQ-009](01-02-requirements.md#scheduler-and-tools), [REQ-010](01-02-requirements.md#scheduler-and-tools)).
7. **LLM logging:** Write request/response to configurable path ([REQ-014](01-02-requirements.md#llm-and-logging), [REQ-015](01-02-requirements.md#llm-and-logging)).
8. **Hierarchical memory summarization:** Scheduled jobs for end-of-day (day summary from LLM logs, tool results, scheduler events), end-of-month (month from day summaries), end-of-year (year from month summaries); configurable timezone/boundaries; optional approval before persist ([REQ-019](01-02-requirements.md#memory-and-indexing), [REQ-020](01-02-requirements.md#memory-and-indexing)). Depends on scheduler and LLM logging. Details: [implementation-plan §8](11-12-implementation-plan.md#8-hierarchical-memory-summarization).
9. **Deploy:** Dockerfile and compose for DS220+, validate on x86_64 ([REQ-002](01-02-requirements.md#interface-and-deployment)).

---

## 3. Success criteria and checkpoints

- **Per iteration:** All tests for the completed scope pass; behaviour matches [10-acceptance-criteria.md](10-acceptance-criteria.md) for the delivered user stories.
- **Checkpoints:** The [implementation-plan](11-12-implementation-plan.md) defines checkpoints after each major section; verification at each checkpoint ensures the increment remains shippable.
