# New Relic Provider

The New Relic provider exports telemetry to New Relic using OTLP via gRPC.

## Installation

```go
import (
    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/newrelic"
)
```

## Configuration

### US Region (Default)

```go
provider, err := observops.Open("newrelic",
    observops.WithAPIKey("YOUR_NEW_RELIC_LICENSE_KEY"),
    observops.WithServiceName("my-service"),
)
```

### EU Region

```go
import "github.com/plexusone/omniobserve/observops/newrelic"

provider, err := observops.Open("newrelic",
    observops.WithAPIKey("YOUR_NEW_RELIC_LICENSE_KEY"),
    observops.WithServiceName("my-service"),
    newrelic.WithNewRelicRegion(newrelic.RegionEU),
)
```

### Full Configuration

```go
provider, err := observops.Open("newrelic",
    observops.WithAPIKey("YOUR_NEW_RELIC_LICENSE_KEY"),
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

### Custom Endpoint

```go
provider, err := observops.Open("newrelic",
    observops.WithEndpoint("custom-otlp-endpoint:4317"),
    observops.WithAPIKey("YOUR_NEW_RELIC_LICENSE_KEY"),
    observops.WithServiceName("my-service"),
)
```

## Regions and Endpoints

| Region | Endpoint |
|--------|----------|
| US (default) | `otlp.nr-data.net:4317` |
| EU | `otlp.eu01.nr-data.net:4317` |

## License Key

The New Relic provider requires a License Key (Ingest - License type):

1. Go to **API Keys** in New Relic
2. Create a new key of type **Ingest - License**
3. Copy the key (format: `NRAK-...` or legacy format)

**Note**: This is different from a User API Key. You need an Ingest License Key.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `NEW_RELIC_LICENSE_KEY` | New Relic license key |
| `OTEL_SERVICE_NAME` | Service name |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Custom endpoint (optional) |

## Capabilities

- Metrics (counters, gauges, histograms)
- Traces (distributed tracing)
- Logs (structured logging)
- Batching
- Sampling

## Viewing Data in New Relic

### Traces

1. Go to **APM & Services**
2. Select your service
3. Click **Distributed tracing**

### Metrics

1. Go to **Query your data**
2. Use NRQL: `FROM Metric SELECT * WHERE service.name = 'my-service'`

### Logs

1. Go to **Logs**
2. Filter by `service.name`

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
    _ "github.com/plexusone/omniobserve/observops/newrelic"
)

func main() {
    provider, err := observops.Open("newrelic",
        observops.WithAPIKey(os.Getenv("NEW_RELIC_LICENSE_KEY")),
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
    counter, _ := provider.Meter().Counter("requests_total",
        observops.WithDescription("Total number of requests"),
    )
    counter.Add(ctx, 1, observops.WithAttributes(
        observops.Attribute("endpoint", "/api/users"),
    ))

    time.Sleep(100 * time.Millisecond)
    logger.InfoContext(ctx, "Application finished")
}
```

## NRQL Queries

### Find Traces

```sql
FROM Span SELECT *
WHERE service.name = 'my-service'
SINCE 1 hour ago
```

### Find Metrics

```sql
FROM Metric SELECT average(value)
WHERE service.name = 'my-service'
FACET metricName
SINCE 1 hour ago
```

### Find Logs

```sql
FROM Log SELECT *
WHERE service.name = 'my-service'
SINCE 1 hour ago
```

## Troubleshooting

### 403 Forbidden

- Verify your license key is correct
- Ensure you're using an Ingest License Key, not a User API Key
- Check that the key hasn't been revoked

### No Data Appearing

- Verify telemetry is being sent (enable debug mode)
- Check the correct region is configured
- Wait a few minutes for data to appear in the UI
- Verify the service name matches your filters
