# Architecture

OmniObserve follows a modular architecture with clear separation between interfaces and implementations.

## Package Structure

```
omniobserve/
├── omniobserve.go       # Main package with re-exports
├── llmops/              # LLM observability interfaces
│   ├── llmops.go        # Core interfaces (Provider, Tracer, Evaluator, etc.)
│   ├── trace.go         # Trace and Span interfaces
│   ├── types.go         # Data types (EvalInput, Dataset, Prompt, etc.)
│   ├── options.go       # Functional options
│   ├── provider.go      # Provider registration system
│   ├── errors.go        # Error definitions
│   ├── metrics/         # Evaluation metrics (hallucination, relevance, etc.)
│   ├── langfuse/        # Langfuse provider adapter
│   └── slog/            # slog provider adapter
├── integrations/        # Integrations with LLM libraries
│   ├── omnillm/         # OmniLLM observability hook
│   └── sevaluation/     # Structured evaluation integration
├── examples/            # Usage examples
│   └── evaluation/      # Metrics evaluation example
├── mlops/               # ML operations interfaces (experiments, model registry)
├── agentops/            # Agent operations monitoring
├── observops/           # Unified observability operations
├── semconv/             # Semantic conventions for agentic AI
└── sdk/                 # Provider-specific SDKs
    └── langfuse/        # Langfuse Go SDK
```

## Provider Adapters

Provider adapters are distributed in two ways:

### Embedded Adapters

Adapters included in omniobserve:

- `llmops/langfuse` - Langfuse adapter
- `llmops/slog` - slog adapter for local development

### Standalone SDK Adapters

Adapters in their own repositories:

- `github.com/agentplexus/go-opik/llmops` - Opik adapter
- `github.com/agentplexus/go-phoenix/llmops` - Phoenix adapter

## Core Interfaces

### Provider Interface

The main interface that all observability backends implement:

```go
type Provider interface {
    Tracer            // Trace/span operations
    Evaluator         // Evaluation and feedback
    PromptManager     // Prompt template management
    DatasetManager    // Test dataset management
    ProjectManager    // Project/workspace management
    AnnotationManager // Span/trace annotations
    io.Closer

    Name() string
}
```

### Tracer Interface

```go
type Tracer interface {
    StartTrace(ctx context.Context, name string, opts ...TraceOption) (context.Context, Trace, error)
    StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span, error)
    TraceFromContext(ctx context.Context) (Trace, bool)
    SpanFromContext(ctx context.Context) (Span, bool)
    Flush(ctx context.Context) error
}
```

### Evaluator Interface

```go
type Evaluator interface {
    AddFeedbackScore(ctx context.Context, traceID, spanID, name string, score float64, opts ...FeedbackOption) error
}
```

## Provider Registration

Providers register themselves using the `database/sql` style pattern:

```go
// In provider package init()
func init() {
    llmops.Register("opik", &opikDriver{})
}

// In application code
import _ "github.com/agentplexus/go-opik/llmops"

provider, _ := llmops.Open("opik", llmops.WithAPIKey("..."))
```

## Context Propagation

Traces and spans are propagated via `context.Context`:

```go
// Trace stored in context
ctx, trace, _ := provider.StartTrace(ctx, "workflow")

// Spans automatically link to trace from context
ctx, span, _ := provider.StartSpan(ctx, "step")

// Nested spans link to parent span
ctx, childSpan, _ := provider.StartSpan(ctx, "nested")
```

## Functional Options

Configuration uses the functional options pattern:

```go
// Client options
provider, _ := llmops.Open("opik",
    llmops.WithAPIKey("..."),
    llmops.WithProjectName("my-project"),
    llmops.WithTimeout(30 * time.Second),
)

// Trace options
ctx, trace, _ := provider.StartTrace(ctx, "workflow",
    llmops.WithTraceInput(input),
    llmops.WithTraceMetadata(metadata),
)

// Span options
ctx, span, _ := provider.StartSpan(ctx, "llm-call",
    llmops.WithSpanType(llmops.SpanTypeLLM),
    llmops.WithModel("gpt-4"),
)
```

## Error Handling

The library provides typed errors for common conditions:

```go
var (
    ErrMissingAPIKey  = errors.New("missing API key")
    ErrProviderNotFound = errors.New("provider not found")
    ErrTraceNotFound  = errors.New("trace not found")
)

// Error checking
if errors.Is(err, llmops.ErrMissingAPIKey) {
    // Handle missing API key
}

if llmops.IsNotFound(err) {
    // Handle not found
}

if llmops.IsRateLimited(err) {
    // Handle rate limiting
}
```

## Direct SDK Access

For provider-specific features, use the underlying SDKs directly:

```go
import "github.com/plexusone/omniobserve/sdk/langfuse"  // Langfuse SDK
import "github.com/agentplexus/go-opik"                   // Opik SDK
import "github.com/agentplexus/go-phoenix"                // Phoenix SDK
```
