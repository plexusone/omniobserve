# OmniLLM Integration

OmniObserve provides an integration with [OmniLLM](https://github.com/plexusone/omnillm), a multi-LLM abstraction layer. This allows you to automatically instrument all LLM calls made through OmniLLM with any OmniObserve provider.

## Installation

```bash
go get github.com/plexusone/omniobserve/integrations/omnillm
```

## Basic Usage

```go
package main

import (
    "github.com/plexusone/omnillm"
    omnillmhook "github.com/plexusone/omniobserve/integrations/omnillm"
    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"
)

func main() {
    // Initialize an OmniObserve provider
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

## Automatic Capture

The hook automatically captures:

- **Model and provider information** - Which LLM backend was used
- **Input messages** - The prompt and conversation history
- **Output responses** - The generated completion
- **Token usage** - Prompt, completion, and total tokens
- **Streaming responses** - Full streaming support with incremental capture
- **Errors** - Any errors that occur during generation

## Auto-Create Traces

The hook automatically creates traces when none exists in context, ensuring all LLM calls are properly traced even when called outside an existing trace context.

```go
// This works even without an existing trace
response, err := client.Complete(ctx, omnillm.Request{
    Messages: []omnillm.Message{
        {Role: "user", Content: "Hello!"},
    },
})
// A trace is automatically created and ended
```

## With Existing Traces

When a trace already exists in context, the hook creates spans under that trace:

```go
// Start a trace
ctx, trace, _ := provider.StartTrace(ctx, "chat-workflow")
defer trace.End()

// LLM calls create spans under the trace
response, err := client.Complete(ctx, omnillm.Request{
    Messages: []omnillm.Message{
        {Role: "user", Content: "Hello!"},
    },
})
```

## Hook Lifecycle

The hook implements two methods:

- **BeforeRequest** - Called before each LLM request
  - Creates a trace if none exists in context
  - Creates a span for the LLM call
  - Records input messages and model information

- **AfterResponse** - Called after each LLM response
  - Records output and token usage
  - Ends the span
  - Ends the trace (if created by the hook)

## Streaming Support

The hook properly handles streaming responses:

```go
stream, err := client.CompleteStream(ctx, omnillm.Request{
    Messages: []omnillm.Message{
        {Role: "user", Content: "Tell me a story"},
    },
})

for chunk := range stream {
    // Process chunk
}
// Trace/span are automatically ended when stream completes
```

## Context Helpers

The integration provides context helpers for manual trace management:

```go
import omnillmhook "github.com/plexusone/omniobserve/integrations/omnillm"

// Store a trace in context
ctx = omnillmhook.ContextWithTrace(ctx, trace)

// Retrieve a trace from context
trace, ok := omnillmhook.TraceFromContext(ctx)
```
