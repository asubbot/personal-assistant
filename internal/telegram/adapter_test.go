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

// No AC: adapter construction — missing token_path returns error.
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

// No AC: adapter construction — whitespace-only token_path returns error.
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

// No AC: adapter construction — token file not found returns error.
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

// No AC: adapter construction — empty token file returns error.
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

// No AC: adapter construction — whitespace-only token returns error.
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

// No AC: adapter construction — valid token without users_path succeeds; no allowed users.
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

// No AC: adapter construction — empty users_path yields no allowed users.
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

// No AC: adapter construction — valid token and users file loads allowed user IDs.
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

// No AC: adapter construction — users_path relative to config dir is resolved.
func TestNewAdapter_usersPathRelativeToConfigDir(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	usersPath := filepath.Join(dir, "users.json")
	usersJSON := `[{"user_id": 999, "role": "user"}]`
	if err := os.WriteFile(usersPath, []byte(usersJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: usersPath}}
	ad, err := NewAdapter(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ad.allowedUserIDs[999]; !ok {
		t.Error("expected user 999 when users_path points to valid file")
	}
}

// --- NewAdapter: invalid users_path ---

// No AC: adapter construction — users file not found returns error.
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

// No AC: adapter construction — invalid users JSON returns error.
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

// --- NewAdapter: notify_chat_id (REQ-023) ---

// Covers AC-020 (US-11): notify_chat_id from config (scheduler sends to configured chat).
func TestNewAdapter_notifyChatID_fromConfig(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, NotifyChatID: 123}}
	ad, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := ad.NotifyChatID(); got != 123 {
		t.Errorf("NotifyChatID() = %d, want 123", got)
	}
}

// Covers AC-020 (US-11): notify_chat_id fallback to first allowed user when not set in config.
func TestNewAdapter_notifyChatID_fallbackToFirstUser(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	usersPath := filepath.Join(dir, "users.json")
	usersJSON := `[{"user_id": 456, "role": "user"}]`
	if err := os.WriteFile(usersPath, []byte(usersJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: usersPath, NotifyChatID: 0}}
	ad, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := ad.NotifyChatID(); got != 456 {
		t.Errorf("NotifyChatID() = %d, want 456 (first allowed user)", got)
	}
}

// Covers AC-020 (US-11): notify_chat_id is 0 when no users and not set in config.
func TestNewAdapter_notifyChatID_zeroWhenNoUsersAndNotSet(t *testing.T) {
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
	if got := ad.NotifyChatID(); got != 0 {
		t.Errorf("NotifyChatID() = %d, want 0 when no users and not set in config", got)
	}
}

// Covers AC-020 (US-11): config notify_chat_id overrides first-user fallback.
func TestNewAdapter_notifyChatID_configOverridesUsers(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	usersPath := filepath.Join(dir, "users.json")
	usersJSON := `[{"user_id": 111, "role": "user"}]`
	if err := os.WriteFile(usersPath, []byte(usersJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Telegram: config.Telegram{TokenPath: tokenPath, UsersPath: usersPath, NotifyChatID: 999}}
	ad, err := NewAdapter(cfg, filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := ad.NotifyChatID(); got != 999 {
		t.Errorf("NotifyChatID() = %d, want 999 (config overrides first user)", got)
	}
}

// --- SendMessage (scheduler Notifier) ---

// Covers AC-020 (US-11): SendMessage when bot is nil returns error (scheduler Notifier contract).
func TestSendMessage_botNil_returnsError(t *testing.T) {
	ad := &Adapter{notifyChatID: 123}
	err := ad.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when bot is nil")
	}
	if !strings.Contains(err.Error(), "cannot notify") {
		t.Errorf("unexpected error: %v", err)
	}
}

// No AC: SendMessage when notify_chat_id is 0 returns error (cannot determine target chat).
func TestSendMessage_notifyChatIDZero_returnsError(t *testing.T) {
	ad := &Adapter{notifyChatID: 0}
	err := ad.SendMessage(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when notify chat ID is 0")
	}
	if !strings.Contains(err.Error(), "cannot notify") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Run ---

// No AC: adapter.Run contract — nil handler returns error.
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

// No AC: handleUpdate — nil Message does not call handler or send.
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

// No AC: handleUpdate — nil From does not call handler or send.
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

// Covers AC-002 (US-01): empty or whitespace message → handler returns rejection message, adapter sends it.
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

// No AC: handleUpdate — disallowed user gets "not allowed" message, handler not called.
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

// Covers AC-001 (US-01): allowed user message → handler called → reply sent to user.
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

// No AC: handleUpdate — handler error results in generic error message to user.
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

// No AC: handleUpdate — empty reply from handler sends no message.
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
