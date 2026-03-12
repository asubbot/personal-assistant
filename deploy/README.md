# Docker deploy (EP-104 step 9, US-02)

Run PersonalAssistant in a container (target: Synology DS220+, linux/amd64).  
Secrets are provided via **Docker Compose file-based secrets** (explicit list, easy to trace).

## Prerequisites

- Docker and Docker Compose
- Config and secret files (see below)

## Setup

1. **Config directory**  
   Create `deploy/config/` and place the config and users file there:

   ```bash
   mkdir -p deploy/config
   cp deploy/config.example.json deploy/config/config.json
   cp deploy/telegram_users.example.json deploy/config/telegram_users.json
   ```

   Edit `deploy/config/config.json` and `deploy/config/telegram_users.json` as needed.  
   In `telegram_users.json` set real Telegram `user_id` values (and optionally add `scheduled_tasks.json`, `allowlist.txt` for nodes).

2. **Secrets directory**  
   Create `deploy/secrets/` and add one file per secret (no trailing newline for tokens if the app expects a single line):

   | Secret file               | Content              | Referenced in config        |
   |---------------------------|----------------------|-----------------------------|
   | `telegram_bot_token`      | Bot token from BotFather | `telegram.token_path`   |
   | `openai_api_key`          | OpenAI API key       | `llm_providers[].api_key_path`, `embedding.api_key_path` |

   If you add nodes with SSH keys, add more secret files (e.g. `pa_nas_ed25519`) and mount them in `docker-compose.yml` with the same path as in config (e.g. `/run/secrets/pa_nas_ed25519`).

3. **Optional files in `deploy/config/`**  
   - `scheduled_tasks.json` — if `config.json` points `paths.scheduled_tasks_path` to `/etc/pa/scheduled_tasks.json`  
   - `allowlist.txt` — if you have nodes and `command_allowlist_path` points to `/etc/pa/allowlist.txt`

## Run

```bash
docker compose up -d --build
```

Logs:

```bash
docker compose logs -f pa
```

Stop:

```bash
docker compose down
```

## Verification (step 9.2)

1. **Container start**  
   After `docker compose up -d`, check that the container is running and logs show config loaded and adapter starting (e.g. `adapter=telegram`).  
   If config or secrets are wrong, the process exits with a clear error (no serving).

2. **One conversation**  
   Open the bot in Telegram and send one message; you should receive a reply from the assistant (LLM and embedding must be reachable from the container; for OpenAI, network access is required).

## Secrets checklist (traceability)

- [ ] `deploy/secrets/telegram_bot_token`
- [ ] `deploy/secrets/openai_api_key`
- [ ] (optional) Node SSH keys, e.g. `deploy/secrets/pa_nas_ed25519`

These paths are declared in `docker-compose.yml` under `secrets:` and mounted into the container at `/run/secrets/<name>`.
