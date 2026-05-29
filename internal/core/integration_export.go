//go:build integration

package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/runtimeskills"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"pa/internal/vector"
	"testing"
)

// IntegrationConstEmbedder returns a fixed 4-dimensional vector for every text (handler / skill index tests).
type IntegrationConstEmbedder struct{}

func (IntegrationConstEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{1, 0, 0, 0}, nil
}

var _ embedding.Embedder = IntegrationConstEmbedder{}

// IntegrationMockLLM captures Complete inputs; use Result/Err or CompleteFn for behaviour.
type IntegrationMockLLM struct {
	Result       *llm.CompletionResult
	Err          error
	LastMessages []llm.Message
	LastOpts     *llm.CompletionOptions
	CompleteFn   func(context.Context, []llm.Message, *llm.CompletionOptions) (*llm.CompletionResult, error)
}

func (m *IntegrationMockLLM) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.LastMessages = messages
	m.LastOpts = opts
	if m.CompleteFn != nil {
		return m.CompleteFn(ctx, messages, opts)
	}
	if m.Result != nil {
		return m.Result, m.Err
	}
	return &llm.CompletionResult{}, m.Err
}

// IntegrationMockVectorStore is a test double for vector.Store (tool/memory mocks).
type IntegrationMockVectorStore struct {
	AddErr        error
	AddCalls      int
	AddChunks     []string
	SearchResults []vector.SearchResult
	SearchErr     error
}

func (m *IntegrationMockVectorStore) Add(_ context.Context, _ string, _ []float32, chunk string) error {
	m.AddCalls++
	m.AddChunks = append(m.AddChunks, chunk)
	return m.AddErr
}

func (m *IntegrationMockVectorStore) Delete(context.Context, string) error { return nil }

func (m *IntegrationMockVectorStore) Clear(context.Context) error { return nil }

func (m *IntegrationMockVectorStore) Search(_ context.Context, _ []float32, _ int) ([]vector.SearchResult, error) {
	if m.SearchErr != nil {
		return nil, m.SearchErr
	}
	if m.SearchResults != nil {
		return m.SearchResults, nil
	}
	return nil, nil
}

func (m *IntegrationMockVectorStore) Exists(context.Context, string) (bool, error) { return false, nil }

func (m *IntegrationMockVectorStore) Close() error { return nil }

// IntegrationMockToolIndex implements ToolIndex for handler tests.
type IntegrationMockToolIndex struct {
	StoreObj  vector.Store
	ReadyFlag bool
}

func (m *IntegrationMockToolIndex) Store() vector.Store { return m.StoreObj }
func (m *IntegrationMockToolIndex) Ready() bool         { return m.ReadyFlag }

// IntegrationMockNodeRunner implements NodeRunner for catalog tool execution tests.
type IntegrationMockNodeRunner struct {
	LastNodeID  string
	LastCommand string
	Stdout      string
	Err         error
	RunFunc     func(ctx context.Context, nodeID, command string) (string, error)
}

func (m *IntegrationMockNodeRunner) RunOnNode(ctx context.Context, nodeID, command string) (string, error) {
	m.LastNodeID = nodeID
	m.LastCommand = command
	if m.RunFunc != nil {
		return m.RunFunc(ctx, nodeID, command)
	}
	if m.Err != nil {
		return "", m.Err
	}
	return m.Stdout, nil
}

// IntegrationConversationParams wires conversationHandler for tests/integration (runtime skills, markers, tools).
type IntegrationConversationParams struct {
	Router *llmrouter.Router
	// VectorStore is used when MemoryVectors is nil: handler gets SingleStoreMemoryVectors(VectorStore).
	VectorStore                vector.Store
	MemoryVectors              *MemoryVectors
	Embedder                   embedding.Embedder
	NodeRunner                 NodeRunner
	ToolIndex                  ToolIndex
	SkillIndex                 SkillIndex
	RuntimeSkillsCfg           *config.RuntimeSkillsConfig
	ToolsCfg                   *config.ToolsConfig
	SkillPackagesByID          map[string]*runtimeskills.Package
	NativeRegistry             *tools.Registry
	Catalog                    *toolcatalog.Catalog
	ToolSearchTopK             int
	ToolMinCount               int
	ToolFallbackCap            int
	Logger                     *slog.Logger
	MaxMessageLength           int
	MaxDynamicSystemRunes      int
	MemoryVector               config.MemoryVectorConfig
	LLMLog                     llmlog.Writer
	Model                      string
	LogRedactor                func(string) string
	FirstProviderSupportsTools bool
	ConversationSession        *config.ConversationSessionConfig
}

// NewIntegrationConversationHandler returns a MessageHandler for integration tests.
func NewIntegrationConversationHandler(p IntegrationConversationParams) MessageHandler {
	if p.Logger == nil {
		p.Logger = slog.New(slog.DiscardHandler)
	}
	if p.ToolSearchTopK == 0 {
		p.ToolSearchTopK = 10
	}
	if p.ToolMinCount == 0 {
		p.ToolMinCount = 1
	}
	if p.ToolFallbackCap == 0 {
		p.ToolFallbackCap = 50
	}
	if p.MaxDynamicSystemRunes == 0 {
		p.MaxDynamicSystemRunes = 4000
	}
	if p.MemoryVector.NotesTopK == 0 && p.MemoryVector.SummariesTopK == 0 && p.MemoryVector.TurnsTopK == 0 {
		p.MemoryVector = config.MemoryVectorConfig{NotesTopK: 10, SummariesTopK: 10, TurnsTopK: 10}
	}
	mv := p.MemoryVectors
	if mv == nil {
		mv = SingleStoreMemoryVectors(p.VectorStore)
	}
	var sessCfg *config.ConversationSessionConfig
	var sessStore *sessionWindowStore
	if p.ConversationSession != nil && p.ConversationSession.Enabled {
		sessCfg = p.ConversationSession
		sessStore = newSessionWindowStore()
	}
	return &conversationHandler{
		router:                     p.Router,
		memVec:                     mv,
		embedder:                   p.Embedder,
		nodeRunner:                 p.NodeRunner,
		toolIndex:                  p.ToolIndex,
		skillIndex:                 p.SkillIndex,
		runtimeSkillsCfg:           p.RuntimeSkillsCfg,
		toolsCfg:                   p.ToolsCfg,
		skillPackagesByID:          p.SkillPackagesByID,
		nativeRegistry:             p.NativeRegistry,
		catalog:                    p.Catalog,
		toolSearchTopK:             p.ToolSearchTopK,
		toolMinCount:               p.ToolMinCount,
		toolFallbackCap:            p.ToolFallbackCap,
		logger:                     p.Logger,
		maxMessageLength:           p.MaxMessageLength,
		maxDynamicSystemRunes:      p.MaxDynamicSystemRunes,
		memoryVectorTopK:           p.MemoryVector,
		llmLog:                     p.LLMLog,
		model:                      p.Model,
		logRedactor:                p.LogRedactor,
		firstProviderSupportsTools: p.FirstProviderSupportsTools,
		sessionCfg:                 sessCfg,
		sessionStore:               sessStore,
	}
}

// IntegrationMustRouterSingle builds a single-provider router (test label "test/default").
func IntegrationMustRouterSingle(tb testing.TB, provider llm.Provider) *llmrouter.Router {
	tb.Helper()
	r, err := llmrouter.New([]llm.Provider{provider}, []string{"test/default"}, llmrouter.Config{}, slog.Default())
	if err != nil {
		tb.Fatalf("llmrouter.New: %v", err)
	}
	return r
}

// IntegrationIndexTurn exposes indexTurn for integration tests (forbidden marker rejection).
func IntegrationIndexTurn(h MessageHandler, ctx context.Context, userText, reply string) error {
	ch, ok := h.(*conversationHandler)
	if !ok {
		return fmt.Errorf("integration: handler is not a core conversation handler")
	}
	return ch.indexTurn(ctx, userText, reply)
}
