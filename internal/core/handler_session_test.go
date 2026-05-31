package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"strings"
	"testing"
)

// Covers AC-14.006, AC-14.010, AC-14.012: prior exchange appears between system and current user, oldest first.
func TestHandleMessage_sessionMemory_injectsHistoryBetweenSystemAndUser(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "first reply"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "chat-1", "hello"); err != nil {
		t.Fatalf("first turn: %v", err)
	}
	p.result = &llm.CompletionResult{Content: "second reply"}
	if _, err := h.HandleMessage(ctx, 1, "chat-1", "follow up"); err != nil {
		t.Fatalf("second turn: %v", err)
	}
	msgs := p.lastMessages
	if len(msgs) != 4 {
		t.Fatalf("len(messages) = %d, want 4", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[1].Role != "user" || msgs[1].Content != "hello" ||
		msgs[2].Role != "assistant" || msgs[2].Content != "first reply" ||
		msgs[3].Role != "user" || msgs[3].Content != "follow up" {
		t.Errorf("messages: %#v", msgs)
	}
}

// Covers AC-14.007: when session memory off, only system + one user message.
func TestHandleMessage_sessionDisabled_singleUserAfterSystem(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "chat-1", "hi"); err != nil {
		t.Fatal(err)
	}
	if len(p.lastMessages) != 2 || p.lastMessages[0].Role != "system" || p.lastMessages[1].Role != "user" {
		t.Errorf("got %#v", p.lastMessages)
	}
}

// Covers AC-14.003: distinct session keys keep separate windows.
func TestHandleMessage_sessionMemory_distinctKeysIsolated(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r1"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "100", "only A"); err != nil {
		t.Fatal(err)
	}
	p.result = &llm.CompletionResult{Content: "r2"}
	if _, err := h.HandleMessage(ctx, 1, "200", "only B"); err != nil {
		t.Fatal(err)
	}
	if len(p.lastMessages) != 2 {
		t.Fatalf("second chat should have no history, got %d msgs", len(p.lastMessages))
	}
	if p.lastMessages[1].Content != "only B" {
		t.Errorf("user msg = %q", p.lastMessages[1].Content)
	}
}

// Covers AC-14.008: cap drops oldest exchange.
func TestHandleMessage_sessionMemory_capEvictsOldest(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 1},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	for _, text := range []string{"m1", "m2", "m3"} {
		if _, err := h.HandleMessage(ctx, 1, "k", text); err != nil {
			t.Fatal(err)
		}
	}
	msgs := p.lastMessages
	if len(msgs) != 4 {
		t.Fatalf("want 4 msgs (system + 1 pair + current), got %d", len(msgs))
	}
	if msgs[1].Content != "m2" || msgs[2].Content != "r" {
		t.Errorf("expected window u2/a2 before m3, got %#v", msgs)
	}
}

// Covers AC-14.009: empty user message does not grow session window.
func TestHandleMessage_sessionMemory_emptyUser_noAppend(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "   "); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "after"); err != nil {
		t.Fatal(err)
	}
	msgs := p.lastMessages
	if len(msgs) != 4 {
		t.Fatalf("want one stored exchange before 'after', got %d msgs", len(msgs))
	}
	if msgs[1].Content != "ok" {
		t.Errorf("first user in history = %q", msgs[1].Content)
	}
}

// Covers AC-14.009: over max length rejection does not append.
func TestHandleMessage_sessionMemory_overMaxLength_noAppend(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxMessageLength:      3,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "toobig"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "z"); err != nil {
		t.Fatal(err)
	}
	msgs := p.lastMessages
	if len(msgs) != 4 || msgs[1].Content != "ok" {
		t.Errorf("unexpected messages %#v", msgs)
	}
}

// Covers AC-14.011: vector path unchanged with session (no hits → still system + history + user).
func TestHandleMessage_sessionMemory_withVectorStoreEmpty_coexists(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	vec := &mockVectorStore{}
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}
	p := &mockProvider{result: &llm.CompletionResult{Content: "r1"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		memVec:                SingleStoreMemoryVectors(vec),
		embedder:              emb,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "one"); err != nil {
		t.Fatal(err)
	}
	p.result = &llm.CompletionResult{Content: "r2"}
	if _, err := h.HandleMessage(ctx, 1, "k", "two"); err != nil {
		t.Fatal(err)
	}
	if len(p.lastMessages) < 4 {
		t.Fatalf("expected history + user, got %d", len(p.lastMessages))
	}
	if !strings.Contains(p.lastMessages[0].Content, "Host rules in this message") {
		t.Error("expected merged system first")
	}
}

// Covers AC-14.013: DEBUG logs redact session history content like other user text.
func TestHandleMessage_sessionMemory_debugLogsRedactHistoryUserText(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelDebug}
	logger := slog.New(cap)
	secret := "SECRET_TOKEN_XYZ"
	p := &mockProvider{result: &llm.CompletionResult{Content: "r1"}}
	redact := func(s string) string { return strings.ReplaceAll(s, secret, "[REDACTED]") }
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		logRedactor:           redact,
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "hello "+secret); err != nil {
		t.Fatal(err)
	}
	p.result = &llm.CompletionResult{Content: "r2"}
	if _, err := h.HandleMessage(ctx, 1, "k", "next"); err != nil {
		t.Fatal(err)
	}
	for _, r := range cap.records {
		if c, ok := r.attrs["content"]; ok && strings.Contains(c, secret) {
			t.Errorf("debug log leaked secret in content attr: %q", c)
		}
	}
}
