# Configuration

## Main config file

- The application loads **`config.json`** from the directory set by **`PA_CONFIG_DIR`** (default `./.config`).
- The full path is always `<PA_CONFIG_DIR>/config.json`. The filename is fixed in code (`config.ConfigFileName`).
- **Explicit keys (fail fast):** the loader does **not** back-fill omitted tuning fields with hidden defaults. Every JSON-backed knob the product validates must appear in **`config.json`** with an explicit value (for example **`read_memory`**, **`write_memory`**, and when **`runtime_skills`** is present, **`max_skills_per_turn`** and **`tool_vector_top_k_cap`**). **`PA_CONFIG_DIR` / `PA_DATA_DIR` / `PA_SECRETS_DIR`** only anchor relative paths; they do not replace missing JSON sections.
- **Exhaustive top-level object:** the root of **`config.json`** must contain exactly the documented top-level keys (no omissions, no extras). Disabled optional sections use JSON **`null`**. The canonical key list is maintained in code (`config.ConfigRootJSONKeys()` / `config.ExplainConfigRootKeysForDocs()` for tooling and docs).

Start from the checked-in templates in **`config.examples/`** (copy into **`.config/`**, which is gitignored):

- **[config.examples/config.example.json](../config.examples/config.example.json)** → `.config/config.json`
- **[config.examples/known_hosts.example](../config.examples/known_hosts.example)** → `.config/known_hosts` (required when `nodes` is non-empty; populate with host keys, e.g. `ssh-keyscan`)
- **[config.examples/nas_allowlist.example](../config.examples/nas_allowlist.example)** → `.config/nas_allowlist` when using node allowlists
- **[config.examples/tools.yaml](../config.examples/tools.yaml)** → `.config/tools.yaml` (tool catalog path in JSON is relative to `PA_CONFIG_DIR`)

## Environment variables

These are read at process start; they control how **relative paths** in JSON are resolved.

| Variable | Default | Purpose |
|----------|---------|---------|
| `PA_CONFIG_DIR` | `./.config` | Directory containing `config.json`. Also the base for **relative** `command_allowlist_path`, `ssh_known_hosts_path`, `tool_catalog_path`. |
| `PA_DATA_DIR` | `.` | Base for **relative** `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`, `jobs_db_path`. |
| `PA_SECRETS_DIR` | `.` | Base for **relative** `token_path`, `users_path`, LLM/embedding `api_key_path`, node `private_key_path`. |
| `PA_LOG_LEVEL` | `info` | Log level for application output (`slog`). Invalid values fall back to `info`. At **`debug`**, the conversation handler logs full LLM request/response (sensitive — use only when needed). |
| `PA_ENV` | (unset) | Set to `development` (case-insensitive) to acknowledge intentional **diagnostic** sessions when using `PA_LOG_LEVEL=debug` on trusted hosts; suppresses the sensitive-logging startup warning. See [llm-provider-roles-and-logging.md](llm-provider-roles-and-logging.md). |

**Absolute paths** in JSON are used as-is (no joining with the bases above).

## Secrets

- Put tokens and keys in **files**; JSON only references paths (e.g. `telegram_bot_token.txt`).
- Prefer **bare filenames** in config so the same file works locally (`PA_SECRETS_DIR=.secrets`) and in Docker (`PA_SECRETS_DIR=/run/secrets` with secrets mounted by name).
- Do not commit secret files or real tokens.

## Notable JSON sections (overview)

Exact validation rules are enforced in `internal/config` at load time (fail fast).

- **`telegram`** — `token_path`, `users_path`, optional `notify_chat_id`, `max_message_length`.
- **`llm_providers`** — ordered list; at least one provider is required. Each entry has `type`, `endpoint`, `model`, optional `api_key_path`, **`supports_tools`**, **`default_temperature`**, **`default_max_tokens`**, **`default_response_format`** (must be **`"text"`**), and **`http_timeout`**. **Native tool calling:** conversation tools are sent only when the **first** provider (index **0**) has **`supports_tools`** true. **Removed:** `supports_json_mode` (reject at load). How indices map to main chat, summarization, transport fallback, and how the intent classifier differs, is described in **[llm-provider-roles-and-logging.md](llm-provider-roles-and-logging.md)**.
- **`embedding`** — separate provider for memory embeddings (vector index).
- **`paths`** — `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`, **`llm_log_retention_days`** (required, ≥ 1), `jobs_db_path`, `ssh_known_hosts_path`, `tool_catalog_path`.
- **`nodes`** — named nodes with `host`, `port`, `dedicated_user`, `auth.private_key_path`, `command_allowlist_path`. The allowlist file is one pattern per line: exact command string, or a line whose **only** `*` is the **final** character (prefix wildcard; the prefix is the text before `*`, including any trailing space — e.g. `docker images *` requires an argument after `images`, while `docker images*` matches bare `docker images`). Lines with bare `*`, multiple `*`, or `*` not at the end fail load. Executed commands must also satisfy the remote command character policy (letters, numbers, Mn/Mc, fixed ASCII punctuation including space and `"` — see [nas_allowlist.example](../config.examples/nas_allowlist.example) header); tab and shell metacharacters are rejected before SSH. EP-009 `create_tool` validates a whitelisted `docker run` prefix and a 30s timeout substring; operators SHOULD add Docker resource flags (e.g. `--memory=256m`, `--cpus=0.5`) in templates for production sandboxes. When **two or more** nodes are configured, each must use a **different** private key file after resolving `PA_SECRETS_DIR` and `filepath.Clean` (including no two nodes pointing at the same file via symlink or hard link); otherwise config load fails fast.
- **`tools`** — **required** object (use `{}` minimum). Optional **`always_include`**: array of catalog or allowed-native tool ids merged into every LLM turn’s tool set (validated at load; EP-013). Optional **`dynamic_selection`** (EP-018): object with **`enabled`** (bool) and **`max_tools_for_llm_request`** (int). When **`enabled`** is true, **`max_tools_for_llm_request`** must be **≥ 1** and **≥** the count of distinct valid `always_include` tool ids in the catalog or native allowlist (no default). **`TierFull`** and **`TierFullLite`** apply the cap after merge when dynamic selection is enabled (the former `text_based_enabled` gate is removed). When **`enabled`** is false, **`max_tools_for_llm_request`** may be zero. Optional **`vector_search_tools`** (EP-032): unified settings block for `search_vector_memory`, `search_vector_tool`, and `search_vector_skill`; each per-tool object defines `enabled`, `default_top_k`, `max_top_k`, `max_output_bytes`, and `snippet_runes`. Optional **`create_tool_secret_patterns`**: array of Go `regexp` strings (RE2). If present, each pattern must compile at config load (fail fast on invalid regex). When non-empty, the native **`create_tool`** tool rejects persisted tool definitions whose concatenated fields match any pattern (see EP-009). **Removed keys:** `tools.text_based_enabled`, `tools.llm_escalation`, and `llm_providers[].supports_json_mode` are rejected at load; conversation tools require **`llm_providers[0].supports_tools`** true.
- **`tool_pre_selection`** — **required**; `tool_search_top_k`, `tool_min_count`, and `tool_fallback_cap` must each be **≥ 1** (with documented upper caps to catch typos). No implicit defaults.
- **`conversation_context`** — **required**; **`max_dynamic_system_runes`** (UTF-8 rune budget for the dynamic system tail: tool instructions, retrieved memory, runtime skills) must be **≥ 1**. **`memory_vector`** (required object) has **`notes_top_k`**, **`summaries_top_k`**, and **`turns_top_k`**: each is an integer **≥ 0** and **≤ 500** (same upper bound as the former single `vector_search_top_k`). **0** disables automatic vector retrieval for that memory lane (no hits merged into the system message for that table). All three **0** disables automatic retrieved memory entirely (embedding for retrieval is skipped); **`read_memory`** / **`write_memory`** are unchanged.
- **`conversation_session`** — **optional** (EP-014); sliding **in-memory** window of recent user/assistant **exchanges** per session key. When present and **`enabled`** is true: **`max_session_exchanges`** must be **≥ 1** (fail fast at load otherwise). Each exchange is one user message plus the final assistant reply for that turn. Session keys come from the inbound adapter (Telegram uses **chat id** as a decimal string). When disabled or omitted, the LLM request is built as before (**system** then a single **user** message). The window is **not persisted**; a process restart clears it.
- **`runtime_skills`** — **optional** (EP-013). If the object is present, **`max_skills_per_turn`** and **`tool_vector_top_k_cap`** must each be **≥ 1** (explicit values; no defaults). When **`enabled`** is true, **`paths.skills_dir`** must point to an existing directory of skill packages.
- **`pa_timezone`** — **required**; non-empty IANA name (e.g. `UTC`, `Europe/Moscow`) for assistant day boundaries, LLM log daily filenames, `memory_dir` day paths, automatic summarization, and **`read_memory`** date interpretation.
- **`read_memory`** — **required** (EP-002). **`max_span_days`** (**1–3660**) and **`max_output_bytes`** (**1024–52428800**). The native **`read_memory`** tool is registered when **`paths.memory_dir`** is configured and the memory store initializes.
- **`write_memory`** — **required** (EP-016). **`max_append_bytes`** (**256–1048576**) and **`max_file_bytes`** (≥ **`max_append_bytes`**, at most **52428800**) bound each appended entry and the per-day **`notes.md`** file size. The native **`write_memory`** tool is a core feature; startup fails fast if required runtime dependencies are missing (**`paths.memory_dir`**, notes vector table **`vec_notes`**, embedding provider).
- **`intent_classifier`** — **optional** (EP-017, EP-018); two-stage intent classification to steer main LLM prompt assembly. When present and **`enabled`** is true: **`heuristic`** (optional) defines **`simple_patterns`**, **`full_patterns`**, optional **`full_lite_patterns`** (Go `regexp` strings, case-insensitive; invalid regex fails load), and **`max_simple_len`** (must be ≥ 1; messages longer in runes skip the simple tier and are treated as confident **`full`**). Match order in the heuristic stage is: length guard → simple → full → full_lite → ambiguous. **`model_stage`** (optional) configures a cheap classification LLM (`type`, `endpoint`, `api_key_path`, `model`, `default_temperature`, `default_max_tokens` ≥ 1, optional `timeout` as Go duration string e.g. `"5s"`, and required **`http_timeout`** as a positive Go duration — total outbound HTTP timeout for the classifier client per EP-022 REQ-22.003; distinct from the per-classification `timeout` context deadline). The cascade runs heuristic first; if ambiguous and model stage is enabled, calls the classification model (three-way tier label); on any failure defaults to **`full`**. When disabled or omitted, all messages use the full prompt path. `model_stage.api_key_path` is resolved via `PA_SECRETS_DIR`.

### Intent tiers (EP-017 / EP-018)

| Tier | RAG / retrieved chunks | Session exchanges | Runtime skill playbook in system tail | Tool list shaping |
|------|------------------------|-------------------|----------------------------------------|-------------------|
| **`simple`** | No | No | No | No catalog tools (no tool defs) |
| **`full_lite`** | No | Yes (same rules as full) | No | Merged ranked tools + `always_include`; when **`tools.dynamic_selection.enabled`**, list is filtered and capped by **`max_tools_for_llm_request`** (same rule as **`full`**) |
| **`full`** | Yes | Yes | Yes (when runtime skills enabled) | Same merge as EP-017; when **`tools.dynamic_selection.enabled`**, cap applies after ranking |
- **`log_redaction`** — **required**; `additional_patterns` may be an empty array. Each pattern has `id`, `regex`, `replacement`; IDs must not collide with built-in redactor IDs.
- **`vector_store_reliability`** / **`jobs_store_reliability`** — **required** (EP-022). Explicit per-store SQLite PRAGMA policy; see [Local SQLite stores: reliability policy](#local-sqlite-stores-reliability-policy-ep-022) below.
- **`llm_providers[].http_timeout`**, **`embedding.http_timeout`** — **required** (EP-022). Per-request total HTTP timeout for outbound calls (Go duration literal, e.g. `"120s"`). See [Outbound HTTP timeouts](#outbound-http-timeouts-ep-022) below.
- **`observability_http`** — **optional** (EP-029). When the object is present, **every** field is required: `listen_address` (TCP bind, e.g. `127.0.0.1:9090`), `health_path` and `readiness_path` (absolute paths starting with `/`, must differ), and boolean `probe_llm` (when `true`, readiness performs a short completion probe against the first configured LLM provider). When the object is **absent** (JSON `null` at the root key), the process does not start an observability HTTP listener. Operator usage: [observability-http.md](observability-http.md).

### Local SQLite stores: reliability policy (EP-022)

Both **`vector_store_reliability`** and **`jobs_store_reliability`** must be present; startup fails fast on missing fields or invalid values. The same four fields are required per store, and are applied as `mattn/go-sqlite3` DSN query parameters so every pooled connection re-asserts the PRAGMAs:

| Field | Required | Recommended | Notes |
|-------|----------|-------------|-------|
| `journal_mode` | yes | `"WAL"` | Applied at database level; WAL allows one writer + many readers concurrently. |
| `busy_timeout` | yes (Go duration, e.g. `"5s"`) | `"5s"` | Must be > 0. Re-applied per connection. |
| `synchronous` | yes | `"NORMAL"` | Per-connection. |
| `foreign_keys` | yes (bool) | `false` for vector, `true` for jobs | **Vector store** (`paths.vector_index_path`) MUST be `false`; **jobs store** (`paths.jobs_db_path`) MUST be `true`. Startup fails fast on mismatch. |

Single-writer expectation: the product opens each SQLite file from a single process; operators MUST NOT share `vector_index_path` or `jobs_db_path` between processes. The concurrent-writer reliability is covered within one process by WAL + `busy_timeout` (tested with `-race` and `-tags=integration` under `tests/integration`; see `TestConcurrentWrites_NoBusyErrors`).

### Outbound HTTP timeouts (EP-022)

Every outbound HTTP client governed by this product exposes an explicit, per-request total timeout. Missing or non-positive values fail startup:

| Field | Client | Recommended | Required |
|-------|--------|-------------|----------|
| `llm_providers[].http_timeout` | LLM chat completions (`internal/llm`) | `"120s"` | yes |
| `embedding.http_timeout` | Embeddings client (`internal/embedding`) | `"60s"` | yes |
| `web_tools.http_timeout` | Shared `*http.Client` used by `web_search` upstream calls (DuckDuckGo / Brave) and `web_fetch` (`internal/tools`) — per-request **total** timeout | `"30s"` | yes when `web_tools.enabled=true` |

`web_tools.search.timeout_seconds` and `web_tools.fetch.timeout_seconds` are per-operation context timeouts on the tool caller side and are **not** a substitute for `web_tools.http_timeout`, which bounds the whole outbound HTTP request including connection setup, TLS, and body read.

Telegram long-poll HTTP behaviour is owned by `go-telegram/bot` and is out of scope for this configuration section. SSH transport timeouts are owned by `internal/noderunner` (see the nodes section).

## Cost-aware profile (recommended)

If prompt usage spikes (especially after large tool outputs), start with this conservative runtime profile:

- `conversation_context.max_dynamic_system_runes`: `5000`
- `conversation_context.memory_vector`: e.g. `{ "notes_top_k": 2, "summaries_top_k": 2, "turns_top_k": 4 }` for a conservative cap per lane (or set notes/summaries to **0** to rely on turns + tools only)
- `conversation_session.max_session_exchanges`: `4`

This keeps retrieval and session carry-over useful, while reducing average and peak prompt size in Telegram turns.

## Automatic memory summarization and read_memory (EP-002)

- In **bot mode**, when **`paths.memory_dir`**, **`paths.llm_log_dir`**, embedding, and the vector index are available, a background worker always runs automatic day/month/year summarization (previous calendar day at **01:00** local `pa_timezone`, month/year rollups on the first local day of the month/year at **01:00**, tick **60s**, job timeout **1800s**, reconciliation scan **90** days; not configurable) and startup catch-up (see epic **EP-002**). Interactive Telegram turns take precedence over background jobs when both are pending.
- LLM JSONL logs use one file per **calendar day in `pa_timezone`** (`llm-YYYY-MM-DD.jsonl`), aligned with day summaries under `memory_dir`.
- Sample runtime skills (copy each package under your configured **`paths.skills_dir`** when **`runtime_skills.enabled`** is true): **[memory-retrieval](../config.examples/skills/memory-retrieval/SKILL.md)** (`read_memory`, `search_vector_memory`, `search_vector_tool`, `search_vector_skill`); skills may list **`write_memory`** alongside memory tools under EP-013 native-tool validation (see **`internal/config/runtime_skills.go`** allowlist). **[web-source-research](../config.examples/skills/web-source-research/SKILL.md)** (`web_fetch`, `web_search`, `run_on_node`) for bounded website and GitHub research. **[scheduled-jobs](../config.examples/skills/scheduled-jobs/SKILL.md)** (`create_scheduled_job`) when **`paths.jobs_db_path`** is set — playbook for daily in-chat scheduled jobs (EP-021); the native tool id is accepted by config validation only with a non-empty jobs DB path.

## Scheduled jobs migration note

- `paths.jobs_db_path` is the new path reserved for the EP-019 scheduled jobs database (SQLite).
- Legacy scheduler configuration fields are rejected at config load.
- The legacy file-based scheduled task list is not supported anymore.

## Tool catalog

When `paths.tool_catalog_path` is set, the YAML catalog is loaded at startup; a missing or invalid catalog prevents startup.

The assistant can define new catalog tools at runtime via the native **`create_tool`** tool (EP-009): templates must use a whitelisted `docker run` prefix and include sandbox resource flags as enforced by the product. Operators must extend each node’s **allowlist** so the resulting `docker run …` line is permitted (see [ep-scope.md](../ai-sdlc-artefacts/epics/EP-009/ep-scope.md) and `config.examples/`). Sandbox images (`pa-sandbox:*`) are built and tagged separately from this repo’s config; reference Dockerfiles and operator README: **[deploy/pa-sandbox/README.md](../deploy/pa-sandbox/README.md)**.

## Log redaction

Built-in patterns redact common secret shapes (e.g. OpenAI-style keys, Telegram bot tokens, Bearer tokens, paths suggesting secrets). Additional patterns come from `log_redaction.additional_patterns`. LLM JSONL logs are redacted before write as well.

**Tool invocation** lines at **INFO** (arguments, results, error text; for native `run_on_node` also **`remote_command`** parsed from arguments) use the redactor from config. **Noderunner** logs **`remote_command`** on validation/allowlist denials (**INFO**/**WARN**) and before successful exec (**DEBUG**); **SSH** stdout/stderr fragments in **Error** and **DEBUG** use the same redactor when `cmd/pa` wires `noderunner.SetLogRedactor(core.BuildLogRedactor(cfg))`. **Returned** tool error strings may still embed truncated remote stdout/stderr for model diagnostics — treat logs and user-visible tool text as sensitive contexts.

For epic-level design notes (not required for day-to-day ops), see under `ai-sdlc-artefacts/epics/`. For a consolidated threat summary, see [threat-model.md](../ai-sdlc-artefacts/threat-model.md).
