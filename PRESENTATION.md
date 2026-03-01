---
marp: true
theme: vibeminds
paginate: true
style: |
  /* Two-column layout */
  .columns {
    display: flex;
    gap: 40px;
    align-items: flex-start;
  }

  .column-left {
    flex: 1;
  }

  .column-right {
    flex: 1;
  }

  /* Section divider slides */
  section.section-divider {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    text-align: center;
    background: linear-gradient(135deg, #1a1a3e 0%, #4a3f8a 50%, #2d2d5a 100%);
  }

  section.section-divider h1 {
    font-size: 3.5em;
    margin-bottom: 0.2em;
  }

  section.section-divider h2 {
    font-size: 1.5em;
    color: #b39ddb;
    font-weight: 400;
  }

  section.section-divider p {
    font-size: 1.1em;
    color: #9575cd;
    margin-top: 1em;
  }
---

<!-- _paginate: false -->

# OmniObserve

## Unified LLM & ML Observability for Go

A vendor-agnostic abstraction layer for AI application observability

---

# The Problem

## Observability Fragmentation

- Multiple LLM observability platforms exist (Langfuse, Opik, Phoenix, etc.)
- Each has its own SDK with different APIs
- Switching providers requires significant code changes
- Teams get locked into specific vendors
- No standardized way to instrument Go AI applications

---

# The Solution

## OmniObserve

**Instrument once, observe anywhere**

- Single unified API for all providers
- Seamless provider switching without code changes
- Clean Go idioms and patterns
- Full feature support: tracing, evaluation, prompts, datasets

```go
provider, _ := llmops.Open("opik", llmops.WithAPIKey("..."))
// Switch to Langfuse? Just change "opik" to "langfuse"
```

---

# Key Features

| Feature | Description |
|---------|-------------|
| **Unified Interface** | Single API across all providers |
| **Full Tracing** | Traces, spans, token usage, costs |
| **Evaluation** | Metrics, feedback scores |
| **Dataset Management** | Create and manage test datasets |
| **Prompt Versioning** | Store and version templates |
| **Context Propagation** | Automatic via `context.Context` |

---

# Supported Providers

| Provider | Description | Highlights |
|----------|-------------|------------|
| **Comet Opik** | Open-source, self-hosted | Full-featured, prompt versioning |
| **Langfuse** | Cloud & self-hosted | Batch ingestion, cost tracking |
| **Arize Phoenix** | OpenTelemetry-based | OTel integration, distributed tracing |

All providers support core tracing and evaluation capabilities.

---

# Architecture Overview

```
omniobserve/
├── omniobserve.go        # Main package, re-exports
├── llmops/               # LLM observability
│   ├── llmops.go         # Core interfaces
│   ├── trace.go          # Trace/Span interfaces
│   └── langfuse/         # Langfuse adapter
├── integrations/         # LLM library integrations
│   └── omnillm/          # OmniLLM hook
├── mlops/                # ML operations (future)
└── sdk/                  # Provider SDKs

# Provider adapters in standalone SDKs:
# go-opik/llmops, go-phoenix/llmops
```

---

# Core Interfaces

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

Composed of specialized interfaces for maximum flexibility.

---

# Tracer Interface

```go
type Tracer interface {
    // Start a new trace
    StartTrace(ctx context.Context, name string, opts ...TraceOption)
        (context.Context, Trace, error)

    // Start a span within current trace
    StartSpan(ctx context.Context, name string, opts ...SpanOption)
        (context.Context, Span, error)

    // Retrieve from context
    TraceFromContext(ctx context.Context) (Trace, bool)
    SpanFromContext(ctx context.Context) (Span, bool)
}
```

---

# Span Types

Categorize your operations for better observability:

```go
const (
    SpanTypeGeneral   // General operations
    SpanTypeLLM       // LLM API calls
    SpanTypeTool      // Tool/function calls
    SpanTypeRetrieval // Vector search, RAG retrieval
    SpanTypeAgent     // Agent orchestration
    SpanTypeChain     // Chain operations
    SpanTypeGuardrail // Safety checks
)
```

---

# Quick Start

```go
import (
    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"  // Register provider
)

func main() {
    // Open provider
    provider, _ := llmops.Open("opik",
        llmops.WithAPIKey("your-api-key"),
        llmops.WithProjectName("my-project"),
    )
    defer provider.Close()

    // Start tracing...
}
```

---

# Basic Tracing Example

```go
ctx := context.Background()

// Start a trace
ctx, trace, _ := provider.StartTrace(ctx, "chat-workflow")
defer trace.End()

// Start a span for the LLM call
ctx, span, _ := provider.StartSpan(ctx, "gpt-4-call",
    llmops.WithSpanType(llmops.SpanTypeLLM),
    llmops.WithModel("gpt-4"),
)

span.SetInput(messages)
// ... call LLM ...
span.SetOutput(response)
span.SetUsage(llmops.TokenUsage{TotalTokens: 150})
span.End()
```

---

# RAG Pipeline Example

```go
ctx, trace, _ := provider.StartTrace(ctx, "rag-pipeline")

// Retrieval span
ctx, retrieval, _ := provider.StartSpan(ctx, "vector-search",
    llmops.WithSpanType(llmops.SpanTypeRetrieval))
retrieval.SetOutput(documents)
retrieval.End()

// Generation span
ctx, generation, _ := provider.StartSpan(ctx, "generate",
    llmops.WithSpanType(llmops.SpanTypeLLM),
    llmops.WithModel("gpt-4"))
generation.SetOutput(response)
generation.End()

trace.End()
```

---

# Adding Feedback Scores

```go
// Add quality score to a span
span.AddFeedbackScore(ctx, "relevance", 0.95,
    llmops.WithFeedbackReason("Directly addressed the query"),
    llmops.WithFeedbackCategory("quality"),
)

// Add user satisfaction to a trace
trace.AddFeedbackScore(ctx, "user_satisfaction", 0.8)

// Programmatic evaluation
provider.AddFeedbackScore(ctx, llmops.FeedbackScoreOpts{
    TraceID: trace.ID(),
    Name:    "accuracy",
    Score:   0.92,
})
```

---

# Dataset Management

```go
// Create a dataset
dataset, _ := provider.CreateDataset(ctx, "qa-test-cases",
    llmops.WithDatasetDescription("Q&A evaluation dataset"),
)

// Add test items
provider.AddDatasetItems(ctx, "qa-test-cases", []llmops.DatasetItem{
    {
        Input:    map[string]any{"query": "What is Go?"},
        Expected: map[string]any{"answer": "A programming language..."},
    },
    {
        Input:    map[string]any{"query": "Who created Go?"},
        Expected: map[string]any{"answer": "Google engineers..."},
    },
})
```

---

# Prompt Management

```go
// Create a versioned prompt template
prompt, _ := provider.CreatePrompt(ctx, "chat-system",
    `You are {{.role}}. Answer questions about {{.topic}}.`,
    llmops.WithPromptDescription("Main chat template"),
)

// Retrieve latest version
prompt, _ := provider.GetPrompt(ctx, "chat-system")

// Render with variables
rendered := prompt.Render(map[string]any{
    "role":  "a helpful assistant",
    "topic": "programming",
})
// "You are a helpful assistant. Answer questions about programming."
```

---

# Switching Providers

```go
// Just change the import and provider name!

// Opik
import _ "github.com/agentplexus/go-opik/llmops"
provider, _ := llmops.Open("opik", llmops.WithAPIKey("..."))

// Langfuse
import _ "github.com/plexusone/omniobserve/llmops/langfuse"
provider, _ := llmops.Open("langfuse", llmops.WithAPIKey("..."))

// Phoenix
import _ "github.com/agentplexus/go-phoenix/llmops"
provider, _ := llmops.Open("phoenix", llmops.WithEndpoint("..."))
```

**Your tracing code stays exactly the same!**

---

# OmniLLM Integration

Automatically instrument LLM calls via OmniLLM:

```go
import (
    "github.com/plexusone/omnillm"
    omnillmhook "github.com/plexusone/omniobserve/integrations/omnillm"
)

// Create hook with any OmniObserve provider
hook := omnillmhook.NewHook(provider)

// Attach to OmniLLM client
client := omnillm.NewClient(
    omnillm.WithObservabilityHook(hook),
)
// All LLM calls are now automatically traced!
```

Captures: model, provider, input/output, tokens, streaming, errors

---

# Configuration Options

## Client Options
```go
llmops.WithAPIKey("...")
llmops.WithEndpoint("https://...")
llmops.WithProjectName("my-project")
llmops.WithTimeout(30 * time.Second)
llmops.WithDebug(true)
```

## Span Options
```go
llmops.WithSpanType(llmops.SpanTypeLLM)
llmops.WithModel("gpt-4")
llmops.WithProvider("openai")
llmops.WithTokenUsage(usage)
```

---

# Provider Capabilities Matrix

| Feature | Opik | Langfuse | Phoenix |
|---------|:----:|:--------:|:-------:|
| Tracing | ✅ Yes | ✅ Yes | ✅ Yes |
| Evaluation | ✅ Yes | ✅ Yes | ✅ Yes |
| Prompts | ✅ Yes | 🟡 Partial | ❌ No |
| Datasets | ✅ Yes | ✅ Yes | 🟡 Partial |
| Cost Tracking | ✅ Yes | ✅ Yes | ❌ No |
| OpenTelemetry | ❌ No | ❌ No | ✅ Yes |

---

# Design Patterns Used

- **Adapter Pattern**: Provider implementations wrap native SDKs
- **Factory Pattern**: Dynamic provider registration and instantiation
- **Functional Options**: Clean, extensible configuration
- **Composition**: Provider interface composes specialized interfaces
- **Context Propagation**: Trace/span context via `context.Context`

---

# Error Handling

```go
// Sentinel errors
if errors.Is(err, llmops.ErrMissingAPIKey) {
    log.Fatal("API key required")
}

// Error classification
if llmops.IsNotFound(err) {
    // Handle 404
}
if llmops.IsRateLimited(err) {
    // Implement backoff
}
if llmops.IsNotImplemented(err) {
    // Feature not available for this provider
}
```

---

# Roadmap

## Current (v0.1.0)
- LLM observability with 3 providers
- Full tracing, evaluation, datasets
- OmniLLM integration for automatic instrumentation

## Planned
- MLOps interfaces (experiments, model registry)
- Additional providers (Lunary, MLflow, W&B)
- OTLP export support
- Async batch processing

---

# Getting Started

## Install

```bash
go get github.com/plexusone/omniobserve
go get github.com/agentplexus/go-opik  # For Opik provider
```

## Import

```go
import (
    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"
)
```

## Documentation

- GitHub: `github.com/plexusone/omniobserve`
- Go Docs: `pkg.go.dev/github.com/plexusone/omniobserve`

---

# Summary

## OmniObserve provides:

1. **Unified API** for LLM observability
2. **Provider flexibility** without vendor lock-in
3. **Clean Go patterns** (options, interfaces, context)
4. **Comprehensive features** (tracing, evaluation, datasets, prompts)
5. **Easy adoption** with minimal code changes

**Instrument once. Observe anywhere.**

---

<!-- _class: section-divider -->

# Thank You

## Questions?

GitHub: `github.com/plexusone/omniobserve`

```go
provider, _ := llmops.Open("your-choice", ...)
```
