package core

import "context"

// MessageHandler is the interface the core implements: one text message in, one text reply out.
type MessageHandler interface {
	HandleMessage(ctx context.Context, userID int64, text string) (reply string, err error)
}

// Adapter is the abstraction for a message source (Telegram, Matrix, etc.).
// Run blocks until ctx is cancelled; incoming messages are passed to the handler, replies sent back by the adapter.
type Adapter interface {
	Run(ctx context.Context, handler MessageHandler) error
}

// NodeRunner runs an allowlisted command on a node via SSH (REQ-004, REQ-005, REQ-013).
// When a tool or flow requires node action, core (or tools) calls RunOnNode; implementation checks allowlist then runs via SSH.
// Optional: pass nil from Run when no node execution is needed.
type NodeRunner interface {
	RunOnNode(ctx context.Context, nodeID, command string) (stdout string, err error)
}
