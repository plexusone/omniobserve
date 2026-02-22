# OmniObserve

**Unified Go library for LLM and ML observability**

[![Build Status](https://github.com/agentplexus/omniobserve/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/agentplexus/omniobserve/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agentplexus/omniobserve)](https://goreportcard.com/report/github.com/agentplexus/omniobserve)
[![Go Reference](https://pkg.go.dev/badge/github.com/agentplexus/omniobserve.svg)](https://pkg.go.dev/github.com/agentplexus/omniobserve)

OmniObserve provides a vendor-agnostic abstraction layer that enables you to instrument your AI applications once and seamlessly switch between different observability backends without code changes.

## Features

- **Unified Interface**: Single API for tracing, evaluation, prompts, and datasets across all providers
- **Provider Agnostic**: Switch between Opik, Langfuse, Phoenix, and slog without changing your code
- **Full Tracing**: Trace LLM calls with spans, token usage, and cost tracking
- **Evaluation Support**: Run metrics and add feedback scores to traces
- **Dataset Management**: Create and manage evaluation datasets
- **Prompt Versioning**: Store and version prompt templates (provider-dependent)
- **Context Propagation**: Automatic trace/span context propagation via `context.Context`
- **Functional Options**: Clean, extensible configuration using the options pattern

## Quick Example

```go
package main

import (
    "context"
    "log"

    "github.com/agentplexus/omniobserve/llmops"
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

    // Start a trace
    ctx, trace, _ := provider.StartTrace(ctx, "chat-workflow")
    defer trace.End()

    // Start a span for the LLM call
    _, span, _ := provider.StartSpan(ctx, "gpt-4-completion",
        llmops.WithSpanType(llmops.SpanTypeLLM),
        llmops.WithModel("gpt-4"),
    )

    span.SetUsage(llmops.TokenUsage{TotalTokens: 18})
    span.End()
}
```

## Supported Providers

| Provider | Package | Description |
|----------|---------|-------------|
| [Opik](providers/opik.md) | `go-opik/llmops` | Comet Opik - Open-source, full-featured |
| [Langfuse](providers/langfuse.md) | `omniobserve/llmops/langfuse` | Cloud & self-hosted, batch ingestion |
| [Phoenix](providers/phoenix.md) | `go-phoenix/llmops` | Arize Phoenix - OpenTelemetry-based |
| [slog](providers/slog.md) | `omniobserve/llmops/slog` | Local structured logging for development |

## Next Steps

- [Installation](getting-started/installation.md) - Get OmniObserve set up
- [Quick Start](getting-started/quickstart.md) - Your first trace
- [Providers](providers/index.md) - Configure specific providers
- [OmniLLM Integration](integrations/omnillm.md) - Auto-instrument LLM calls
