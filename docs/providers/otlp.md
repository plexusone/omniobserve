# OTLP Provider

The OTLP (OpenTelemetry Protocol) provider exports telemetry to any OTLP-compatible backend using gRPC.

## Supported Backends

- OpenTelemetry Collector
- Jaeger (with OTLP receiver)
- Tempo
- Any OTLP-compatible backend

## Installation

```go
import (
    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/otlp"
)
```

## Configuration

### Basic Usage

```go
provider, err := observops.Open("otlp",
    observops.WithEndpoint("localhost:4317"),
    observops.WithServiceName("my-service"),
)
```

### With TLS Disabled (Development)

```go
provider, err := observops.Open("otlp",
    observops.WithEndpoint("localhost:4317"),
    observops.WithServiceName("my-service"),
    observops.WithInsecure(),
)
```

### With Custom Headers

```go
provider, err := observops.Open("otlp",
    observops.WithEndpoint("collector.example.com:4317"),
    observops.WithServiceName("my-service"),
    observops.WithHeaders(map[string]string{
        "Authorization": "Bearer token",
    }),
)
```

### Full Configuration

```go
provider, err := observops.Open("otlp",
    observops.WithEndpoint("localhost:4317"),
    observops.WithServiceName("my-service"),
    observops.WithServiceVersion("1.0.0"),
    observops.WithInsecure(),
    observops.WithBatchTimeout(5*time.Second),
    observops.WithBatchSize(512),
    observops.WithResource(&observops.Resource{
        ServiceNamespace: "production",
        DeploymentEnv:    "prod",
        Attributes: map[string]string{
            "host.name": "server-01",
        },
    }),
)
```

## Environment Variables

The OTLP provider respects standard OpenTelemetry environment variables:

| Variable | Description |
|----------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP endpoint |
| `OTEL_EXPORTER_OTLP_HEADERS` | Headers as comma-separated key=value pairs |
| `OTEL_SERVICE_NAME` | Service name |
| `OTEL_RESOURCE_ATTRIBUTES` | Resource attributes |

## Endpoints

| Protocol | Default Port | Example |
|----------|--------------|---------|
| gRPC | 4317 | `localhost:4317` |
| HTTP | 4318 | `localhost:4318` |

## Capabilities

- Metrics (counters, gauges, histograms)
- Traces (distributed tracing)
- Logs (structured logging)
- Batching
- Sampling
- Resource Detection

## Example: OpenTelemetry Collector

docker-compose.yml:

```yaml
version: "3"
services:
  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"   # gRPC
      - "4318:4318"   # HTTP
    volumes:
      - ./otel-config.yaml:/etc/otel/config.yaml
    command: ["--config=/etc/otel/config.yaml"]
```

otel-config.yaml:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      exporters: [debug]
    logs:
      receivers: [otlp]
      exporters: [debug]
```
