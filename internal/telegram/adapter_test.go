package telegram

import (
	"context"
	"os"
	"pa/internal/config"
	"pa/internal/core"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// --- NewAdapter: token_path ---

func TestNewAdapter_missingTokenPath(t *testing.T) {
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: ""}}
	_, err := NewAdapter(cfg, "/etc/pa/config.json")
	if err == nil {
		t.Fatal("expected error for empty token_path")
	}
	if err.Error() != "telegram: token_path is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAdapter_tokenPathWhitespaceOnly(t *testing.T) {
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: "  \t  "}}
	_, err := NewAdapter(cfg, "/etc/pa/config.json")
	if err == nil {
		t.Fatal("expected error for whitespace-only token_path")
	}
	if err.Error() != "telegram: token_path is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAdapter_tokenFileNotFound(t *testing.T) {
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: "/nonexistent/token.txt"}}
	_, err := NewAdapter(cfg, "/etc/pa/config.json")
	if err == nil {
		t.Fatal("expected error when token file does not exist")
	}
	if !strings.Contains(err.Error(), "telegram: read token:") {
		t.Errorf("expected wrapped read error, got: %v", err)
	}
}

func TestNewAdapter_tokenFileEmpty(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath}}
	_, err := NewAdapter(cfg, "/etc/pa/config.json")
	if err == nil {
		t.Fatal("expected error for empty token file")
	}
	if err.Error() != "telegram: token is empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAdapter_tokenFileWhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("  \n\t  "), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath}}
	_, err := NewAdapter(cfg, "/etc/pa/config.json")
	if err == nil {
		t.Fatal("expected error for whitespace-only token")
	}
	if err.Error() != "telegram: token is empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- NewAdapter: no users_path (allow-none) ---

func TestNewAdapter_validTokenNoUsersPath(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath}}
	ad, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if ad == nil {
		t.Fatal("expected non-nil adapter")
	}
	if len(ad.allowedUserIDs) != 0 {
		t.Errorf("expected no allowed users when users_path empty, got %d", len(ad.allowedUserIDs))
	}
}

func TestNewAdapter_validTokenEmptyUsersPath(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: ""}}
	ad, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ad.allowedUserIDs) != 0 {
		t.Errorf("expected no allowed users when users_path empty, got %d", len(ad.allowedUserIDs))
	}
}

// --- NewAdapter: valid users_path ---

func TestNewAdapter_validTokenAndUsersFile(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	usersPath := filepath.Join(dir, "users.json")
	usersJSON := `[{"user_id": 123, "role": "user"}, {"user_id": 456, "role": "admin"}]`
	if err := os.WriteFile(usersPath, []byte(usersJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: usersPath}}
	ad, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ad.allowedUserIDs[123]; !ok {
		t.Error("expected user 123 in allowed list")
	}
	if _, ok := ad.allowedUserIDs[456]; !ok {
		t.Error("expected user 456 in allowed list")
	}
	if len(ad.allowedUserIDs) != 2 {
		t.Errorf("expected 2 allowed users, got %d", len(ad.allowedUserIDs))
	}
}

func TestNewAdapter_usersPathRelativeToConfigDir(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	usersJSON := `[{"user_id": 999, "role": "user"}]`
	if err := os.WriteFile(filepath.Join(dir, "users.json"), []byte(usersJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.json")
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: "users.json"}}
	ad, err := NewAdapter(cfg, configPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ad.allowedUserIDs[999]; !ok {
		t.Error("expected user 999 when users_path is relative to config dir")
	}
}

// --- NewAdapter: invalid users_path ---

func TestNewAdapter_usersFileNotFound(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: filepath.Join(dir, "nonexistent_users.json")}}
	_, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err == nil {
		t.Fatal("expected error when users file does not exist")
	}
	if !strings.Contains(err.Error(), "telegram: load users:") {
		t.Errorf("expected load users error, got: %v", err)
	}
}

func TestNewAdapter_usersFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	usersPath := filepath.Join(dir, "users.json")
	if err := os.WriteFile(usersPath, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: usersPath}}
	_, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err == nil {
		t.Fatal("expected error for malformed users JSON")
	}
	if !strings.Contains(err.Error(), "telegram: load users:") {
		t.Errorf("expected load users error, got: %v", err)
	}
}

// --- Run ---

func TestRun_nilHandler(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{}, token: "dummy"}
	err := ad.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when handler is nil")
	}
	if err.Error() != "telegram: handler is nil" {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- handleUpdate (message handling logic) ---

type mockSender struct {
	sent []string
}

func (m *mockSender) SendMessage(_ context.Context, params *bot.SendMessageParams) (*models.Message, error) {
	if m.sent == nil {
		m.sent = []string{}
	}
	m.sent = append(m.sent, params.Text)
	return nil, nil
}

type mockHandler struct {
	called   bool
	userID   int64
	text     string
	reply    string
	errReply error
}

func (m *mockHandler) HandleMessage(_ context.Context, userID int64, text string) (string, error) {
	m.called = true
	m.userID = userID
	m.text = text
	return m.reply, m.errReply
}

func TestHandleUpdate_nilMessage(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{Message: nil})
	if handler.called {
		t.Error("handler should not be called when Message is nil")
	}
	if len(sender.sent) != 0 {
		t.Errorf("no message should be sent, got %d", len(sender.sent))
	}
}

func TestHandleUpdate_nilFrom(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "hi", Chat: models.Chat{ID: 1}},
	})
	if handler.called {
		t.Error("handler should not be called when From is nil")
	}
	if len(sender.sent) != 0 {
		t.Errorf("no message should be sent, got %d", len(sender.sent))
	}
}

func TestHandleUpdate_emptyText_sendsRejectionMessage(t *testing.T) {
	// Empty or whitespace-only text is passed to the handler; handler returns rejection message, adapter sends it.
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{reply: "Please send a non-empty message."}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	if !handler.called {
		t.Error("handler should be called for empty text (to return rejection message)")
	}
	if handler.text != "" {
		t.Errorf("handler.text = %q", handler.text)
	}
	if len(sender.sent) != 1 || sender.sent[0] != "Please send a non-empty message." {
		t.Errorf("expected rejection message sent, got: %v", sender.sent)
	}
	sender.sent = nil
	handler.called = false
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "  \t\n  ", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	if !handler.called {
		t.Error("handler should be called for whitespace-only text")
	}
	if len(sender.sent) != 1 || sender.sent[0] != "Please send a non-empty message." {
		t.Errorf("expected rejection message sent, got: %v", sender.sent)
	}
}

func TestHandleUpdate_disallowedUser(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "hello", Chat: models.Chat{ID: 1}, From: &models.User{ID: 999}},
	},
	)
	if handler.called {
		t.Error("handler should not be called for disallowed user")
	}
	if len(sender.sent) != 1 || sender.sent[0] != "You are not allowed to use this bot." {
		t.Errorf("expected 'not allowed' message, got: %v", sender.sent)
	}
}

func TestHandleUpdate_allowedUser_callsHandlerAndSendsReply(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{reply: "hello back"}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "hello", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	if !handler.called {
		t.Fatal("handler should be called for allowed user")
	}
	if handler.userID != 123 || handler.text != "hello" {
		t.Errorf("handler called with userID=%d text=%q, want 123 \"hello\"", handler.userID, handler.text)
	}
	if len(sender.sent) != 1 || sender.sent[0] != "hello back" {
		t.Errorf("expected reply \"hello back\", got: %v", sender.sent)
	}
}

func TestHandleUpdate_allowedUser_handlerErrorSendsGenericMessage(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{errReply: os.ErrNotExist}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "hello", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	if !handler.called {
		t.Fatal("handler should be called")
	}
	if len(sender.sent) != 1 || sender.sent[0] != "Sorry, an error occurred. Please try again." {
		t.Errorf("expected error message to user, got: %v", sender.sent)
	}
}

func TestHandleUpdate_allowedUser_emptyReplySendsNothing(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{reply: ""}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "hello", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	if !handler.called {
		t.Fatal("handler should be called")
	}
	if len(sender.sent) != 0 {
		t.Errorf("empty reply should not send a message, got: %v", sender.sent)
	}
}

// ensure Adapter implements core.Adapter
var _ core.Adapter = (*Adapter)(nil)
