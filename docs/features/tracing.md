# Tracing

OmniObserve provides a unified tracing interface across all providers.

## Core Concepts

- **Trace**: A top-level unit representing an end-to-end operation (e.g., a user request)
- **Span**: A unit of work within a trace (e.g., an LLM call, retrieval operation)

## Trace Interface

```go
type Trace interface {
    ID() string
    Name() string
    StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span, error)
    SetInput(input any) error
    SetOutput(output any) error
    SetMetadata(metadata map[string]any) error
    AddTag(tag string) error
    AddFeedbackScore(ctx context.Context, name string, score float64, opts ...FeedbackOption) error
    End(opts ...EndOption) error
}
```

## Span Interface

```go
type Span interface {
    ID() string
    TraceID() string
    ParentSpanID() string
    Name() string
    Type() SpanType
    StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span, error)
    SetInput(input any) error
    SetOutput(output any) error
    SetModel(model string) error
    SetProvider(provider string) error
    SetUsage(usage TokenUsage) error
    SetMetadata(metadata map[string]any) error
    AddTag(tag string) error
    AddFeedbackScore(ctx context.Context, name string, score float64, opts ...FeedbackOption) error
    End(opts ...EndOption) error
}
```

## Span Types

```go
const (
    SpanTypeGeneral   SpanType = "general"   // Default type
    SpanTypeLLM       SpanType = "llm"       // LLM call
    SpanTypeTool      SpanType = "tool"      // Tool/function execution
    SpanTypeRetrieval SpanType = "retrieval" // Vector search, retrieval
    SpanTypeAgent     SpanType = "agent"     // Agent operation
    SpanTypeChain     SpanType = "chain"     // Chain of operations
    SpanTypeGuardrail SpanType = "guardrail" // Safety check
)
```

## Starting Traces and Spans

```go
// Start a trace
ctx, trace, err := provider.StartTrace(ctx, "chat-workflow",
    llmops.WithTraceInput(map[string]any{"query": "Hello"}),
)
defer trace.End()

// Start a span from provider (uses trace from context)
ctx, span, err := provider.StartSpan(ctx, "llm-call",
    llmops.WithSpanType(llmops.SpanTypeLLM),
    llmops.WithModel("gpt-4"),
)
defer span.End()

// Start a nested span from existing span
ctx, childSpan, err := span.StartSpan(ctx, "embedding",
    llmops.WithSpanType(llmops.SpanTypeRetrieval),
)
defer childSpan.End()
```

## Recording Token Usage

```go
span.SetUsage(llmops.TokenUsage{
    PromptTokens:     150,
    CompletionTokens: 50,
    TotalTokens:      200,
})
```

## Context Propagation

Traces and spans are propagated via `context.Context`:

```go
// Trace is stored in context
ctx, trace, _ := provider.StartTrace(ctx, "workflow")

// Subsequent spans automatically link to the trace
ctx, span, _ := provider.StartSpan(ctx, "step-1")

// Get trace from context
trace, ok := provider.TraceFromContext(ctx)

// Get span from context
span, ok := provider.SpanFromContext(ctx)
```

## Ending with Errors

```go
if err != nil {
    span.End(llmops.WithError(err))
    return err
}
span.End()
```
