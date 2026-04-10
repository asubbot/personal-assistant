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
- **`nodes`** — named nodes with `host`, `port`, `dedicated_user`, `auth.private_key_path`, `command_allowlist_path`. The allowlist file is one pattern per line: exact command string, or a line whose **only** `*` is the **final** character (prefix wildcard; the prefix is the text before `*`, including any trailing space — e.g. `docker images *` requires an argument after `images`, while `docker images*` matches bare `docker images`). Lines with bare `*`, multiple `*`, or `*` not at the end fail load. Executed commands must also satisfy the remote command character policy (letters, numbers, Mn/Mc, fixed ASCII punctuation including space and `"` — see [nas_allowlist.example](../config.examples/nas_allowlist.example) header); tab and shell metacharacters are rejected before SSH. EP-009 `create_tool` validates a whitelisted `docker run` prefix and a 30s timeout substring; operators SHOULD add Docker resource flags (e.g. `--memory=256m`, `--cpus=0.5`) in templates for production sandboxes.
- **`tools`** — **required** object (use `{}` minimum). Optional `text_based_enabled`; optional **`always_include`**: array of catalog or allowed-native tool ids merged into every LLM turn’s tool set (validated at load; EP-013). Optional **`llm_escalation`** (`enabled`, `max_per_user_message`, `baseline_index`). When `enabled` is true: at least two `llm_providers`, valid `baseline_index`, and **`max_per_user_message` ≥ 1**. Optional **`create_tool_secret_patterns`**: array of Go `regexp` strings (RE2). If present, each pattern must compile at config load (fail fast on invalid regex). When non-empty, the native **`create_tool`** tool rejects persisted tool definitions whose concatenated fields match any pattern (see EP-009).
- **`tool_pre_selection`** — **required**; `tool_search_top_k`, `tool_min_count`, and `tool_fallback_cap` must each be **≥ 1** (with documented upper caps to catch typos). No implicit defaults.
- **`conversation_context`** — **required**; `max_dynamic_system_runes` (UTF-8 rune budget for the dynamic system tail: tool instructions, optional Hermes block, retrieved memory, runtime skills) and `vector_search_top_k` must each be **≥ 1**.
- **`pa_timezone`** — **required**; non-empty IANA name (e.g. `UTC`, `Europe/Moscow`) for assistant day boundaries / summarization.
- **`log_redaction`** — **required**; `additional_patterns` may be an empty array. Each pattern has `id`, `regex`, `replacement`; IDs must not collide with built-in redactor IDs.

## Scheduled tasks

If `paths.scheduled_tasks_path` is non-empty, the file must exist and contain a JSON array of tasks (unique non-empty `name`, `schedule`, `action`, `params`). The scheduler starts only when there is at least one task. Duplicate names cause load failure.

## Tool catalog

When `paths.tool_catalog_path` is set, the YAML catalog is loaded at startup; a missing or invalid catalog prevents startup.

The assistant can define new catalog tools at runtime via the native **`create_tool`** tool (EP-009): templates must use a whitelisted `docker run` prefix and include sandbox resource flags as enforced by the product. Operators must extend each node’s **allowlist** so the resulting `docker run …` line is permitted (see [ep-scope.md](../ai-sdlc-artefacts/epics/EP-009/ep-scope.md) and `config.examples/`). Sandbox images (`pa-sandbox:*`) are built and tagged separately from this repo’s config; reference Dockerfiles and operator README: **[deploy/pa-sandbox/README.md](../deploy/pa-sandbox/README.md)**.

## Log redaction

Built-in patterns redact common secret shapes (e.g. OpenAI-style keys, Telegram bot tokens, Bearer tokens, paths suggesting secrets). Additional patterns come from `log_redaction.additional_patterns`. LLM JSONL logs are redacted before write as well.

**Tool invocation** lines at **INFO** (arguments, results, error text; for native `run_on_node` also **`remote_command`** parsed from arguments) use the redactor from config. **Noderunner** logs **`remote_command`** on validation/allowlist denials (**INFO**/**WARN**) and before successful exec (**DEBUG**); **SSH** stdout/stderr fragments in **Error** and **DEBUG** use the same redactor when `cmd/pa` wires `noderunner.SetLogRedactor(core.BuildLogRedactor(cfg))`. **Returned** tool error strings may still embed truncated remote stdout/stderr for model diagnostics — treat logs and user-visible tool text as sensitive contexts.

For epic-level design notes (not required for day-to-day ops), see under `ai-sdlc-artefacts/epics/`. For a consolidated threat summary, see [threat-model.md](../ai-sdlc-artefacts/threat-model.md).
