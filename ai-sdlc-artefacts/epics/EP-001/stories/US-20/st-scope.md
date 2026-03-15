# Story scope — US-20 Configuration paths (environment)

**Story:** US-20  
**Title:** Configuration paths — override via environment (PA_CONFIG_DIR, PA_DATA_DIR, PA_SECRETS_DIR)

---

## Formulation

As an operator, I want to override the config file location and data/secrets directories via environment variables (`PA_CONFIG_DIR`, `PA_DATA_DIR`, `PA_SECRETS_DIR`), so that I can deploy in different environments without changing config file paths in code.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-042](../../ep-acceptance-criteria.md#ac-042) | [REQ-030](../../ep-requirements.md#configuration-paths-and-environment) | PA_CONFIG_DIR / PA_DATA_DIR / PA_SECRETS_DIR resolution (relative, absolute, unset) |
