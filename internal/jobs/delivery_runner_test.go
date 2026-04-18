package jobs

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/core"
	"strings"
	"testing"
)

type stubHandler struct {
	reply string
	err   error
	text  string
}

func (s *stubHandler) HandleMessage(_ context.Context, _ int64, _ string, text string) (string, error) {
	s.text = text
	return s.reply, s.err
}

var _ core.MessageHandler = (*stubHandler)(nil)

type stubSender struct {
	chatID int64
	text   string
}

func (s *stubSender) SendMessageToChat(_ context.Context, chatID int64, text string) error {
	s.chatID = chatID
	s.text = text
	return nil
}

// Covers AC-25.007. Supporting AC-25.008: exercised under full `make check`.
func TestDeliveryRunner_SuccessNotifiesChat(t *testing.T) {
	h := &stubHandler{reply: "Digest body"}
	snd := &stubSender{}
	r := NewDeliveryRunner(h, snd, slog.Default())
	job := Job{ID: "job-1", DeliveryChatID: 7, Instruction: "collect digest"}
	if err := r.Run(context.Background(), job); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if h.text != "collect digest" {
		t.Fatalf("handler text = %q", h.text)
	}
	if snd.chatID != 7 || !strings.Contains(snd.text, "Digest body") {
		t.Fatalf("sender got chat=%d text=%q", snd.chatID, snd.text)
	}
}

// Covers AC-25.007
func TestDeliveryRunner_FailureNotifiesClass(t *testing.T) {
	h := &stubHandler{err: errors.New("boom")}
	snd := &stubSender{}
	r := NewDeliveryRunner(h, snd, slog.Default())
	job := Job{ID: "job-2", DeliveryChatID: 9, Instruction: "collect digest"}
	if err := r.Run(context.Background(), job); err == nil {
		t.Fatal("expected error")
	}
	if snd.chatID != 9 || !strings.Contains(snd.text, "execution_error") {
		t.Fatalf("sender got chat=%d text=%q", snd.chatID, snd.text)
	}
}
