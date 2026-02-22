# Phoenix

[Arize Phoenix](https://phoenix.arize.com/) is an open-source observability platform built on OpenTelemetry.

## Installation

```bash
go get github.com/agentplexus/go-phoenix
```

## Configuration

```go
import (
    "github.com/agentplexus/omniobserve/llmops"
    _ "github.com/agentplexus/go-phoenix/llmops"
)

provider, err := llmops.Open("phoenix",
    llmops.WithEndpoint("http://localhost:6006"),
)
```

## Features

| Feature | Supported |
|---------|:---------:|
| Tracing | :white_check_mark: |
| Evaluation | :white_check_mark: |
| Prompts | :x: |
| Datasets | Partial |
| Experiments | Partial |
| Distributed Tracing | :white_check_mark: |
| OpenTelemetry | :white_check_mark: |

## Running Phoenix Locally

```bash
# Using Docker
docker run -p 6006:6006 arizephoenix/phoenix

# Or using pip
pip install arize-phoenix
phoenix serve
```

## OpenTelemetry Integration

Phoenix is built on OpenTelemetry, making it compatible with the broader OTel ecosystem:

```go
provider, _ := llmops.Open("phoenix",
    llmops.WithEndpoint("http://localhost:6006"),
)
// Traces are exported using OTLP protocol
```

## Direct SDK Access

```go
import "github.com/agentplexus/go-phoenix"

client := phoenix.NewClient(phoenix.Config{
    Endpoint: "http://localhost:6006",
})
```
