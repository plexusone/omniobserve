# Dynatrace Provider

The Dynatrace provider exports telemetry to Dynatrace using OTLP via HTTP.

## Installation

```go
import (
    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/dynatrace"
)
```

## Configuration

### Dynatrace SaaS

```go
provider, err := observops.Open("dynatrace",
    observops.WithEndpoint("https://{environment-id}.live.dynatrace.com/api/v2/otlp"),
    observops.WithAPIKey("dt0c01.XXXXXXXX.YYYYYYYY"),
    observops.WithServiceName("my-service"),
)
```

### Dynatrace Managed

```go
provider, err := observops.Open("dynatrace",
    observops.WithEndpoint("https://{your-domain}/e/{environment-id}/api/v2/otlp"),
    observops.WithAPIKey("dt0c01.XXXXXXXX.YYYYYYYY"),
    observops.WithServiceName("my-service"),
)
```

### Via ActiveGate

```go
provider, err := observops.Open("dynatrace",
    observops.WithEndpoint("https://{activegate-address}:9999/e/{environment-id}/api/v2/otlp"),
    observops.WithAPIKey("dt0c01.XXXXXXXX.YYYYYYYY"),
    observops.WithServiceName("my-service"),
)
```

### Full Configuration

```go
provider, err := observops.Open("dynatrace",
    observops.WithEndpoint("https://abc12345.live.dynatrace.com/api/v2/otlp"),
    observops.WithAPIKey("dt0c01.XXXXXXXX.YYYYYYYY"),
    observops.WithServiceName("my-service"),
    observops.WithServiceVersion("1.0.0"),
    observops.WithBatchTimeout(5*time.Second),
    observops.WithResource(&observops.Resource{
        DeploymentEnv: "production",
        Attributes: map[string]string{
            "host.name": "server-01",
        },
    }),
)
```

## Endpoint Formats

| Deployment | Endpoint Format |
|------------|-----------------|
| SaaS | `https://{environment-id}.live.dynatrace.com/api/v2/otlp` |
| Managed | `https://{your-domain}/e/{environment-id}/api/v2/otlp` |
| ActiveGate | `https://{activegate}:9999/e/{environment-id}/api/v2/otlp` |

## API Token Setup

Create an API token in Dynatrace with the following scopes:

| Scope | Purpose |
|-------|---------|
| `openTelemetryTrace.ingest` | Traces |
| `metrics.ingest` | Metrics |
| `logs.ingest` | Logs |

### Creating a Token

1. Go to **Settings > Integration > Dynatrace API**
2. Click **Generate token**
3. Name your token (e.g., "OmniObserve OTLP Ingest")
4. Select the required scopes
5. Click **Generate**
6. Copy the token (starts with `dt0c01.`)

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DT_API_TOKEN` | Dynatrace API token |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Dynatrace OTLP endpoint |
| `OTEL_SERVICE_NAME` | Service name |

## Capabilities

- Metrics (counters, gauges, histograms)
- Traces (distributed tracing)
- Logs (structured logging)
- Batching
- Sampling
- Resource Detection

## Protocol Notes

Unlike the OTLP and Datadog providers which use gRPC, the Dynatrace provider uses HTTP for OTLP ingestion. This is because Dynatrace's OTLP endpoints are HTTP-based.

The provider automatically:

- Strips the `/api/v2/otlp` suffix from your endpoint
- Appends the correct paths (`/v1/traces`, `/v1/metrics`)
- Sets the `Authorization: Api-Token {token}` header

## Troubleshooting

### 401 Unauthorized

- Verify your API token is correct
- Check that the token has the required scopes
- Ensure the token hasn't expired

### 403 Forbidden

- Verify the environment ID in your endpoint is correct
- Check that your token has access to that environment

### Connection Refused

- Verify the endpoint URL is correct
- For ActiveGate, ensure port 9999 is accessible
- Check firewall rules

## Example: Full Application

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "os"
    "time"

    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/dynatrace"
)

func main() {
    provider, err := observops.Open("dynatrace",
        observops.WithEndpoint(os.Getenv("DT_ENDPOINT")),
        observops.WithAPIKey(os.Getenv("DT_API_TOKEN")),
        observops.WithServiceName("my-service"),
        observops.WithServiceVersion("1.0.0"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(context.Background())

    // Set up logging
    handler := provider.SlogHandler(
        observops.WithSlogLocalHandler(slog.NewTextHandler(os.Stdout, nil)),
    )
    logger := slog.New(handler)

    // Create a trace
    ctx, span := provider.Tracer().Start(context.Background(), "main")
    defer span.End()

    // Log with trace correlation
    logger.InfoContext(ctx, "Application started")

    // Record a metric
    counter, _ := provider.Meter().Counter("requests_total")
    counter.Add(ctx, 1)

    time.Sleep(100 * time.Millisecond)
    logger.InfoContext(ctx, "Application finished")
}
```
