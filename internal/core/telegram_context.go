package core

import "context"

type telegramMessageDateKey struct{}

// WithTelegramMessageDate attaches the Telegram message unix time (seconds) for the inbound user message (EP-016 event-aligned indexing). Values <= 0 are ignored.
func WithTelegramMessageDate(ctx context.Context, unixSeconds int64) context.Context {
	if unixSeconds <= 0 {
		return ctx
	}
	return context.WithValue(ctx, telegramMessageDateKey{}, unixSeconds)
}

// TelegramMessageDateUnix returns the unix seconds set by WithTelegramMessageDate, or 0.
func TelegramMessageDateUnix(ctx context.Context) int64 {
	v, _ := ctx.Value(telegramMessageDateKey{}).(int64)
	return v
}
