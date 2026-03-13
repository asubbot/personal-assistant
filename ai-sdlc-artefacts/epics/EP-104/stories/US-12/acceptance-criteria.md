# Acceptance criteria — US-12

**Story:** [08-user-stories.md](../../08-user-stories.md#us-12--extensible-tools)

---

## AC-022 ([US-12](../../08-user-stories.md#us-12--extensible-tools))

**Given** a tool with name, description, and validated input schema registered with the core, **When** the core invokes the tool, **Then** the invocation follows the single contract (e.g. input validated, result returned).

**Given** tool registration with invalid data (e.g. empty name or duplicate name), **When** Register is called, **Then** the system rejects or fails fast (e.g. panic or error).

---

## AC-023 ([US-12](../../08-user-stories.md#us-12--extensible-tools))

**Given** an invalid or out-of-schema input for a tool, **When** the core would invoke it, **Then** the system validates and rejects or reports the error without executing the tool.

---

## AC-035 ([US-12](../../08-user-stories.md#us-12--extensible-tools))

**Given** a tool is invoked with a nil runner (or equivalent invalid dependency) or the runner returns an error, **When** the tool Run is called, **Then** the tool returns an error to the caller and does not execute the violating action.
