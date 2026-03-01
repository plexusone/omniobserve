# Langfuse

[Langfuse](https://langfuse.com/) is an open-source LLM observability platform with cloud and self-hosted options.

## Installation

Langfuse is included in OmniObserve:

```bash
go get github.com/plexusone/omniobserve
```

## Configuration

```go
import (
    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/plexusone/omniobserve/llmops/langfuse"
)

provider, err := llmops.Open("langfuse",
    llmops.WithAPIKey("sk-lf-..."),
    llmops.WithEndpoint("https://cloud.langfuse.com"),  // Or self-hosted URL
)
```

## Features

| Feature | Supported |
|---------|:---------:|
| Tracing | :white_check_mark: |
| Evaluation | :white_check_mark: |
| Prompts | Partial |
| Datasets | :white_check_mark: |
| Experiments | :white_check_mark: |
| Streaming | :white_check_mark: |
| Cost Tracking | :white_check_mark: |

## Self-Hosted

Connect to a self-hosted Langfuse instance:

```go
provider, _ := llmops.Open("langfuse",
    llmops.WithAPIKey("sk-lf-..."),
    llmops.WithEndpoint("https://langfuse.your-company.com"),
)
```

## Direct SDK Access

For Langfuse-specific features:

```go
import "github.com/plexusone/omniobserve/sdk/langfuse"

client := langfuse.NewClient(langfuse.Config{
    PublicKey:  "pk-lf-...",
    SecretKey:  "sk-lf-...",
    BaseURL:    "https://cloud.langfuse.com",
})
```
