// Package llmlog writes LLM request/response entries as JSON Lines (one JSON object per line)
// to a configurable directory. One file per day is used (llm-YYYY-MM-DD.jsonl) so that logs
// do not grow unbounded in a single file.
//
// Startup (NewWriter): When dir is non-empty, the directory is created if it does not exist
// (with mode 0755). If the path exists and is not a directory, or the directory is not
// writable, NewWriter returns an error so the application can fail fast with a clear error
// (AC-01.019). Callers may then refuse to start or skip LLM logging and log a warning; this
// package does not decide that.
//
// Write-time: Log(entry) never returns an error. On write failure (e.g. disk full,
// read-only filesystem, permission revoked), the error is logged to the slog.Logger
// passed to NewWriter (if non-nil) and that entry is skipped; the application does not
// crash. Subsequent Log calls are still attempted.
package llmlog
