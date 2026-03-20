# PersonalAssistant — user documentation

Operator-focused guides for installing, configuring, running, and troubleshooting PersonalAssistant. For SDLC and epic artefacts, see the repository root [README.md](../README.md) and [ai-sdlc/](../ai-sdlc/).

| Document | Description |
|----------|-------------|
| [installation.md](installation.md) | Prerequisites (Go, CGO/SQLite), clone, dependencies, first build. |
| [configuration.md](configuration.md) | Environment variables, `config.json`, path resolution, secrets, key JSON sections. |
| [docker.md](docker.md) | Docker Compose, volumes, secrets, optional `TZ`, in-container summarization cron. |
| [operations.md](operations.md) | Running the binary, CLI flags, logs, LLM log files, scheduler. |
| [troubleshooting.md](troubleshooting.md) | Common failures and checks. |
| [Threat model (artefact)](../ai-sdlc-artefacts/threat-model.md) | Code-grounded security overview for operators (not a pentest report). |
