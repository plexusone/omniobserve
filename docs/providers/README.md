# Observops Providers

This directory contains documentation for each supported observability backend provider.

## Available Providers

| Provider | Metrics | Traces | Logs | Protocol |
|----------|---------|--------|------|----------|
| [OTLP](otlp.md) | Yes | Yes | Yes | gRPC |
| [Datadog](datadog.md) | Yes | Yes | Yes | OTLP/gRPC |
| [Dynatrace](dynatrace.md) | Yes | Yes | Yes | OTLP/HTTP |
| [New Relic](newrelic.md) | Yes | Yes | Yes | OTLP/gRPC |

## Quick Start

```go
import (
    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/otlp" // or datadog, dynatrace, newrelic
)

provider, err := observops.Open("otlp",
    observops.WithEndpoint("localhost:4317"),
    observops.WithServiceName("my-service"),
)
if err != nil {
    log.Fatal(err)
}
defer provider.Shutdown(context.Background())
```

## Choosing a Provider

- **OTLP**: Use when you have an OpenTelemetry Collector or any OTLP-compatible backend
- **Datadog**: Use when sending directly to Datadog Agent or Datadog intake
- **Dynatrace**: Use when sending to Dynatrace SaaS, Managed, or ActiveGate
- **New Relic**: Use when sending directly to New Relic's OTLP endpoints

## slog Integration

All providers support `slog.Handler` integration for unified logging:

```go
// Get an slog handler from the provider
handler := provider.SlogHandler(
    observops.WithSlogLocalHandler(slog.NewJSONHandler(os.Stdout, nil)),
    observops.WithSlogRemoteLevel(int(slog.LevelWarn)),
)

logger := slog.New(handler)
logger.Info("hello world")  // local only
logger.Warn("warning!")     // local + remote
```
