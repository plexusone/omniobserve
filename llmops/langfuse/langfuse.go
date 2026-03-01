// Package langfuse provides a Langfuse adapter for the llmops abstraction.
//
// Import this package to register the Langfuse provider:
//
//	import _ "github.com/plexusone/omniobserve/llmops/langfuse"
//
// Then open it:
//
//	provider, err := llmops.Open("langfuse",
//		llmops.WithAPIKey("pk-..."),      // Public key
//		llmops.WithWorkspace("sk-..."),   // Secret key (using workspace field)
//	)
package langfuse

import (
	"context"
	"time"

	"github.com/plexusone/omniobserve/llmops"
	sdk "github.com/plexusone/omniobserve/sdk/langfuse"
)

const ProviderName = "langfuse"

func init() {
	llmops.Register(ProviderName, New)
	llmops.RegisterInfo(llmops.ProviderInfo{
		Name:        ProviderName,
		Description: "Langfuse - Open-source LLM observability and analytics",
		Website:     "https://langfuse.com",
		OpenSource:  true,
		SelfHosted:  true,
		Capabilities: []llmops.Capability{
			llmops.CapabilityTracing,
			llmops.CapabilityEvaluation,
			llmops.CapabilityPrompts,
			llmops.CapabilityDatasets,
			llmops.CapabilityExperiments,
			llmops.CapabilityStreaming,
			llmops.CapabilityCostTracking,
		},
	})
}

// Provider implements llmops.Provider for Langfuse.
type Provider struct {
	client *sdk.Client
}

// New creates a new Langfuse provider.
// Note: For Langfuse, use:
//   - APIKey for the public key
//   - Workspace for the secret key
func New(opts ...llmops.ClientOption) (llmops.Provider, error) {
	cfg := llmops.ApplyClientOptions(opts...)

	sdkOpts := []sdk.Option{}
	if cfg.APIKey != "" {
		sdkOpts = append(sdkOpts, sdk.WithPublicKey(cfg.APIKey))
	}
	if cfg.Workspace != "" {
		sdkOpts = append(sdkOpts, sdk.WithSecretKey(cfg.Workspace))
	}
	if cfg.Endpoint != "" {
		sdkOpts = append(sdkOpts, sdk.WithEndpoint(cfg.Endpoint))
	}
	if cfg.HTTPClient != nil {
		sdkOpts = append(sdkOpts, sdk.WithHTTPClient(cfg.HTTPClient))
	}
	if cfg.Timeout > 0 {
		sdkOpts = append(sdkOpts, sdk.WithTimeout(cfg.Timeout))
	}
	if cfg.Disabled {
		sdkOpts = append(sdkOpts, sdk.WithDisabled(true))
	}
	if cfg.Debug {
		sdkOpts = append(sdkOpts, sdk.WithDebug(true))
	}

	client, err := sdk.NewClient(sdkOpts...)
	if err != nil {
		return nil, err
	}

	return &Provider{client: client}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Close closes the provider and flushes pending events.
func (p *Provider) Close() error {
	return p.client.Close()
}

// StartTrace starts a new trace.
func (p *Provider) StartTrace(ctx context.Context, name string, opts ...llmops.TraceOption) (context.Context, llmops.Trace, error) {
	cfg := llmops.ApplyTraceOptions(opts...)

	sdkOpts := []sdk.TraceOption{}
	if cfg.Input != nil {
		sdkOpts = append(sdkOpts, sdk.WithInput(cfg.Input))
	}
	if cfg.Metadata != nil {
		sdkOpts = append(sdkOpts, sdk.WithMetadata(cfg.Metadata))
	}
	if len(cfg.Tags) > 0 {
		sdkOpts = append(sdkOpts, sdk.WithTags(cfg.Tags...))
	}
	if cfg.ThreadID != "" {
		sdkOpts = append(sdkOpts, sdk.WithSessionID(cfg.ThreadID))
	}

	newCtx, trace, err := p.client.StartTrace(ctx, name, sdkOpts...)
	if err != nil {
		return ctx, nil, err
	}

	return newCtx, &traceAdapter{trace: trace}, nil
}

// StartSpan starts a new span.
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...llmops.SpanOption) (context.Context, llmops.Span, error) {
	cfg := llmops.ApplySpanOptions(opts...)

	sdkOpts := []sdk.SpanOption{}
	if cfg.Input != nil {
		sdkOpts = append(sdkOpts, sdk.WithSpanInput(cfg.Input))
	}
	if cfg.Metadata != nil {
		sdkOpts = append(sdkOpts, sdk.WithSpanMetadata(cfg.Metadata))
	}

	newCtx, span, err := sdk.StartSpan(ctx, name, sdkOpts...)
	if err != nil {
		return ctx, nil, err
	}

	return newCtx, &spanAdapter{span: span}, nil
}

// TraceFromContext gets the current trace from context.
func (p *Provider) TraceFromContext(ctx context.Context) (llmops.Trace, bool) {
	trace := sdk.TraceFromContext(ctx)
	if trace == nil {
		return nil, false
	}
	return &traceAdapter{trace: trace}, true
}

// SpanFromContext gets the current span from context.
func (p *Provider) SpanFromContext(ctx context.Context) (llmops.Span, bool) {
	span := sdk.SpanFromContext(ctx)
	if span == nil {
		// Also check for generation
		gen := sdk.GenerationFromContext(ctx)
		if gen != nil {
			return &generationAdapter{gen: gen}, true
		}
		return nil, false
	}
	return &spanAdapter{span: span}, true
}

// Evaluate runs evaluation metrics.
func (p *Provider) Evaluate(ctx context.Context, input llmops.EvalInput, metrics ...llmops.Metric) (*llmops.EvalResult, error) {
	startTime := time.Now()

	scores := make([]llmops.MetricScore, 0, len(metrics))
	for _, metric := range metrics {
		score, err := metric.Evaluate(input)
		if err != nil {
			scores = append(scores, llmops.MetricScore{
				Name:  metric.Name(),
				Error: err.Error(),
			})
		} else {
			scores = append(scores, score)
		}
	}

	return &llmops.EvalResult{
		Scores:   scores,
		Duration: time.Since(startTime),
	}, nil
}

// AddFeedbackScore adds a feedback score.
func (p *Provider) AddFeedbackScore(ctx context.Context, opts llmops.FeedbackScoreOpts) error {
	// Try span/generation first, then trace
	if span := sdk.SpanFromContext(ctx); span != nil {
		return span.Score(ctx, opts.Name, opts.Score)
	}
	if gen := sdk.GenerationFromContext(ctx); gen != nil {
		return gen.Score(ctx, opts.Name, opts.Score)
	}
	if trace := sdk.TraceFromContext(ctx); trace != nil {
		return trace.Score(ctx, opts.Name, opts.Score)
	}
	return llmops.ErrNoActiveTrace
}

// CreatePrompt creates a new prompt (not directly supported in Langfuse SDK).
func (p *Provider) CreatePrompt(ctx context.Context, name string, template string, opts ...llmops.PromptOption) (*llmops.Prompt, error) {
	// Langfuse prompts are managed through the UI primarily
	// This would require direct API calls
	return nil, llmops.WrapNotImplemented(ProviderName, "CreatePrompt")
}

// GetPrompt gets a prompt by name.
func (p *Provider) GetPrompt(ctx context.Context, name string, version ...string) (*llmops.Prompt, error) {
	return nil, llmops.WrapNotImplemented(ProviderName, "GetPrompt")
}

// ListPrompts lists prompts.
func (p *Provider) ListPrompts(ctx context.Context, opts ...llmops.ListOption) ([]*llmops.Prompt, error) {
	return nil, llmops.WrapNotImplemented(ProviderName, "ListPrompts")
}

// CreateDataset creates a new dataset.
func (p *Provider) CreateDataset(ctx context.Context, name string, opts ...llmops.DatasetOption) (*llmops.Dataset, error) {
	cfg := &llmops.DatasetOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	sdkOpts := []sdk.DatasetOption{}
	if cfg.Description != "" {
		sdkOpts = append(sdkOpts, sdk.WithDatasetDescription(cfg.Description))
	}

	dataset, err := p.client.CreateDataset(ctx, name, sdkOpts...)
	if err != nil {
		return nil, err
	}

	return &llmops.Dataset{
		ID:          dataset.ID,
		Name:        dataset.Name,
		Description: dataset.Description,
		CreatedAt:   dataset.CreatedAt,
		UpdatedAt:   dataset.UpdatedAt,
	}, nil
}

// GetDataset gets a dataset by name.
func (p *Provider) GetDataset(ctx context.Context, name string) (*llmops.Dataset, error) {
	dataset, err := p.client.GetDataset(ctx, name)
	if err != nil {
		return nil, err
	}

	return &llmops.Dataset{
		ID:          dataset.ID,
		Name:        dataset.Name,
		Description: dataset.Description,
		CreatedAt:   dataset.CreatedAt,
		UpdatedAt:   dataset.UpdatedAt,
	}, nil
}

// AddDatasetItems adds items to a dataset.
func (p *Provider) AddDatasetItems(ctx context.Context, datasetName string, items []llmops.DatasetItem) error {
	for _, item := range items {
		sdkItem := sdk.DatasetItem{
			Input:          item.Input,
			ExpectedOutput: item.Expected,
		}
		_, err := p.client.CreateDatasetItem(ctx, datasetName, sdkItem)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListDatasets lists datasets.
func (p *Provider) ListDatasets(ctx context.Context, opts ...llmops.ListOption) ([]*llmops.Dataset, error) {
	cfg := llmops.ApplyListOptions(opts...)

	page := 1
	if cfg.Offset > 0 && cfg.Limit > 0 {
		page = (cfg.Offset / cfg.Limit) + 1
	}

	datasets, err := p.client.ListDatasets(ctx, cfg.Limit, page)
	if err != nil {
		return nil, err
	}

	result := make([]*llmops.Dataset, len(datasets))
	for i, ds := range datasets {
		result[i] = &llmops.Dataset{
			ID:          ds.ID,
			Name:        ds.Name,
			Description: ds.Description,
			CreatedAt:   ds.CreatedAt,
			UpdatedAt:   ds.UpdatedAt,
		}
	}
	return result, nil
}

// CreateProject is not supported in Langfuse (projects are managed via UI).
func (p *Provider) CreateProject(ctx context.Context, name string, opts ...llmops.ProjectOption) (*llmops.Project, error) {
	return nil, llmops.WrapNotImplemented(ProviderName, "CreateProject")
}

// GetProject is not supported in Langfuse.
func (p *Provider) GetProject(ctx context.Context, name string) (*llmops.Project, error) {
	return nil, llmops.WrapNotImplemented(ProviderName, "GetProject")
}

// ListProjects is not supported in Langfuse.
func (p *Provider) ListProjects(ctx context.Context, opts ...llmops.ListOption) ([]*llmops.Project, error) {
	return nil, llmops.WrapNotImplemented(ProviderName, "ListProjects")
}

// SetProject is not supported in Langfuse.
func (p *Provider) SetProject(ctx context.Context, name string) error {
	return llmops.WrapNotImplemented(ProviderName, "SetProject")
}

// GetDatasetByID is not supported in Langfuse.
func (p *Provider) GetDatasetByID(ctx context.Context, id string) (*llmops.Dataset, error) {
	return nil, llmops.WrapNotImplemented(ProviderName, "GetDatasetByID")
}

// DeleteDataset is not supported in Langfuse.
func (p *Provider) DeleteDataset(ctx context.Context, datasetID string) error {
	return llmops.WrapNotImplemented(ProviderName, "DeleteDataset")
}

// CreateAnnotation is not directly supported in Langfuse.
func (p *Provider) CreateAnnotation(ctx context.Context, annotation llmops.Annotation) error {
	return llmops.WrapNotImplemented(ProviderName, "CreateAnnotation")
}

// ListAnnotations is not supported in Langfuse.
func (p *Provider) ListAnnotations(ctx context.Context, opts llmops.ListAnnotationsOptions) ([]*llmops.Annotation, error) {
	return nil, llmops.WrapNotImplemented(ProviderName, "ListAnnotations")
}
