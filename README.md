# OmniObserve

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omniobserve/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omniobserve/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omniobserve/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omniobserve/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omniobserve/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omniobserve/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omniobserve
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omniobserve
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omniobserve
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omniobserve
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomniobserve
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omniobserve
 [repo-url]: https://github.com/plexusone/omniobserve
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omniobserve/blob/master/LICENSE

A unified Go library for observability. OmniObserve provides vendor-agnostic abstraction layers that enable you to instrument your applications once and seamlessly switch between different observability backends without code changes.

## Two Provider Systems

| Package | Purpose | Providers |
|---------|---------|-----------|
| **llmops** | LLM/ML observability | Opik, Langfuse, Phoenix, slog |
| **observops** | App observability (metrics, traces, logs) | OTLP, Datadog, New Relic, Dynatrace |

## Features

### LLM Observability (llmops)

- 🔗 **Unified Interface**: Single API for tracing, evaluation, prompts, and datasets across all providers
- 🔄 **Provider Agnostic**: Switch between Opik, Langfuse, and Phoenix without changing your code
- 🔍 **Full Tracing**: Trace LLM calls with spans, token usage, and cost tracking
- 📊 **Evaluation Support**: Run metrics and add feedback scores to traces
- 📦 **Dataset Management**: Create and manage evaluation datasets
- 📝 **Prompt Versioning**: Store and version prompt templates (provider-dependent)

### App Observability (observops)

- 📈 **Vendor-Agnostic**: Single API for OTLP, Datadog, New Relic, and Dynatrace
- 📊 **Full Telemetry**: Metrics (counters, gauges, histograms), distributed traces, and structured logs
- 📝 **slog Integration**: Trace-correlated logging with automatic context injection
- ⚡ **Minimal Overhead**: No-op mode for disabled observability

### Common

- 🔀 **Context Propagation**: Automatic trace/span context propagation via `context.Context`
- ⚙️ **Functional Options**: Clean, extensible configuration using the options pattern

## Installation

```bash
go get github.com/plexusone/omniobserve
```

## Quick Start

### LLM Observability (llmops)

```go
package main

import (
    "context"
    "log"

    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"  // Register Opik provider
)

func main() {
    provider, err := llmops.Open("opik",
        llmops.WithAPIKey("your-api-key"),
        llmops.WithProjectName("my-project"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    ctx := context.Background()

    ctx, trace, _ := provider.StartTrace(ctx, "chat-workflow",
        llmops.WithTraceInput(map[string]any{"query": "Hello, world!"}),
    )
    defer trace.End()

    _, span, _ := provider.StartSpan(ctx, "gpt-4-completion",
        llmops.WithSpanType(llmops.SpanTypeLLM),
        llmops.WithModel("gpt-4"),
    )

    span.SetUsage(llmops.TokenUsage{TotalTokens: 18})
    span.End()
}
```

### App Observability (observops)

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "os"

    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/otlp"  // or datadog, newrelic, dynatrace
)

func main() {
    provider, err := observops.Open("otlp",
        observops.WithEndpoint("localhost:4317"),
        observops.WithServiceName("my-service"),
        observops.WithInsecure(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(context.Background())

    ctx := context.Background()

    // Metrics
    counter, _ := provider.Meter().Counter("requests_total",
        observops.WithDescription("Total HTTP requests"),
    )
    counter.Add(ctx, 1, observops.WithAttributes(
        observops.Attribute("method", "GET"),
        observops.Attribute("path", "/api/users"),
    ))

    // Tracing
    ctx, span := provider.Tracer().Start(ctx, "handle-request")
    defer span.End()

    // slog with trace correlation
    handler := provider.SlogHandler(
        observops.WithSlogLocalHandler(slog.NewJSONHandler(os.Stdout, nil)),
    )
    slog.SetDefault(slog.New(handler))

    slog.InfoContext(ctx, "request processed")  // includes trace_id, span_id
}
```

## Supported Providers

### LLM Providers (llmops)

| Provider | Package | Description |
|----------|---------|-------------|
| **Opik** | `go-opik/llmops` | Comet Opik - Open-source, full-featured |
| **Langfuse** | `omniobserve/llmops/langfuse` | Cloud & self-hosted, batch ingestion |
| **Phoenix** | `go-phoenix/llmops` | Arize Phoenix - OpenTelemetry-based |
| **slog** | `omniobserve/llmops/slog` | Local structured logging for development/debugging |

### App Observability Providers (observops)

| Provider | Package | Description |
|----------|---------|-------------|
| **OTLP** | `omniobserve/observops/otlp` | OpenTelemetry Protocol - vendor-agnostic |
| **Datadog** | `omniobserve/observops/datadog` | Datadog APM via OTLP |
| **New Relic** | `omniobserve/observops/newrelic` | New Relic via OTLP |
| **Dynatrace** | `omniobserve/observops/dynatrace` | Dynatrace via OTLP |

### LLM Provider Capabilities

| Feature | Opik | Langfuse | Phoenix | slog |
|---------|:----:|:--------:|:-------:|:----:|
| Tracing | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| Evaluation | :white_check_mark: | :white_check_mark: | :white_check_mark: | :x: |
| Prompts | :white_check_mark: | Partial | :x: | :x: |
| Datasets | :white_check_mark: | :white_check_mark: | Partial | :x: |
| Experiments | :white_check_mark: | :white_check_mark: | Partial | :x: |
| Streaming | :white_check_mark: | :white_check_mark: | Planned | :x: |
| Distributed Tracing | :white_check_mark: | :x: | :white_check_mark: | :x: |
| Cost Tracking | :white_check_mark: | :white_check_mark: | :x: | :x: |
| OpenTelemetry | :x: | :x: | :white_check_mark: | :x: |

### observops Capabilities

| Feature | OTLP | Datadog | New Relic | Dynatrace |
|---------|:----:|:-------:|:---------:|:---------:|
| Metrics | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| Traces | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| Logs | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| slog Handler | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |

## Architecture

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
│   └── langfuse/        # Langfuse provider adapter
├── observops/           # App observability interfaces
│   ├── observops.go     # Core interfaces (Provider, Meter, Tracer)
│   ├── options.go       # Functional options
│   ├── otlp/            # OTLP provider (vendor-agnostic)
│   ├── datadog/         # Datadog provider
│   ├── newrelic/        # New Relic provider
│   └── dynatrace/       # Dynatrace provider
├── sloghandler/         # slog.Handler implementations
│   ├── dual.go          # Local + remote handler
│   ├── fanout.go        # Multi-handler fanout
│   └── trace.go         # Trace context injection
├── integrations/        # Integrations with LLM libraries
│   └── omnillm/         # OmniLLM observability hook (separate module)
├── examples/            # Usage examples
│   └── evaluation/      # Metrics evaluation example
├── mlops/               # ML operations interfaces (experiments, model registry)
└── sdk/                 # Provider-specific SDKs
    └── langfuse/        # Langfuse Go SDK

# LLM provider adapters in standalone SDKs:
# github.com/agentplexus/go-opik/llmops      # Opik provider
# github.com/agentplexus/go-phoenix/llmops   # Phoenix provider
```

## Core Interfaces

### Provider

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

### Trace and Span

```go
type Trace interface {
    ID() string
    Name() string
    StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span, error)
    SetInput(input any)
    SetOutput(output any)
    SetMetadata(metadata map[string]any)
    AddTag(key, value string)
    AddFeedbackScore(ctx context.Context, name string, score float64, opts ...FeedbackOption) error
    End(opts ...EndOption)
}

type Span interface {
    ID() string
    TraceID() string
    Name() string
    Type() SpanType
    StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span, error)
    SetInput(input any)
    SetOutput(output any)
    SetModel(model string)
    SetProvider(provider string)
    SetUsage(usage TokenUsage)
    End(opts ...EndOption)
}
```

### Span Types

```go
const (
    SpanTypeGeneral   SpanType = "general"
    SpanTypeLLM       SpanType = "llm"
    SpanTypeTool      SpanType = "tool"
    SpanTypeRetrieval SpanType = "retrieval"
    SpanTypeAgent     SpanType = "agent"
    SpanTypeChain     SpanType = "chain"
    SpanTypeGuardrail SpanType = "guardrail"
)
```

## Usage Examples

### Using Different Providers

```go
// Opik
import _ "github.com/agentplexus/go-opik/llmops"
provider, _ := llmops.Open("opik", llmops.WithAPIKey("..."))

// Langfuse
import _ "github.com/plexusone/omniobserve/llmops/langfuse"
provider, _ := llmops.Open("langfuse",
    llmops.WithAPIKey("sk-lf-..."),
    llmops.WithEndpoint("https://cloud.langfuse.com"),
)

// Phoenix
import _ "github.com/agentplexus/go-phoenix/llmops"
provider, _ := llmops.Open("phoenix",
    llmops.WithEndpoint("http://localhost:6006"),
)
```

### Nested Spans

```go
ctx, trace, _ := provider.StartTrace(ctx, "rag-pipeline")
defer trace.End()

// Retrieval span
ctx, retrievalSpan, _ := provider.StartSpan(ctx, "vector-search",
    llmops.WithSpanType(llmops.SpanTypeRetrieval),
)
// ... perform retrieval ...
retrievalSpan.SetOutput(documents)
retrievalSpan.End()

// LLM span
ctx, llmSpan, _ := provider.StartSpan(ctx, "generate-response",
    llmops.WithSpanType(llmops.SpanTypeLLM),
    llmops.WithModel("gpt-4"),
)
// ... call LLM ...
llmSpan.SetUsage(llmops.TokenUsage{
    PromptTokens:     150,
    CompletionTokens: 50,
    TotalTokens:      200,
})
llmSpan.End()
```

### Adding Feedback Scores

```go
// Add a score to a span
span.AddFeedbackScore(ctx, "relevance", 0.95,
    llmops.WithFeedbackReason("Response directly addressed the query"),
    llmops.WithFeedbackCategory("quality"),
)

// Add a score to a trace
trace.AddFeedbackScore(ctx, "user_satisfaction", 0.8)
```

### Working with Datasets

```go
// Create a dataset
dataset, _ := provider.CreateDataset(ctx, "test-cases",
    llmops.WithDatasetDescription("Test cases for RAG evaluation"),
)

// Add items
provider.AddDatasetItems(ctx, "test-cases", []llmops.DatasetItem{
    {
        Input:    map[string]any{"query": "What is Go?"},
        Expected: map[string]any{"answer": "Go is a programming language..."},
    },
})
```

### Working with Prompts (Opik)

```go
// Create a versioned prompt
prompt, _ := provider.CreatePrompt(ctx, "chat-template",
    `You are a helpful assistant. User: {{.query}}`,
    llmops.WithPromptDescription("Main chat template"),
)

// Get a prompt
prompt, _ := provider.GetPrompt(ctx, "chat-template")

// Render with variables
rendered := prompt.Render(map[string]any{"query": "Hello!"})
```

## Configuration Options

### Client Options

```go
llmops.WithAPIKey("...")           // API key for authentication
llmops.WithEndpoint("...")         // Custom endpoint URL
llmops.WithWorkspace("...")        // Workspace/organization name
llmops.WithProjectName("...")      // Default project name
llmops.WithHTTPClient(client)      // Custom HTTP client
llmops.WithTimeout(30 * time.Second)
llmops.WithDisabled(true)          // Disable tracing (no-op mode)
llmops.WithDebug(true)             // Enable debug logging
```

### Trace Options

```go
llmops.WithTraceProject("...")
llmops.WithTraceInput(input)
llmops.WithTraceOutput(output)
llmops.WithTraceMetadata(map[string]any{...})
llmops.WithTraceTags(map[string]string{...})
llmops.WithThreadID("...")
```

### Span Options

```go
llmops.WithSpanType(llmops.SpanTypeLLM)
llmops.WithSpanInput(input)
llmops.WithSpanOutput(output)
llmops.WithSpanMetadata(map[string]any{...})
llmops.WithModel("gpt-4")
llmops.WithProvider("openai")
llmops.WithTokenUsage(usage)
llmops.WithParentSpan(parentSpan)
```

## Error Handling

The library provides typed errors for common conditions:

```go
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

For provider-specific features, you can use the underlying SDKs directly:

```go
import "github.com/plexusone/omniobserve/sdk/langfuse"  // Langfuse SDK
import "github.com/agentplexus/go-opik"                   // Opik SDK
import "github.com/agentplexus/go-phoenix"                // Phoenix SDK
```

## OmniLLM Integration

OmniObserve provides an integration with [OmniLLM](https://github.com/plexusone/omnillm), a multi-LLM abstraction layer. This allows you to automatically instrument all LLM calls made through OmniLLM with any OmniObserve provider.

```bash
go get github.com/plexusone/omniobserve/integrations/omnillm
```

```go
package main

import (
    "github.com/plexusone/omnillm"
    omnillmhook "github.com/plexusone/omniobserve/integrations/omnillm"
    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"
)

func main() {
    // Initialize a OmniObserve provider
    provider, _ := llmops.Open("opik",
        llmops.WithAPIKey("your-api-key"),
        llmops.WithProjectName("my-project"),
    )
    defer provider.Close()

    // Create the observability hook
    hook := omnillmhook.NewHook(provider)

    // Attach to your OmniLLM client
    client := omnillm.NewClient(
        omnillm.WithObservabilityHook(hook),
    )

    // All LLM calls through this client are now automatically traced
}
```

The hook automatically captures:

- Model and provider information
- Input messages and output responses
- Token usage (prompt, completion, total)
- Streaming responses
- Errors

The hook also automatically creates traces when none exists in context, ensuring all LLM calls are properly traced.

## Requirements

- Go 1.24.5 or later

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

See [LICENSE](LICENSE) for details.
