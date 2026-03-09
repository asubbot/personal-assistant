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
