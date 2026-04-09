package telegram

import (
	"context"
	"fmt"
	"os"
	"pa/internal/config"
	"pa/internal/core"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// --- NewAdapter: token_path ---

// Covers AC-01.033 (US-19): adapter construction — missing token_path returns error (startup validation).
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

// Covers AC-01.033 (US-19): adapter construction — whitespace-only token_path returns error.
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

// Covers AC-01.033 (US-19): adapter construction — token file not found returns error.
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

// Covers AC-01.033 (US-19): adapter construction — empty token file returns error.
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

// Covers AC-01.033 (US-19): adapter construction — whitespace-only token returns error.
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

// Covers AC-01.033 (US-19): adapter construction — valid token without users_path succeeds (optional users).
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

// Covers AC-01.033 (US-19): adapter construction — empty users_path yields no allowed users.
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

// Covers AC-01.033 (US-19): adapter construction — valid token and users file loads allowed user IDs.
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

// Covers AC-01.033 (US-19): adapter construction — users_path relative to config dir is resolved.
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

// Covers AC-01.033 (US-19): adapter construction — users file not found returns error.
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

// Covers AC-01.033 (US-19): adapter construction — invalid users JSON returns error.
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

// --- NewAdapter: notify_chat_id (REQ-01.023) ---

// Covers AC-01.020 (US-11): notify_chat_id from config (scheduler sends to configured chat).
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

// Covers AC-01.020 (US-11): notify_chat_id fallback to first allowed user when not set in config.
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

// Covers AC-01.020 (US-11): notify_chat_id is 0 when no users and not set in config.
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

// Covers AC-01.020 (US-11): config notify_chat_id overrides first-user fallback.
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

// Covers AC-01.020 (US-11): SendMessage when bot is nil returns error (scheduler Notifier contract).
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

// Covers AC-01.020, REQ-01.023 (US-11): SendMessage when notify_chat_id is 0 returns error (no destination).
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

// Covers AC-01.003 (US-02): adapter.Run with nil handler returns error and does not start serving.
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

type sentRecord struct {
	text      string
	parseMode models.ParseMode
}

type mockSender struct {
	mu              sync.Mutex
	sent            []sentRecord
	chatActionCalls int
	chatActionErr   error
	failEntityOnce  bool // first HTML send returns entity parse bad request, then succeeds plain
}

func (m *mockSender) SendMessage(_ context.Context, params *bot.SendMessageParams) (*models.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failEntityOnce && params.ParseMode == models.ParseModeHTML {
		m.failEntityOnce = false
		return nil, fmt.Errorf("%w: can't parse entities", bot.ErrorBadRequest)
	}
	if m.sent == nil {
		m.sent = []sentRecord{}
	}
	m.sent = append(m.sent, sentRecord{text: params.Text, parseMode: params.ParseMode})
	return nil, nil
}

func (m *mockSender) SendChatAction(_ context.Context, _ *bot.SendChatActionParams) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatActionCalls++
	if m.chatActionErr != nil {
		return false, m.chatActionErr
	}
	return true, nil
}

func (m *mockSender) sentTexts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.sent))
	for i, r := range m.sent {
		out[i] = r.text
	}
	return out
}

func (m *mockSender) lastParseModes() []models.ParseMode {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.ParseMode, len(m.sent))
	for i, r := range m.sent {
		out[i] = r.parseMode
	}
	return out
}

func (m *mockSender) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = nil
	m.chatActionCalls = 0
	m.failEntityOnce = false
	m.chatActionErr = nil
}

type mockHandler struct {
	called   bool
	userID   int64
	text     string
	reply    string
	errReply error
	delay    time.Duration
}

func (m *mockHandler) HandleMessage(_ context.Context, userID int64, text string) (string, error) {
	m.called = true
	m.userID = userID
	m.text = text
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.reply, m.errReply
}

// Supporting AC-01.001 (US-01): handleUpdate with nil Message does not call handler or send.
func TestHandleUpdate_nilMessage(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{Message: nil})
	if handler.called {
		t.Error("handler should not be called when Message is nil")
	}
	if len(sender.sentTexts()) != 0 {
		t.Errorf("no message should be sent, got %d", len(sender.sentTexts()))
	}
}

// Supporting AC-01.001 (US-01): handleUpdate with nil From does not call handler or send.
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
	if len(sender.sentTexts()) != 0 {
		t.Errorf("no message should be sent, got %d", len(sender.sentTexts()))
	}
}

// Covers AC-01.002 (US-01): empty or whitespace message → handler returns rejection message, adapter sends it.
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
	if len(sender.sentTexts()) != 1 || sender.sentTexts()[0] != "Please send a non-empty message." {
		t.Errorf("expected rejection message sent, got: %v", sender.sentTexts())
	}
	if sender.lastParseModes()[0] != models.ParseModeHTML {
		t.Errorf("expected ParseModeHTML, got %q", sender.lastParseModes()[0])
	}
	sender.reset()
	handler.called = false
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "  \t\n  ", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	if !handler.called {
		t.Error("handler should be called for whitespace-only text")
	}
	if len(sender.sentTexts()) != 1 || sender.sentTexts()[0] != "Please send a non-empty message." {
		t.Errorf("expected rejection message sent, got: %v", sender.sentTexts())
	}
}

// Supporting AC-01.001 (US-01): disallowed user gets "not allowed" message, handler not called.
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
	if len(sender.sentTexts()) != 1 || sender.sentTexts()[0] != "You are not allowed to use this bot." {
		t.Errorf("expected 'not allowed' message, got: %v", sender.sentTexts())
	}
}

// Covers AC-01.001 (US-01): allowed user message → handler called → reply sent to user.
// Covers AC-12.003 (EP-012): outbound reply uses ParseModeHTML.
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
	if len(sender.sentTexts()) != 1 || sender.sentTexts()[0] != "hello back" {
		t.Errorf("expected reply \"hello back\", got: %v", sender.sentTexts())
	}
	if sender.lastParseModes()[0] != models.ParseModeHTML {
		t.Errorf("expected ParseModeHTML, got %q", sender.lastParseModes()[0])
	}
}

// Supporting AC-01.001 (US-01): handler error results in generic error message to user.
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
	if len(sender.sentTexts()) != 1 || sender.sentTexts()[0] != "Sorry, an error occurred. Please try again." {
		t.Errorf("expected error message to user, got: %v", sender.sentTexts())
	}
}

// Supporting AC-01.001 (US-01): empty reply from handler sends no message.
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
	if len(sender.sentTexts()) != 0 {
		t.Errorf("empty reply should not send a message, got: %v", sender.sentTexts())
	}
}

// Covers AC-12.004 (EP-012): entity parse error → retry plain text without parse mode.
func TestSendOutboundText_entityErrorRetriesPlain(t *testing.T) {
	m := &mockSender{failEntityOnce: true}
	source := "plain **fallback**"
	err := sendOutboundText(context.Background(), m, 1, source)
	if err != nil {
		t.Fatalf("sendOutboundText: %v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) != 1 {
		t.Fatalf("want 1 successful send after retry, got %d records", len(m.sent))
	}
	if m.sent[0].text != source {
		t.Errorf("fallback text = %q, want %q", m.sent[0].text, source)
	}
	if m.sent[0].parseMode != "" {
		t.Errorf("fallback parse mode = %q, want empty", m.sent[0].parseMode)
	}
}

// Covers AC-12.005 (EP-012): notifier path uses same HTML conversion (sendOutboundText).
func TestSendOutboundText_schedulerStyle_usesHTMLParseMode(t *testing.T) {
	m := &mockSender{}
	err := sendOutboundText(context.Background(), m, 99, "task **done**")
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) != 1 || m.sent[0].parseMode != models.ParseModeHTML {
		t.Fatalf("got %+v", m.sent)
	}
	if !strings.Contains(m.sent[0].text, "<b>") {
		t.Errorf("expected converted HTML, got %q", m.sent[0].text)
	}
}

// Covers AC-12.006 (EP-012): typing refreshed while handler runs.
func TestHandleUpdate_typingRefreshedDuringSlowHandler(t *testing.T) {
	saved := atomic.LoadInt64(&typingRefreshNs)
	atomic.StoreInt64(&typingRefreshNs, int64(45*time.Millisecond))
	defer atomic.StoreInt64(&typingRefreshNs, saved)

	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{}
	handler := &mockHandler{reply: "done", delay: 120 * time.Millisecond}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "hello", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	sender.mu.Lock()
	n := sender.chatActionCalls
	sender.mu.Unlock()
	if n < 2 {
		t.Errorf("expected at least 2 typing actions, got %d", n)
	}
}

// Covers AC-12.007 (EP-012): SendChatAction errors do not block handler or reply.
func TestHandleUpdate_chatActionErrorStillSendsReply(t *testing.T) {
	ad := &Adapter{allowedUserIDs: map[int64]struct{}{123: {}}, token: ""}
	sender := &mockSender{chatActionErr: os.ErrClosed}
	handler := &mockHandler{reply: "ok"}
	ad.handleUpdate(context.Background(), sender, handler, &models.Update{
		Message: &models.Message{Text: "hello", Chat: models.Chat{ID: 1}, From: &models.User{ID: 123}},
	})
	if len(sender.sentTexts()) != 1 || sender.sentTexts()[0] != "ok" {
		t.Errorf("expected reply ok, got %v", sender.sentTexts())
	}
}

// ensure Adapter implements core.Adapter
var _ core.Adapter = (*Adapter)(nil)
