# OmniObserve

**Unified Go library for observability**

[![Build Status](https://github.com/plexusone/omniobserve/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/plexusone/omniobserve/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/plexusone/omniobserve)](https://goreportcard.com/report/github.com/plexusone/omniobserve)
[![Go Reference](https://pkg.go.dev/badge/github.com/plexusone/omniobserve.svg)](https://pkg.go.dev/github.com/plexusone/omniobserve)

OmniObserve provides vendor-agnostic abstraction layers for observability, enabling you to instrument your applications once and seamlessly switch between different backends without code changes.

## Two Provider Systems

| Package | Purpose | Providers |
|---------|---------|-----------|
| **llmops** | LLM/ML observability | Opik, Langfuse, Phoenix, slog |
| **observops** | App observability (metrics, traces, logs) | OTLP, Datadog, New Relic, Dynatrace |

## Features

### LLM Observability (llmops)

- **Unified Interface**: Single API for tracing, evaluation, prompts, and datasets across all providers
- **Provider Agnostic**: Switch between Opik, Langfuse, Phoenix, and slog without changing your code
- **Full Tracing**: Trace LLM calls with spans, token usage, and cost tracking
- **Evaluation Support**: Run metrics and add feedback scores to traces
- **Dataset Management**: Create and manage evaluation datasets
- **Prompt Versioning**: Store and version prompt templates (provider-dependent)

### App Observability (observops)

- **Vendor-Agnostic**: Single API for OTLP, Datadog, New Relic, and Dynatrace
- **Full Telemetry**: Metrics (counters, gauges, histograms), distributed traces, and structured logs
- **slog Integration**: Trace-correlated logging with automatic context injection
- **Minimal Overhead**: No-op mode for disabled observability

### Common

- **Context Propagation**: Automatic trace/span context propagation via `context.Context`
- **Functional Options**: Clean, extensible configuration using the options pattern

## Quick Examples

### LLM Observability (llmops)

```go
import (
    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"  // Register Opik provider
)

provider, _ := llmops.Open("opik",
    llmops.WithAPIKey("your-api-key"),
    llmops.WithProjectName("my-project"),
)
defer provider.Close()

ctx, trace, _ := provider.StartTrace(ctx, "chat-workflow")
defer trace.End()

_, span, _ := provider.StartSpan(ctx, "gpt-4-completion",
    llmops.WithSpanType(llmops.SpanTypeLLM),
    llmops.WithModel("gpt-4"),
)
span.SetUsage(llmops.TokenUsage{TotalTokens: 18})
span.End()
```

### App Observability (observops)

```go
import (
    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/otlp"  // or datadog, newrelic
)

provider, _ := observops.Open("otlp",
    observops.WithEndpoint("localhost:4317"),
    observops.WithServiceName("my-service"),
    observops.WithInsecure(),
)
defer provider.Shutdown(ctx)

// Create metrics
counter, _ := provider.Meter().Counter("requests_total")
counter.Add(ctx, 1, observops.WithAttributes(
    observops.Attribute("method", "GET"),
))

// Create spans
ctx, span := provider.Tracer().Start(ctx, "handle-request")
defer span.End()

// slog integration with trace correlation
handler := provider.SlogHandler(
    observops.WithSlogLocalHandler(slog.NewJSONHandler(os.Stdout, nil)),
)
slog.SetDefault(slog.New(handler))
```

## Supported Providers

### LLM Providers (llmops)

| Provider | Package | Description |
|----------|---------|-------------|
| [Opik](providers/opik.md) | `go-opik/llmops` | Comet Opik - Open-source, full-featured |
| [Langfuse](providers/langfuse.md) | `omniobserve/llmops/langfuse` | Cloud & self-hosted, batch ingestion |
| [Phoenix](providers/phoenix.md) | `go-phoenix/llmops` | Arize Phoenix - OpenTelemetry-based |
| [slog](providers/slog.md) | `omniobserve/llmops/slog` | Local structured logging for development |

### App Observability Providers (observops)

| Provider | Package | Description |
|----------|---------|-------------|
| [OTLP](providers/otlp.md) | `omniobserve/observops/otlp` | OpenTelemetry Protocol - vendor-agnostic |
| [Datadog](providers/datadog.md) | `omniobserve/observops/datadog` | Datadog APM via OTLP |
| [New Relic](providers/newrelic.md) | `omniobserve/observops/newrelic` | New Relic via OTLP |
| [Dynatrace](providers/dynatrace.md) | `omniobserve/observops/dynatrace` | Dynatrace via OTLP |

## Next Steps

- [Installation](getting-started/installation.md) - Get OmniObserve set up
- [Quick Start](getting-started/quickstart.md) - Your first trace
- [Providers](providers/index.md) - Configure specific providers
- [OmniLLM Integration](integrations/omnillm.md) - Auto-instrument LLM calls
