# Providers

OmniObserve supports multiple observability backends through a unified interface. Each provider is registered via blank import and opened with `llmops.Open()`.

## Supported Providers

| Provider | Package | Description |
|----------|---------|-------------|
| [Opik](opik.md) | `github.com/agentplexus/go-opik/llmops` | Comet Opik - Open-source, full-featured |
| [Langfuse](langfuse.md) | `omniobserve/llmops/langfuse` | Cloud & self-hosted, batch ingestion |
| [Phoenix](phoenix.md) | `github.com/agentplexus/go-phoenix/llmops` | Arize Phoenix - OpenTelemetry-based |
| [slog](slog.md) | `omniobserve/llmops/slog` | Local structured logging for development |

## Provider Capabilities

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

Providers are registered via blank imports:

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

## Direct SDK Access

For provider-specific features, use the underlying SDKs directly:

```go
import "github.com/plexusone/omniobserve/sdk/langfuse"  // Langfuse SDK
import "github.com/agentplexus/go-opik"                   // Opik SDK
import "github.com/agentplexus/go-phoenix"                // Phoenix SDK
```
