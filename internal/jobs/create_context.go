package jobs

import "context"

type (
	createActorUserIDKey    struct{}
	createDeliveryChatIDKey struct{}
)

func WithCreateContext(ctx context.Context, actorUserID int64, deliveryChatID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if actorUserID != 0 {
		ctx = context.WithValue(ctx, createActorUserIDKey{}, actorUserID)
	}
	if deliveryChatID != 0 {
		ctx = context.WithValue(ctx, createDeliveryChatIDKey{}, deliveryChatID)
	}
	return ctx
}

func CreateContextActorUserID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	v, _ := ctx.Value(createActorUserIDKey{}).(int64)
	return v
}

func CreateContextDeliveryChatID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	v, _ := ctx.Value(createDeliveryChatIDKey{}).(int64)
	return v
}
