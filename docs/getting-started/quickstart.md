# Quick Start

## Basic Tracing

```go
package main

import (
    "context"
    "log"

    "github.com/agentplexus/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"  // Register Opik provider
)

func main() {
    // Open a provider
    provider, err := llmops.Open("opik",
        llmops.WithAPIKey("your-api-key"),
        llmops.WithProjectName("my-project"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    ctx := context.Background()

    // Start a trace
    ctx, trace, err := provider.StartTrace(ctx, "chat-workflow",
        llmops.WithTraceInput(map[string]any{"query": "Hello, world!"}),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer trace.End()

    // Start a span for the LLM call
    ctx, span, err := provider.StartSpan(ctx, "gpt-4-completion",
        llmops.WithSpanType(llmops.SpanTypeLLM),
        llmops.WithModel("gpt-4"),
        llmops.WithProvider("openai"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Record the LLM interaction
    span.SetInput(map[string]any{
        "messages": []map[string]string{
            {"role": "user", "content": "Hello!"},
        },
    })

    // ... call your LLM here ...

    span.SetOutput(map[string]any{
        "response": "Hello! How can I help you today?",
    })
    span.SetUsage(llmops.TokenUsage{
        PromptTokens:     10,
        CompletionTokens: 8,
        TotalTokens:      18,
    })

    span.End()
    trace.SetOutput(map[string]any{"response": "Hello! How can I help you today?"})
}
```

## Nested Spans

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

## Adding Feedback Scores

```go
// Add a score to a span
span.AddFeedbackScore(ctx, "relevance", 0.95,
    llmops.WithFeedbackReason("Response directly addressed the query"),
    llmops.WithFeedbackCategory("quality"),
)

// Add a score to a trace
trace.AddFeedbackScore(ctx, "user_satisfaction", 0.8)
```
