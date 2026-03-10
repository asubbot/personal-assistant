package telegram

import (
	"context"
	"fmt"
	"os"
	"pa/internal/config"
	"pa/internal/core"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Adapter runs the Telegram bot with long polling; filters by allowed users and forwards text to the handler.
// Implements scheduler.Notifier when bot is set (SendMessage sends to notifyChatID).
type Adapter struct {
	allowedUserIDs map[int64]struct{}
	token          string
	notifyChatID   int64    // chat ID for scheduler notify; 0 = none
	bot            *bot.Bot // set in Run before Start() so scheduler can send
}

// NewAdapter builds an adapter from config. Reads token from telegram.token_path and allowed users from telegram.users_path.
// All paths are relative to project root (CWD at startup). If users_path is empty, no users are allowed (allow-none).
func NewAdapter(cfg *config.Config, _ string) (*Adapter, error) {
	tokenPath := strings.TrimSpace(cfg.Telegram.TokenPath)
	if tokenPath == "" {
		return nil, fmt.Errorf("telegram: token_path is required")
	}
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("telegram: read token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return nil, fmt.Errorf("telegram: token is empty")
	}

	allowed := make(map[int64]struct{})
	if cfg.Telegram.UsersPath != "" {
		users, err := config.LoadTelegramUsers(cfg.Telegram.UsersPath)
		if err != nil {
			return nil, fmt.Errorf("telegram: load users: %w", err)
		}
		for _, u := range users {
			allowed[u.UserID] = struct{}{}
		}
	}
	notifyChatID := cfg.Telegram.NotifyChatID
	if notifyChatID == 0 && len(allowed) > 0 {
		for uid := range allowed {
			notifyChatID = uid
			break
		}
	}
	return &Adapter{allowedUserIDs: allowed, token: token, notifyChatID: notifyChatID}, nil
}

// NotifyChatID returns the chat ID used for scheduler notify (for tests).
func (a *Adapter) NotifyChatID() int64 { return a.notifyChatID }

// SendMessage sends a text message to the notify chat (scheduler.Notifier). No-op if bot not yet started or notifyChatID 0.
func (a *Adapter) SendMessage(ctx context.Context, text string) error {
	if a.bot == nil || a.notifyChatID == 0 {
		return fmt.Errorf("telegram: cannot notify (bot not started or no chat id)")
	}
	_, err := a.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: a.notifyChatID,
		Text:   text,
	})
	return err
}

// Run starts long polling and blocks until ctx is cancelled. Incoming text messages from allowed users are passed to handler; replies are sent back.
func (a *Adapter) Run(ctx context.Context, handler core.MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("telegram: handler is nil")
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(a.makeUpdateHandler(handler)),
	}
	b, err := bot.New(a.token, opts...)
	if err != nil {
		return fmt.Errorf("telegram: create bot: %w", err)
	}
	a.bot = b
	b.Start(ctx)
	return nil
}

// messageSender sends a message to a chat (used so update handling can be tested with a mock).
type messageSender interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
}

func (a *Adapter) makeUpdateHandler(handler core.MessageHandler) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		a.handleUpdate(ctx, b, handler, update)
	}
}

// handleUpdate processes one update; sender is used to send replies (allows tests to use a mock).
func (a *Adapter) handleUpdate(ctx context.Context, sender messageSender, handler core.MessageHandler, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	text := strings.TrimSpace(msg.Text)
	userID := msg.From.ID
	if _, ok := a.allowedUserIDs[userID]; !ok {
		_, _ = sender.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "You are not allowed to use this bot.",
		})
		return
	}

	reply, err := handler.HandleMessage(ctx, userID, text)
	if err != nil {
		_, _ = sender.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Sorry, an error occurred. Please try again.",
		})
		return
	}
	if reply != "" {
		_, _ = sender.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   reply,
		})
	}
}
