package langfuse

import (
	"context"
	"time"

	"github.com/plexusone/omniobserve/llmops"
	sdk "github.com/plexusone/omniobserve/sdk/langfuse"
)

// spanCreator is an interface for types that can create child spans and generations.
type spanCreator interface {
	Span(ctx context.Context, name string, opts ...sdk.SpanOption) (context.Context, *sdk.Span, error)
	Generation(ctx context.Context, name string, opts ...sdk.GenerationOption) (context.Context, *sdk.Generation, error)
}

// createChildSpan is a helper that creates a child span or generation based on span type.
func createChildSpan(ctx context.Context, creator spanCreator, name string, cfg *llmops.SpanOptions) (context.Context, llmops.Span, error) {
	// Check if this should be a generation (LLM span)
	if cfg.Type == llmops.SpanTypeLLM {
		genOpts := []sdk.GenerationOption{}
		if cfg.Input != nil {
			genOpts = append(genOpts, sdk.WithGenerationInput(cfg.Input))
		}
		if cfg.Metadata != nil {
			genOpts = append(genOpts, sdk.WithGenerationMetadata(cfg.Metadata))
		}
		if cfg.Model != "" {
			genOpts = append(genOpts, sdk.WithModel(cfg.Model))
		}
		if cfg.Usage != nil {
			genOpts = append(genOpts, sdk.WithUsage(
				cfg.Usage.PromptTokens,
				cfg.Usage.CompletionTokens,
				cfg.Usage.TotalTokens,
			))
		}

		newCtx, gen, err := creator.Generation(ctx, name, genOpts...)
		if err != nil {
			return ctx, nil, err
		}
		return newCtx, &generationAdapter{gen: gen}, nil
	}

	// Regular span
	sdkOpts := []sdk.SpanOption{}
	if cfg.Input != nil {
		sdkOpts = append(sdkOpts, sdk.WithSpanInput(cfg.Input))
	}
	if cfg.Metadata != nil {
		sdkOpts = append(sdkOpts, sdk.WithSpanMetadata(cfg.Metadata))
	}

	newCtx, span, err := creator.Span(ctx, name, sdkOpts...)
	if err != nil {
		return ctx, nil, err
	}
	return newCtx, &spanAdapter{span: span}, nil
}

// traceAdapter adapts sdk.Trace to llmops.Trace.
type traceAdapter struct {
	trace *sdk.Trace
}

func (t *traceAdapter) ID() string {
	return t.trace.ID()
}

func (t *traceAdapter) Name() string {
	return t.trace.Name()
}

func (t *traceAdapter) StartSpan(ctx context.Context, name string, opts ...llmops.SpanOption) (context.Context, llmops.Span, error) {
	cfg := llmops.ApplySpanOptions(opts...)
	return createChildSpan(ctx, t.trace, name, cfg)
}

func (t *traceAdapter) SetInput(input any) error {
	return t.trace.Update(context.Background(), sdk.WithInput(input))
}

func (t *traceAdapter) SetOutput(output any) error {
	return t.trace.Update(context.Background(), sdk.WithOutput(output))
}

func (t *traceAdapter) SetMetadata(metadata map[string]any) error {
	return t.trace.Update(context.Background(), sdk.WithMetadata(metadata))
}

func (t *traceAdapter) AddTag(tag string) error {
	return t.trace.Update(context.Background(), sdk.WithTags(tag))
}

func (t *traceAdapter) AddFeedbackScore(ctx context.Context, name string, score float64, opts ...llmops.FeedbackOption) error {
	return t.trace.Score(ctx, name, score)
}

func (t *traceAdapter) End(opts ...llmops.EndOption) error {
	cfg := &llmops.EndOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	sdkOpts := []sdk.TraceOption{}
	if cfg.Output != nil {
		sdkOpts = append(sdkOpts, sdk.WithOutput(cfg.Output))
	}
	if cfg.Metadata != nil {
		sdkOpts = append(sdkOpts, sdk.WithMetadata(cfg.Metadata))
	}

	return t.trace.End(context.Background(), sdkOpts...)
}

func (t *traceAdapter) EndTime() *time.Time {
	return t.trace.EndTime()
}

func (t *traceAdapter) Duration() time.Duration {
	startTime := t.trace.StartTime()
	if endTime := t.trace.EndTime(); endTime != nil {
		return endTime.Sub(startTime)
	}
	return time.Since(startTime)
}

// spanAdapter adapts sdk.Span to llmops.Span.
type spanAdapter struct {
	span *sdk.Span
}

func (s *spanAdapter) ID() string {
	return s.span.ID()
}

func (s *spanAdapter) TraceID() string {
	return s.span.TraceID()
}

func (s *spanAdapter) ParentSpanID() string {
	return s.span.ParentSpanID()
}

func (s *spanAdapter) Name() string {
	return s.span.Name()
}

func (s *spanAdapter) Type() llmops.SpanType {
	return llmops.SpanTypeGeneral
}

func (s *spanAdapter) StartSpan(ctx context.Context, name string, opts ...llmops.SpanOption) (context.Context, llmops.Span, error) {
	cfg := llmops.ApplySpanOptions(opts...)
	return createChildSpan(ctx, s.span, name, cfg)
}

func (s *spanAdapter) SetInput(input any) error {
	return s.span.Update(context.Background(), sdk.WithSpanInput(input))
}

func (s *spanAdapter) SetOutput(output any) error {
	return s.span.Update(context.Background(), sdk.WithSpanOutput(output))
}

func (s *spanAdapter) SetMetadata(metadata map[string]any) error {
	return s.span.Update(context.Background(), sdk.WithSpanMetadata(metadata))
}

func (s *spanAdapter) SetModel(model string) error {
	// Spans don't have model in Langfuse, only generations do
	return nil
}

func (s *spanAdapter) SetProvider(provider string) error {
	return nil
}

func (s *spanAdapter) SetUsage(usage llmops.TokenUsage) error {
	// Spans don't have usage in Langfuse
	return nil
}

func (s *spanAdapter) AddTag(tag string) error {
	return nil
}

func (s *spanAdapter) AddFeedbackScore(ctx context.Context, name string, score float64, opts ...llmops.FeedbackOption) error {
	return s.span.Score(ctx, name, score)
}

func (s *spanAdapter) End(opts ...llmops.EndOption) error {
	cfg := &llmops.EndOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	sdkOpts := []sdk.SpanOption{}
	if cfg.Output != nil {
		sdkOpts = append(sdkOpts, sdk.WithSpanOutput(cfg.Output))
	}
	if cfg.Metadata != nil {
		sdkOpts = append(sdkOpts, sdk.WithSpanMetadata(cfg.Metadata))
	}

	return s.span.End(context.Background(), sdkOpts...)
}

func (s *spanAdapter) EndTime() *time.Time {
	return s.span.EndTime()
}

func (s *spanAdapter) Duration() time.Duration {
	startTime := s.span.StartTime()
	if endTime := s.span.EndTime(); endTime != nil {
		return endTime.Sub(startTime)
	}
	return time.Since(startTime)
}

// generationAdapter adapts sdk.Generation to llmops.Span.
type generationAdapter struct {
	gen *sdk.Generation
}

func (g *generationAdapter) ID() string {
	return g.gen.ID()
}

func (g *generationAdapter) TraceID() string {
	return g.gen.TraceID()
}

func (g *generationAdapter) ParentSpanID() string {
	return g.gen.ParentSpanID()
}

func (g *generationAdapter) Name() string {
	return g.gen.Name()
}

func (g *generationAdapter) Type() llmops.SpanType {
	return llmops.SpanTypeLLM
}

func (g *generationAdapter) StartSpan(ctx context.Context, name string, opts ...llmops.SpanOption) (context.Context, llmops.Span, error) {
	// Generations can't have child spans in Langfuse
	return ctx, nil, llmops.WrapNotImplemented("langfuse", "generation child spans")
}

func (g *generationAdapter) SetInput(input any) error {
	return g.gen.Update(context.Background(), sdk.WithGenerationInput(input))
}

func (g *generationAdapter) SetOutput(output any) error {
	return g.gen.SetOutput(output)
}

func (g *generationAdapter) SetMetadata(metadata map[string]any) error {
	return g.gen.Update(context.Background(), sdk.WithGenerationMetadata(metadata))
}

func (g *generationAdapter) SetModel(model string) error {
	return g.gen.Update(context.Background(), sdk.WithModel(model))
}

func (g *generationAdapter) SetProvider(provider string) error {
	// Store in metadata
	return g.gen.Update(context.Background(), sdk.WithGenerationMetadata(map[string]any{"provider": provider}))
}

func (g *generationAdapter) SetUsage(usage llmops.TokenUsage) error {
	return g.gen.SetUsage(usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
}

func (g *generationAdapter) AddTag(tag string) error {
	return nil
}

func (g *generationAdapter) AddFeedbackScore(ctx context.Context, name string, score float64, opts ...llmops.FeedbackOption) error {
	return g.gen.Score(ctx, name, score)
}

func (g *generationAdapter) End(opts ...llmops.EndOption) error {
	cfg := &llmops.EndOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	genOpts := []sdk.GenerationOption{}
	if cfg.Output != nil {
		genOpts = append(genOpts, sdk.WithGenerationOutput(cfg.Output))
	}
	if cfg.Metadata != nil {
		genOpts = append(genOpts, sdk.WithGenerationMetadata(cfg.Metadata))
	}

	return g.gen.End(context.Background(), genOpts...)
}

func (g *generationAdapter) EndTime() *time.Time {
	return g.gen.EndTime()
}

func (g *generationAdapter) Duration() time.Duration {
	startTime := g.gen.StartTime()
	if endTime := g.gen.EndTime(); endTime != nil {
		return endTime.Sub(startTime)
	}
	return time.Since(startTime)
}
