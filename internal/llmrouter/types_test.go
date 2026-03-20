package llmrouter

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

type captureHandler struct {
	lines []string
}

func (c *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (c *captureHandler) Handle(_ context.Context, r slog.Record) error {
	line := r.Message
	r.Attrs(func(a slog.Attr) bool {
		line += " " + a.Key + "=" + a.Value.String()
		return true
	})
	c.lines = append(c.lines, line)
	return nil
}
func (c *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return c }
func (c *captureHandler) WithGroup(_ string) slog.Handler      { return c }

func TestLogEvent_emitsStructuredRoutingLine(t *testing.T) {
	h := &captureHandler{}
	logger := slog.New(h)
	LogEvent(logger, Event{
		Phase:             PhaseCompleteError,
		FailureClass:      string(FailureClassTransport5xx),
		Action:            ActionSwitchNextTransport,
		FromIndex:         0,
		ToIndex:           1,
		Attempt:           1,
		EscalationsUsed:   0,
		FromProviderLabel: "a/m0",
		ProviderLabel:     "b/m1",
	})
	if len(h.lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(h.lines))
	}
	line := h.lines[0]
	if !strings.Contains(line, "llm routing") || !strings.Contains(line, "action=switch_next_transport") {
		t.Errorf("unexpected log line: %s", line)
	}
}
