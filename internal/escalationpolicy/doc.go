// Package escalationpolicy centralizes mapping from classified tool-path failure causes
// to typed tool failures for LLM escalation (EP-006, REQ-06.004, REQ-06.005, REQ-06.017).
// It does not import the conversation handler, Telegram, or concrete LLM implementations.
package escalationpolicy
