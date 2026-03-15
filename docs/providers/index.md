# Providers

OmniObserve provides two provider systems:

- **llmops** - LLM observability for AI/ML applications (tracing, evaluation, prompts, datasets)
- **observops** - General application observability (metrics, traces, logs)

---

## LLM Providers (llmops)

For AI/ML application observability. Register via blank import and open with `llmops.Open()`.

| Provider | Package | Description |
|----------|---------|-------------|
| [Opik](opik.md) | `github.com/agentplexus/go-opik/llmops` | Comet Opik - Open-source, full-featured |
| [Langfuse](langfuse.md) | `omniobserve/llmops/langfuse` | Cloud & self-hosted, batch ingestion |
| [Phoenix](phoenix.md) | `github.com/agentplexus/go-phoenix/llmops` | Arize Phoenix - OpenTelemetry-based |
| [slog](slog.md) | `omniobserve/llmops/slog` | Local structured logging for development |

---

## App Observability Providers (observops)

For general application observability. Register via blank import and open with `observops.Open()`.

| Provider | Package | Description |
|----------|---------|-------------|
| [OTLP](otlp.md) | `omniobserve/observops/otlp` | OpenTelemetry Protocol - vendor-agnostic |
| [Datadog](datadog.md) | `omniobserve/observops/datadog` | Datadog APM via OTLP |
| [New Relic](newrelic.md) | `omniobserve/observops/newrelic` | New Relic via OTLP |
| [Dynatrace](dynatrace.md) | `omniobserve/observops/dynatrace` | Dynatrace via OTLP |

See also: [sloghandler](sloghandler.md) for slog integration with trace correlation.

---

## observops Capabilities

| Feature | OTLP | Datadog | New Relic | Dynatrace |
|---------|:----:|:-------:|:---------:|:---------:|
| Metrics | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| Traces | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| Logs | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| slog Handler | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |

---

## LLM Provider Capabilities

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

## Provider Registration

### LLM Providers (llmops)

```go
import (
    "github.com/plexusone/omniobserve/llmops"

    // Register one or more providers
    _ "github.com/agentplexus/go-opik/llmops"
    _ "github.com/plexusone/omniobserve/llmops/langfuse"
    _ "github.com/agentplexus/go-phoenix/llmops"
    _ "github.com/plexusone/omniobserve/llmops/slog"
)

// Open by name
provider, err := llmops.Open("opik", llmops.WithAPIKey("..."))
```

### App Observability Providers (observops)

```go
import (
    "github.com/plexusone/omniobserve/observops"

    // Register one provider
    _ "github.com/plexusone/omniobserve/observops/otlp"     // OTLP
    // or
    _ "github.com/plexusone/omniobserve/observops/datadog"  // Datadog
    // or
    _ "github.com/plexusone/omniobserve/observops/newrelic" // New Relic
)

// Open by name
provider, err := observops.Open("otlp",
    observops.WithEndpoint("localhost:4317"),
    observops.WithServiceName("my-service"),
)
defer provider.Shutdown(ctx)
```

## Direct SDK Access

For provider-specific features, use the underlying SDKs directly:

```go
import "github.com/plexusone/omniobserve/sdk/langfuse"  // Langfuse SDK
import "github.com/agentplexus/go-opik"                   // Opik SDK
import "github.com/agentplexus/go-phoenix"                // Phoenix SDK
```
