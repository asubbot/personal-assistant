package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"pa/internal/core"
	"strings"
)

// JobChatSender delivers scheduled-job results to a chat transport (e.g. Telegram).
type JobChatSender interface {
	SendMessageToChat(ctx context.Context, chatID int64, text string) error
}

// DeliveryRunner implements [Runner] by executing the configured [core.MessageHandler]
// and notifying [JobChatSender] — same behaviour as the previous cmd/pa scheduled runner.
type DeliveryRunner struct {
	Handler core.MessageHandler
	Sender  JobChatSender
	Logger  *slog.Logger
}

// NewDeliveryRunner returns a [Runner] that runs the handler and sends success or failure text.
func NewDeliveryRunner(handler core.MessageHandler, sender JobChatSender, logger *slog.Logger) Runner {
	return &DeliveryRunner{Handler: handler, Sender: sender, Logger: logger}
}

// Run implements [Runner].
func (r *DeliveryRunner) Run(ctx context.Context, job Job) error {
	if r.Handler == nil {
		return fmt.Errorf("scheduled job runner: handler is nil")
	}
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}
	sessionKey := fmt.Sprintf("scheduled-job:%s", job.ID)
	reply, err := r.Handler.HandleMessage(ctx, job.DeliveryChatID, sessionKey, job.Instruction)
	notifyCtx := context.WithoutCancel(ctx)
	if err != nil {
		if r.Sender != nil {
			msg := fmt.Sprintf("Scheduled job %s failed (%s).", job.ID, classifyJobFailure(err))
			if sendErr := r.Sender.SendMessageToChat(notifyCtx, job.DeliveryChatID, msg); sendErr != nil {
				logger.Warn("scheduled job failure notification", "job_id", job.ID, "error", sendErr)
			}
		}
		logger.Info("jobs audit", "actor_user_id", 0, "job_id", job.ID, "operation", "delivery", "outcome", "failure_notified")
		return err
	}
	if r.Sender != nil {
		body := strings.TrimSpace(reply)
		if body == "" {
			body = "(empty response)"
		}
		msg := fmt.Sprintf("Scheduled job %s result:\n%s", job.ID, body)
		if sendErr := r.Sender.SendMessageToChat(notifyCtx, job.DeliveryChatID, msg); sendErr != nil {
			logger.Warn("scheduled job result notification", "job_id", job.ID, "error", sendErr)
		}
	}
	logger.Info("jobs audit", "actor_user_id", 0, "job_id", job.ID, "operation", "delivery", "outcome", "success_notified")
	return nil
}

func classifyJobFailure(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "execution_error"
}
