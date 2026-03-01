// Package slog provides an llmops.Provider that logs trace events to slog.
// This is useful for local development, debugging, or as a fallback when
// no observability platform is configured.
package slog

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/plexusone/omniobserve/llmops"
)

func init() {
	llmops.Register("slog", func(opts ...llmops.ClientOption) (llmops.Provider, error) {
		return New(opts...)
	})
	llmops.RegisterInfo(llmops.ProviderInfo{
		Name:        "slog",
		Description: "Local structured logging provider using slog",
		OpenSource:  true,
		SelfHosted:  true,
		Capabilities: []llmops.Capability{
			llmops.CapabilityTracing,
		},
	})
}

// Provider implements llmops.Provider using slog for local logging.
type Provider struct {
	logger  *slog.Logger
	project string
}

// New creates a new slog provider.
func New(opts ...llmops.ClientOption) (*Provider, error) {
	options := llmops.ApplyClientOptions(opts...)

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Provider{
		logger:  logger,
		project: options.ProjectName,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "slog"
}

// Close closes the provider.
func (p *Provider) Close() error {
	return nil
}

// StartTrace begins a new trace.
func (p *Provider) StartTrace(ctx context.Context, name string, opts ...llmops.TraceOption) (context.Context, llmops.Trace, error) {
	options := llmops.ApplyTraceOptions(opts...)
	traceID := uuid.New().String()

	p.logger.Info("trace started",
		"trace_id", traceID,
		"name", name,
		"project", p.project,
	)

	trace := &slogTrace{
		id:        traceID,
		name:      name,
		startTime: time.Now(),
		logger:    p.logger,
		input:     options.Input,
		provider:  p,
	}

	ctx = context.WithValue(ctx, traceContextKey{}, trace)
	return ctx, trace, nil
}

// StartSpan begins a new span within the current trace context.
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...llmops.SpanOption) (context.Context, llmops.Span, error) {
	options := llmops.ApplySpanOptions(opts...)
	spanID := uuid.New().String()

	var traceID string
	if trace, ok := p.TraceFromContext(ctx); ok {
		traceID = trace.ID()
	}

	var parentSpanID string
	if parentSpan, ok := p.SpanFromContext(ctx); ok {
		parentSpanID = parentSpan.ID()
	}

	p.logger.Info("span started",
		"span_id", spanID,
		"trace_id", traceID,
		"parent_span_id", parentSpanID,
		"name", name,
		"type", options.Type,
		"model", options.Model,
		"provider", options.Provider,
	)

	span := &slogSpan{
		id:           spanID,
		traceID:      traceID,
		parentSpanID: parentSpanID,
		name:         name,
		startTime:    time.Now(),
		logger:       p.logger,
		spanType:     options.Type,
		model:        options.Model,
		llmProvider:  options.Provider,
		input:        options.Input,
		provider:     p,
	}

	ctx = context.WithValue(ctx, spanContextKey{}, span)
	return ctx, span, nil
}

// TraceFromContext retrieves the current trace from context.
func (p *Provider) TraceFromContext(ctx context.Context) (llmops.Trace, bool) {
	trace, ok := ctx.Value(traceContextKey{}).(*slogTrace)
	return trace, ok
}

// SpanFromContext retrieves the current span from context.
func (p *Provider) SpanFromContext(ctx context.Context) (llmops.Span, bool) {
	span, ok := ctx.Value(spanContextKey{}).(*slogSpan)
	return span, ok
}

// Evaluate is not supported by the slog provider.
func (p *Provider) Evaluate(ctx context.Context, input llmops.EvalInput, metrics ...llmops.Metric) (*llmops.EvalResult, error) {
	return nil, llmops.ErrNotImplemented
}

// AddFeedbackScore is not supported by the slog provider.
func (p *Provider) AddFeedbackScore(ctx context.Context, opts llmops.FeedbackScoreOpts) error {
	return llmops.ErrNotImplemented
}

// CreatePrompt is not supported by the slog provider.
func (p *Provider) CreatePrompt(ctx context.Context, name string, template string, opts ...llmops.PromptOption) (*llmops.Prompt, error) {
	return nil, llmops.ErrNotImplemented
}

// GetPrompt is not supported by the slog provider.
func (p *Provider) GetPrompt(ctx context.Context, name string, version ...string) (*llmops.Prompt, error) {
	return nil, llmops.ErrNotImplemented
}

// ListPrompts is not supported by the slog provider.
func (p *Provider) ListPrompts(ctx context.Context, opts ...llmops.ListOption) ([]*llmops.Prompt, error) {
	return nil, llmops.ErrNotImplemented
}

// CreateDataset is not supported by the slog provider.
func (p *Provider) CreateDataset(ctx context.Context, name string, opts ...llmops.DatasetOption) (*llmops.Dataset, error) {
	return nil, llmops.ErrNotImplemented
}

// GetDataset is not supported by the slog provider.
func (p *Provider) GetDataset(ctx context.Context, name string) (*llmops.Dataset, error) {
	return nil, llmops.ErrNotImplemented
}

// GetDatasetByID is not supported by the slog provider.
func (p *Provider) GetDatasetByID(ctx context.Context, id string) (*llmops.Dataset, error) {
	return nil, llmops.ErrNotImplemented
}

// AddDatasetItems is not supported by the slog provider.
func (p *Provider) AddDatasetItems(ctx context.Context, datasetName string, items []llmops.DatasetItem) error {
	return llmops.ErrNotImplemented
}

// ListDatasets is not supported by the slog provider.
func (p *Provider) ListDatasets(ctx context.Context, opts ...llmops.ListOption) ([]*llmops.Dataset, error) {
	return nil, llmops.ErrNotImplemented
}

// DeleteDataset is not supported by the slog provider.
func (p *Provider) DeleteDataset(ctx context.Context, datasetID string) error {
	return llmops.ErrNotImplemented
}

// CreateProject is not supported by the slog provider.
func (p *Provider) CreateProject(ctx context.Context, name string, opts ...llmops.ProjectOption) (*llmops.Project, error) {
	return nil, llmops.ErrNotImplemented
}

// GetProject is not supported by the slog provider.
func (p *Provider) GetProject(ctx context.Context, name string) (*llmops.Project, error) {
	return nil, llmops.ErrNotImplemented
}

// ListProjects is not supported by the slog provider.
func (p *Provider) ListProjects(ctx context.Context, opts ...llmops.ListOption) ([]*llmops.Project, error) {
	return nil, llmops.ErrNotImplemented
}

// SetProject sets the current project.
func (p *Provider) SetProject(ctx context.Context, name string) error {
	p.project = name
	return nil
}

// CreateAnnotation is not supported by the slog provider.
func (p *Provider) CreateAnnotation(ctx context.Context, annotation llmops.Annotation) error {
	return llmops.ErrNotImplemented
}

// ListAnnotations is not supported by the slog provider.
func (p *Provider) ListAnnotations(ctx context.Context, opts llmops.ListAnnotationsOptions) ([]*llmops.Annotation, error) {
	return nil, llmops.ErrNotImplemented
}

// Context keys for storing trace and span.
type traceContextKey struct{}
type spanContextKey struct{}

// slogTrace implements llmops.Trace.
type slogTrace struct {
	id        string
	name      string
	startTime time.Time
	endTime   *time.Time
	logger    *slog.Logger
	provider  *Provider
	input     any
	output    any
	metadata  map[string]any
	tags      []string
	mu        sync.Mutex
}

func (t *slogTrace) ID() string {
	return t.id
}

func (t *slogTrace) Name() string {
	return t.name
}

func (t *slogTrace) StartSpan(ctx context.Context, name string, opts ...llmops.SpanOption) (context.Context, llmops.Span, error) {
	return t.provider.StartSpan(ctx, name, opts...)
}

func (t *slogTrace) SetInput(input any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.input = input
	return nil
}

func (t *slogTrace) SetOutput(output any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.output = output
	return nil
}

func (t *slogTrace) SetMetadata(metadata map[string]any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metadata = metadata
	return nil
}

func (t *slogTrace) AddTag(tag string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tags = append(t.tags, tag)
	return nil
}

func (t *slogTrace) AddFeedbackScore(ctx context.Context, name string, score float64, opts ...llmops.FeedbackOption) error {
	t.logger.Info("feedback score added",
		"trace_id", t.id,
		"name", name,
		"score", score,
	)
	return nil
}

func (t *slogTrace) End(opts ...llmops.EndOption) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	options := &llmops.EndOptions{}
	for _, opt := range opts {
		opt(options)
	}

	now := time.Now()
	t.endTime = &now
	duration := now.Sub(t.startTime)

	if options.Error != nil {
		t.logger.Error("trace ended",
			"trace_id", t.id,
			"name", t.name,
			"duration", duration,
			"error", options.Error,
		)
	} else {
		t.logger.Info("trace ended",
			"trace_id", t.id,
			"name", t.name,
			"duration", duration,
		)
	}
	return nil
}

func (t *slogTrace) EndTime() *time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.endTime
}

func (t *slogTrace) Duration() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.endTime != nil {
		return t.endTime.Sub(t.startTime)
	}
	return time.Since(t.startTime)
}

// slogSpan implements llmops.Span.
type slogSpan struct {
	id           string
	traceID      string
	parentSpanID string
	name         string
	startTime    time.Time
	endTime      *time.Time
	logger       *slog.Logger
	provider     *Provider
	spanType     llmops.SpanType
	model        string
	llmProvider  string
	input        any
	output       any
	metadata     map[string]any
	tags         []string
	usage        *llmops.TokenUsage
	mu           sync.Mutex
}

func (s *slogSpan) ID() string {
	return s.id
}

func (s *slogSpan) TraceID() string {
	return s.traceID
}

func (s *slogSpan) ParentSpanID() string {
	return s.parentSpanID
}

func (s *slogSpan) Name() string {
	return s.name
}

func (s *slogSpan) Type() llmops.SpanType {
	return s.spanType
}

func (s *slogSpan) StartSpan(ctx context.Context, name string, opts ...llmops.SpanOption) (context.Context, llmops.Span, error) {
	// Add this span as parent
	opts = append(opts, llmops.WithParentSpan(s.id))
	return s.provider.StartSpan(ctx, name, opts...)
}

func (s *slogSpan) SetInput(input any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.input = input
	return nil
}

func (s *slogSpan) SetOutput(output any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.output = output
	return nil
}

func (s *slogSpan) SetMetadata(metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata = metadata
	return nil
}

func (s *slogSpan) SetModel(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
	return nil
}

func (s *slogSpan) SetProvider(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llmProvider = provider
	return nil
}

func (s *slogSpan) SetUsage(usage llmops.TokenUsage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage = &usage
	return nil
}

func (s *slogSpan) AddTag(tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tags = append(s.tags, tag)
	return nil
}

func (s *slogSpan) AddFeedbackScore(ctx context.Context, name string, score float64, opts ...llmops.FeedbackOption) error {
	s.logger.Info("feedback score added",
		"span_id", s.id,
		"trace_id", s.traceID,
		"name", name,
		"score", score,
	)
	return nil
}

func (s *slogSpan) End(opts ...llmops.EndOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	options := &llmops.EndOptions{}
	for _, opt := range opts {
		opt(options)
	}

	now := time.Now()
	s.endTime = &now
	duration := now.Sub(s.startTime)

	attrs := []any{
		"span_id", s.id,
		"trace_id", s.traceID,
		"name", s.name,
		"duration", duration,
	}

	if s.model != "" {
		attrs = append(attrs, "model", s.model)
	}
	if s.llmProvider != "" {
		attrs = append(attrs, "llm_provider", s.llmProvider)
	}
	if s.usage != nil {
		attrs = append(attrs,
			"prompt_tokens", s.usage.PromptTokens,
			"completion_tokens", s.usage.CompletionTokens,
			"total_tokens", s.usage.TotalTokens,
		)
	}

	if options.Error != nil {
		attrs = append(attrs, "error", options.Error)
		s.logger.Error("span ended", attrs...)
	} else {
		s.logger.Info("span ended", attrs...)
	}
	return nil
}

func (s *slogSpan) EndTime() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.endTime
}

func (s *slogSpan) Duration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.endTime != nil {
		return s.endTime.Sub(s.startTime)
	}
	return time.Since(s.startTime)
}

// Ensure Provider implements llmops.Provider.
var _ llmops.Provider = (*Provider)(nil)

// Ensure slogTrace implements llmops.Trace.
var _ llmops.Trace = (*slogTrace)(nil)

// Ensure slogSpan implements llmops.Span.
var _ llmops.Span = (*slogSpan)(nil)
