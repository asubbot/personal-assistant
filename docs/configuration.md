# Configuration

## Main config file

- The application loads **`config.json`** from the directory set by **`PA_CONFIG_DIR`** (default `./.config`).
- The full path is always `<PA_CONFIG_DIR>/config.json`. The filename is fixed in code (`config.ConfigFileName`).

Start from the checked-in templates in **`config.examples/`** (copy into **`.config/`**, which is gitignored):

- **[config.examples/config.example.json](../config.examples/config.example.json)** → `.config/config.json`
- **[config.examples/known_hosts.example](../config.examples/known_hosts.example)** → `.config/known_hosts` (required when `nodes` is non-empty; populate with host keys, e.g. `ssh-keyscan`)
- **[config.examples/nas_allowlist.example](../config.examples/nas_allowlist.example)** → `.config/nas_allowlist` when using node allowlists
- **[config.examples/scheduled_tasks.example.json](../config.examples/scheduled_tasks.example.json)** → `.config/scheduled_tasks.json` when using scheduled tasks
- **[config.examples/tools.yaml](../config.examples/tools.yaml)** → `.config/tools.yaml` (tool catalog path in JSON is relative to `PA_CONFIG_DIR`)

## Environment variables

These are read at process start; they control how **relative paths** in JSON are resolved.

| Variable | Default | Purpose |
|----------|---------|---------|
| `PA_CONFIG_DIR` | `./.config` | Directory containing `config.json`. Also the base for **relative** `command_allowlist_path`, `scheduled_tasks_path`, `ssh_known_hosts_path`, `tool_catalog_path`. |
| `PA_DATA_DIR` | `.` | Base for **relative** `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`. |
| `PA_SECRETS_DIR` | `.` | Base for **relative** `token_path`, `users_path`, LLM/embedding `api_key_path`, node `private_key_path`. |
| `PA_LOG_LEVEL` | `info` | Log level for application output (`slog`). Invalid values fall back to `info`. At **`debug`**, the conversation handler logs full LLM request/response (sensitive — use only when needed). |

**Absolute paths** in JSON are used as-is (no joining with the bases above).

## Secrets

- Put tokens and keys in **files**; JSON only references paths (e.g. `telegram_bot_token.txt`).
- Prefer **bare filenames** in config so the same file works locally (`PA_SECRETS_DIR=.secrets`) and in Docker (`PA_SECRETS_DIR=/run/secrets` with secrets mounted by name).
- Do not commit secret files or real tokens.

## Notable JSON sections (overview)

Exact validation rules are enforced in `internal/config` at load time (fail fast).

- **`telegram`** — `token_path`, `users_path`, optional `notify_chat_id`, `max_message_length`.
- **`llm_providers`** — ordered list; at least one provider is required. Each entry has `type`, `endpoint`, `model`, optional `api_key_path`, `supports_tools`.
- **`embedding`** — separate provider for memory embeddings (vector index).
- **`paths`** — `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`, **`llm_log_retention_days`** (required, ≥ 1), `scheduled_tasks_path`, `ssh_known_hosts_path`, `tool_catalog_path`.
- **`nodes`** — named nodes with `host`, `port`, `dedicated_user`, `auth.private_key_path`, `command_allowlist_path`. The allowlist file is one pattern per line: exact command string, or a line whose **only** `*` is the **final** character (prefix wildcard; the prefix is the text before `*`, including any trailing space — e.g. `docker images *` requires an argument after `images`, while `docker images*` matches bare `docker images`). Lines with bare `*`, multiple `*`, or `*` not at the end fail load. Executed commands must also satisfy the remote command character policy (letters, numbers, Mn/Mc, fixed ASCII punctuation including space and `"` — see [nas_allowlist.example](../config.examples/nas_allowlist.example) header); tab and shell metacharacters are rejected before SSH. EP-009 `create_tool` validates a whitelisted `docker run` prefix and a 30s timeout substring; operators SHOULD add Docker resource flags (e.g. `--memory=256m`, `--cpus=0.5`) in templates for production sandboxes. When **two or more** nodes are configured, each must use a **different** private key file after resolving `PA_SECRETS_DIR` and `filepath.Clean` (including no two nodes pointing at the same file via symlink or hard link); otherwise config load fails fast.
- **`tools`** — **required** object (use `{}` minimum). Optional `text_based_enabled`; optional **`always_include`**: array of catalog or allowed-native tool ids merged into every LLM turn’s tool set (validated at load; EP-013). Optional **`llm_escalation`** (`enabled`, `max_per_user_message`, `baseline_index`). When `enabled` is true: at least two `llm_providers`, valid `baseline_index`, and **`max_per_user_message` ≥ 1**. Optional **`create_tool_secret_patterns`**: array of Go `regexp` strings (RE2). If present, each pattern must compile at config load (fail fast on invalid regex). When non-empty, the native **`create_tool`** tool rejects persisted tool definitions whose concatenated fields match any pattern (see EP-009).
- **`tool_pre_selection`** — **required**; `tool_search_top_k`, `tool_min_count`, and `tool_fallback_cap` must each be **≥ 1** (with documented upper caps to catch typos). No implicit defaults.
- **`conversation_context`** — **required**; `max_dynamic_system_runes` (UTF-8 rune budget for the dynamic system tail: tool instructions, optional Hermes block, retrieved memory, runtime skills) and `vector_search_top_k` must each be **≥ 1**.
- **`conversation_session`** — **optional** (EP-014); sliding **in-memory** window of recent user/assistant **exchanges** per session key. When present and **`enabled`** is true: **`max_session_exchanges`** must be **≥ 1** (fail fast at load otherwise). Each exchange is one user message plus the final assistant reply for that turn. Session keys come from the inbound adapter (Telegram uses **chat id** as a decimal string). When disabled or omitted, the LLM request is built as before (**system** then a single **user** message). The window is **not persisted**; a process restart clears it.
- **`pa_timezone`** — **required**; non-empty IANA name (e.g. `UTC`, `Europe/Moscow`) for assistant day boundaries, LLM log daily filenames, `memory_dir` day paths, automatic summarization, and **`read_memory`** date interpretation.
- **`read_memory`** — **optional** (EP-002). If omitted, defaults apply: **`max_span_days`** **31**, **`max_output_bytes`** **262144**. The native **`read_memory`** tool is **always registered** when **`paths.memory_dir`** is configured and the memory store initializes (baseline product). **`max_span_days`** must be **1–3660** and **`max_output_bytes`** **1024–52428800** when the object is present or after defaults apply.
- **`write_memory`** — **optional** (EP-016) for limits only. **`max_append_bytes`** and **`max_file_bytes`** bound each appended entry and the per-day **`notes.md`** file size (validated at load; zeros are normalized to documented defaults). The native **`write_memory`** tool is a core feature and is always registered in normal bot startup; startup fails fast if required runtime dependencies are missing (**`paths.memory_dir`**, notes vector table **`vec_notes`**, embedding provider). If the block is omitted, documented defaults are used.
- **`log_redaction`** — **required**; `additional_patterns` may be an empty array. Each pattern has `id`, `regex`, `replacement`; IDs must not collide with built-in redactor IDs.

## Automatic memory summarization and read_memory (EP-002)

- In **bot mode**, when **`paths.memory_dir`**, **`paths.llm_log_dir`**, embedding, and the vector index are available, a background worker always runs automatic day/month/year summarization (previous calendar day at **01:00** local `pa_timezone`, month/year rollups on the first local day of the month/year at **01:00**, tick **60s**, job timeout **1800s**, reconciliation scan **90** days; not configurable) and startup catch-up (see epic **EP-002**). Interactive Telegram turns take precedence over background jobs when both are pending.
- LLM JSONL logs use one file per **calendar day in `pa_timezone`** (`llm-YYYY-MM-DD.jsonl`), aligned with day summaries under `memory_dir`.
- Sample runtime skills (copy each package under your configured **`paths.skills_dir`** when **`runtime_skills.enabled`** is true): **[memory-retrieval](../config.examples/skills/memory-retrieval/SKILL.md)** (`read_memory`); when **`write_memory`** is enabled in JSON, skills may list it alongside **`read_memory`** under EP-013 native-tool validation (see **`internal/config/runtime_skills.go`** allowlist). **[web-source-research](../config.examples/skills/web-source-research/SKILL.md)** (`web_fetch`, `web_search`, `run_on_node`) for bounded website and GitHub research.

## Scheduled tasks

If `paths.scheduled_tasks_path` is non-empty, the file must exist and contain a JSON array of tasks (unique non-empty `name`, `schedule`, `action`, `params`). The scheduler starts only when there is at least one task. Duplicate names cause load failure.

## Tool catalog

When `paths.tool_catalog_path` is set, the YAML catalog is loaded at startup; a missing or invalid catalog prevents startup.

The assistant can define new catalog tools at runtime via the native **`create_tool`** tool (EP-009): templates must use a whitelisted `docker run` prefix and include sandbox resource flags as enforced by the product. Operators must extend each node’s **allowlist** so the resulting `docker run …` line is permitted (see [ep-scope.md](../ai-sdlc-artefacts/epics/EP-009/ep-scope.md) and `config.examples/`). Sandbox images (`pa-sandbox:*`) are built and tagged separately from this repo’s config; reference Dockerfiles and operator README: **[deploy/pa-sandbox/README.md](../deploy/pa-sandbox/README.md)**.

## Log redaction

Built-in patterns redact common secret shapes (e.g. OpenAI-style keys, Telegram bot tokens, Bearer tokens, paths suggesting secrets). Additional patterns come from `log_redaction.additional_patterns`. LLM JSONL logs are redacted before write as well.

**Tool invocation** lines at **INFO** (arguments, results, error text; for native `run_on_node` also **`remote_command`** parsed from arguments) use the redactor from config. **Noderunner** logs **`remote_command`** on validation/allowlist denials (**INFO**/**WARN**) and before successful exec (**DEBUG**); **SSH** stdout/stderr fragments in **Error** and **DEBUG** use the same redactor when `cmd/pa` wires `noderunner.SetLogRedactor(core.BuildLogRedactor(cfg))`. **Returned** tool error strings may still embed truncated remote stdout/stderr for model diagnostics — treat logs and user-visible tool text as sensitive contexts.

For epic-level design notes (not required for day-to-day ops), see under `ai-sdlc-artefacts/epics/`. For a consolidated threat summary, see [threat-model.md](../ai-sdlc-artefacts/threat-model.md).
