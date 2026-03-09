// Package telegram implements the Telegram bot adapter (long polling).
// NewAdapter builds from config (token and users file); Run(ctx, handler) starts polling.
// Incoming text messages from allowed users are passed to MessageHandler; replies are sent back to the chat.
package telegram
