# Datadog Provider

The Datadog provider exports telemetry to Datadog using OTLP via gRPC.

## Installation

```go
import (
    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/datadog"
)
```

## Configuration

### Via Datadog Agent (Recommended)

The recommended approach is to send telemetry to a local Datadog Agent with OTLP ingestion enabled.

```go
provider, err := observops.Open("datadog",
    observops.WithServiceName("my-service"),
    // Uses default endpoint: localhost:4317
)
```

### Direct to Datadog (Without Agent)

For direct ingestion, provide your API key and endpoint:

```go
provider, err := observops.Open("datadog",
    observops.WithEndpoint("intake.datadoghq.com:443"),
    observops.WithAPIKey("your-datadog-api-key"),
    observops.WithServiceName("my-service"),
)
```

### With Environment Tags

```go
import "github.com/plexusone/omniobserve/observops/datadog"

provider, err := observops.Open("datadog",
    observops.WithServiceName("my-service"),
    datadog.WithDatadogEnv("production"),
    datadog.WithDatadogVersion("1.0.0"),
)
```

### With Site Selection

```go
import "github.com/plexusone/omniobserve/observops/datadog"

provider, err := observops.Open("datadog",
    observops.WithServiceName("my-service"),
    observops.WithAPIKey("your-api-key"),
    datadog.WithDatadogSite(datadog.SiteEU1), // EU datacenter
)
```

## Datadog Sites

| Site | Domain |
|------|--------|
| US1 (default) | `datadoghq.com` |
| US3 | `us3.datadoghq.com` |
| US5 | `us5.datadoghq.com` |
| EU1 | `datadoghq.eu` |
| AP1 | `ap1.datadoghq.com` |

## Datadog Agent Configuration

Enable OTLP ingestion in your `datadog.yaml`:

```yaml
otlp_config:
  receiver:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
```

Or via environment variables:

```bash
DD_OTLP_CONFIG_RECEIVER_PROTOCOLS_GRPC_ENDPOINT=0.0.0.0:4317
DD_OTLP_CONFIG_RECEIVER_PROTOCOLS_HTTP_ENDPOINT=0.0.0.0:4318
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DD_API_KEY` | Datadog API key (for direct ingestion) |
| `DD_SITE` | Datadog site (datadoghq.com, datadoghq.eu, etc.) |
| `OTEL_SERVICE_NAME` | Service name |

## Capabilities

- Metrics (counters, gauges, histograms)
- Traces (distributed tracing with APM)
- Logs (structured logging)
- Batching
- Sampling

## Example: Docker Compose with Datadog Agent

```yaml
version: "3"
services:
  datadog-agent:
    image: gcr.io/datadoghq/agent:latest
    environment:
      - DD_API_KEY=${DD_API_KEY}
      - DD_SITE=datadoghq.com
      - DD_OTLP_CONFIG_RECEIVER_PROTOCOLS_GRPC_ENDPOINT=0.0.0.0:4317
    ports:
      - "4317:4317"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /proc/:/host/proc/:ro
      - /sys/fs/cgroup/:/host/sys/fs/cgroup:ro
```

## Unified Tagging

Datadog uses unified service tagging. The provider automatically maps:

| observops | Datadog Tag |
|-----------|-------------|
| `ServiceName` | `service` |
| `ServiceVersion` | `version` |
| `DeploymentEnv` | `env` |
