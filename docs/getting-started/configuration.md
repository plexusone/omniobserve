# Configuration

## Client Options

```go
llmops.WithAPIKey("...")           // API key for authentication
llmops.WithEndpoint("...")         // Custom endpoint URL
llmops.WithWorkspace("...")        // Workspace/organization name
llmops.WithProjectName("...")      // Default project name
llmops.WithHTTPClient(client)      // Custom HTTP client
llmops.WithTimeout(30 * time.Second)
llmops.WithDisabled(true)          // Disable tracing (no-op mode)
llmops.WithDebug(true)             // Enable debug logging
llmops.WithLogger(logger)          // Custom slog.Logger
```

## Trace Options

```go
llmops.WithTraceProject("...")
llmops.WithTraceInput(input)
llmops.WithTraceOutput(output)
llmops.WithTraceMetadata(map[string]any{...})
llmops.WithTraceTags(map[string]string{...})
llmops.WithThreadID("...")
```

## Span Options

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

## Span Types

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

## Error Handling

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

## Disabled Mode

For testing or when you want to disable observability:

```go
provider, _ := llmops.Open("opik",
    llmops.WithDisabled(true),
)
// All operations become no-ops
```
