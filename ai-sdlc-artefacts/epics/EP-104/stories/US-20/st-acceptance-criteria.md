# Acceptance criteria — US-20

**Story:** [08-user-stories.md](../../08-user-stories.md#us-20--configuration-paths-environment)

---

## AC-042 ([US-20](../../08-user-stories.md#us-20--configuration-paths-environment))

**Given** the application is started with `PA_CONFIG_DIR` set to a directory or path, **When** the application loads configuration, **Then** the config file path is resolved from that value (e.g. directory + default filename or path as-is). **Given** `PA_CONFIG_DIR` is unset or empty, **When** the application loads configuration, **Then** the application uses the documented default (e.g. current directory or built-in default).

**Given** the application resolves `PA_DATA_DIR` or `PA_SECRETS_DIR`, **When** the value is a relative path, **Then** it is resolved relative to the defined base (e.g. working directory). **Given** the value is absolute, **Then** it is used unchanged. **Given** the environment variable is unset or empty, **Then** the application uses a documented default (e.g. ".").
