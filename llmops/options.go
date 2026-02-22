package llmops

import (
	"log/slog"
	"net/http"
	"time"
)

// TraceOption configures trace creation.
type TraceOption func(*TraceOptions)

// TraceOptions holds trace configuration.
type TraceOptions struct {
	ProjectName string
	Input       any
	Output      any
	Metadata    map[string]any
	Tags        []string
	ThreadID    string // For conversation threading
}

// WithTraceProject sets the project for the trace.
func WithTraceProject(project string) TraceOption {
	return func(o *TraceOptions) {
		o.ProjectName = project
	}
}

// WithTraceInput sets the initial input for the trace.
func WithTraceInput(input any) TraceOption {
	return func(o *TraceOptions) {
		o.Input = input
	}
}

// WithTraceOutput sets the initial output for the trace.
func WithTraceOutput(output any) TraceOption {
	return func(o *TraceOptions) {
		o.Output = output
	}
}

// WithTraceMetadata sets initial metadata for the trace.
func WithTraceMetadata(metadata map[string]any) TraceOption {
	return func(o *TraceOptions) {
		o.Metadata = metadata
	}
}

// WithTraceTags sets initial tags for the trace.
func WithTraceTags(tags ...string) TraceOption {
	return func(o *TraceOptions) {
		o.Tags = tags
	}
}

// WithThreadID sets the thread ID for conversation tracking.
func WithThreadID(threadID string) TraceOption {
	return func(o *TraceOptions) {
		o.ThreadID = threadID
	}
}

// SpanOption configures span creation.
type SpanOption func(*SpanOptions)

// SpanOptions holds span configuration.
type SpanOptions struct {
	Type         SpanType
	Input        any
	Output       any
	Metadata     map[string]any
	Tags         []string
	Model        string
	Provider     string
	Usage        *TokenUsage
	ParentSpanID string
}

// WithSpanType sets the span type.
func WithSpanType(t SpanType) SpanOption {
	return func(o *SpanOptions) {
		o.Type = t
	}
}

// WithSpanInput sets the initial input for the span.
func WithSpanInput(input any) SpanOption {
	return func(o *SpanOptions) {
		o.Input = input
	}
}

// WithSpanOutput sets the initial output for the span.
func WithSpanOutput(output any) SpanOption {
	return func(o *SpanOptions) {
		o.Output = output
	}
}

// WithSpanMetadata sets initial metadata for the span.
func WithSpanMetadata(metadata map[string]any) SpanOption {
	return func(o *SpanOptions) {
		o.Metadata = metadata
	}
}

// WithSpanTags sets initial tags for the span.
func WithSpanTags(tags ...string) SpanOption {
	return func(o *SpanOptions) {
		o.Tags = tags
	}
}

// WithModel sets the LLM model name.
func WithModel(model string) SpanOption {
	return func(o *SpanOptions) {
		o.Model = model
	}
}

// WithProvider sets the LLM provider name.
func WithProvider(provider string) SpanOption {
	return func(o *SpanOptions) {
		o.Provider = provider
	}
}

// WithTokenUsage sets token usage information.
func WithTokenUsage(prompt, completion int) SpanOption {
	return func(o *SpanOptions) {
		o.Usage = &TokenUsage{
			PromptTokens:     prompt,
			CompletionTokens: completion,
			TotalTokens:      prompt + completion,
		}
	}
}

// WithTokenCost sets token cost information.
func WithTokenCost(promptCost, completionCost float64, currency string) SpanOption {
	return func(o *SpanOptions) {
		if o.Usage == nil {
			o.Usage = &TokenUsage{}
		}
		o.Usage.PromptCost = promptCost
		o.Usage.CompletionCost = completionCost
		o.Usage.TotalCost = promptCost + completionCost
		o.Usage.Currency = currency
	}
}

// WithParentSpan sets the parent span ID.
func WithParentSpan(parentSpanID string) SpanOption {
	return func(o *SpanOptions) {
		o.ParentSpanID = parentSpanID
	}
}

// EndOption configures trace/span ending.
type EndOption func(*EndOptions)

// EndOptions holds end configuration.
type EndOptions struct {
	Output   any
	Metadata map[string]any
	Error    error
}

// WithEndOutput sets the final output when ending.
func WithEndOutput(output any) EndOption {
	return func(o *EndOptions) {
		o.Output = output
	}
}

// WithEndMetadata sets additional metadata when ending.
func WithEndMetadata(metadata map[string]any) EndOption {
	return func(o *EndOptions) {
		o.Metadata = metadata
	}
}

// WithEndError records an error when ending.
func WithEndError(err error) EndOption {
	return func(o *EndOptions) {
		o.Error = err
	}
}

// FeedbackOption configures feedback score creation.
type FeedbackOption func(*FeedbackOptions)

// FeedbackOptions holds feedback configuration.
type FeedbackOptions struct {
	Reason   string
	Category string
	Source   string
}

// WithFeedbackReason sets the reason for the score.
func WithFeedbackReason(reason string) FeedbackOption {
	return func(o *FeedbackOptions) {
		o.Reason = reason
	}
}

// WithFeedbackCategory sets the category for the score.
func WithFeedbackCategory(category string) FeedbackOption {
	return func(o *FeedbackOptions) {
		o.Category = category
	}
}

// WithFeedbackSource sets the source of the score.
func WithFeedbackSource(source string) FeedbackOption {
	return func(o *FeedbackOptions) {
		o.Source = source
	}
}

// ListOption configures list operations.
type ListOption func(*ListOptions)

// ListOptions holds list configuration.
type ListOptions struct {
	Limit   int
	Offset  int
	OrderBy string
	Filter  map[string]any
}

// WithLimit sets the maximum number of results.
func WithLimit(limit int) ListOption {
	return func(o *ListOptions) {
		o.Limit = limit
	}
}

// WithOffset sets the offset for pagination.
func WithOffset(offset int) ListOption {
	return func(o *ListOptions) {
		o.Offset = offset
	}
}

// WithOrderBy sets the ordering field.
func WithOrderBy(field string) ListOption {
	return func(o *ListOptions) {
		o.OrderBy = field
	}
}

// WithFilter sets filter criteria.
func WithFilter(filter map[string]any) ListOption {
	return func(o *ListOptions) {
		o.Filter = filter
	}
}

// PromptOption configures prompt creation.
type PromptOption func(*PromptOptions)

// PromptOptions holds prompt configuration.
type PromptOptions struct {
	Description   string
	Tags          []string
	Metadata      map[string]any
	ModelName     string // LLM model name (e.g., "gpt-4", "claude-3")
	ModelProvider string // LLM provider (e.g., "openai", "anthropic")
}

// WithPromptDescription sets the prompt description.
func WithPromptDescription(desc string) PromptOption {
	return func(o *PromptOptions) {
		o.Description = desc
	}
}

// WithPromptTags sets the prompt tags.
func WithPromptTags(tags ...string) PromptOption {
	return func(o *PromptOptions) {
		o.Tags = tags
	}
}

// WithPromptModel sets the LLM model for the prompt.
func WithPromptModel(model string) PromptOption {
	return func(o *PromptOptions) {
		o.ModelName = model
	}
}

// WithPromptProvider sets the LLM provider for the prompt.
func WithPromptProvider(provider string) PromptOption {
	return func(o *PromptOptions) {
		o.ModelProvider = provider
	}
}

// DatasetOption configures dataset creation.
type DatasetOption func(*DatasetOptions)

// DatasetOptions holds dataset configuration.
type DatasetOptions struct {
	Description string
	Tags        []string
	Metadata    map[string]any
}

// WithDatasetDescription sets the dataset description.
func WithDatasetDescription(desc string) DatasetOption {
	return func(o *DatasetOptions) {
		o.Description = desc
	}
}

// WithDatasetTags sets the dataset tags.
func WithDatasetTags(tags ...string) DatasetOption {
	return func(o *DatasetOptions) {
		o.Tags = tags
	}
}

// ProjectOption configures project creation.
type ProjectOption func(*ProjectOptions)

// ProjectOptions holds project configuration.
type ProjectOptions struct {
	Description string
	Metadata    map[string]any
}

// WithProjectDescription sets the project description.
func WithProjectDescription(desc string) ProjectOption {
	return func(o *ProjectOptions) {
		o.Description = desc
	}
}

// ClientOption configures client/provider creation.
type ClientOption func(*ClientOptions)

// ClientOptions holds client configuration.
type ClientOptions struct {
	APIKey      string //nolint:gosec // G117: APIKey is intentionally stored for provider authentication
	Endpoint    string
	Workspace   string
	ProjectName string
	HTTPClient  *http.Client
	Timeout     time.Duration
	Disabled    bool
	Debug       bool
	Logger      *slog.Logger // Optional: log trace events to slog
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) ClientOption {
	return func(o *ClientOptions) {
		o.APIKey = key
	}
}

// WithEndpoint sets the API endpoint URL.
func WithEndpoint(endpoint string) ClientOption {
	return func(o *ClientOptions) {
		o.Endpoint = endpoint
	}
}

// WithWorkspace sets the workspace name.
func WithWorkspace(workspace string) ClientOption {
	return func(o *ClientOptions) {
		o.Workspace = workspace
	}
}

// WithProjectName sets the default project name.
func WithProjectName(project string) ClientOption {
	return func(o *ClientOptions) {
		o.ProjectName = project
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(o *ClientOptions) {
		o.HTTPClient = client
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *ClientOptions) {
		o.Timeout = timeout
	}
}

// WithDisabled disables tracing (useful for testing).
func WithDisabled(disabled bool) ClientOption {
	return func(o *ClientOptions) {
		o.Disabled = disabled
	}
}

// WithDebug enables debug logging.
func WithDebug(debug bool) ClientOption {
	return func(o *ClientOptions) {
		o.Debug = debug
	}
}

// WithLogger sets an slog.Logger for local trace event logging.
// When set, trace and span events are logged to this logger in addition
// to being sent to the observability platform.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(o *ClientOptions) {
		o.Logger = logger
	}
}

// ApplyTraceOptions applies options to a TraceOptions struct.
func ApplyTraceOptions(opts ...TraceOption) *TraceOptions {
	o := &TraceOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// ApplySpanOptions applies options to a SpanOptions struct.
func ApplySpanOptions(opts ...SpanOption) *SpanOptions {
	o := &SpanOptions{
		Type: SpanTypeGeneral,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// ApplyClientOptions applies options to a ClientOptions struct.
func ApplyClientOptions(opts ...ClientOption) *ClientOptions {
	o := &ClientOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// ApplyListOptions applies options to a ListOptions struct.
func ApplyListOptions(opts ...ListOption) *ListOptions {
	o := &ListOptions{
		Limit: 100, // default limit
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
