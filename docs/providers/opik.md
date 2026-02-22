# Opik

[Comet Opik](https://www.comet.com/site/products/opik/) is an open-source LLM observability platform with full-featured tracing, evaluation, and prompt management.

## Installation

```bash
go get github.com/agentplexus/go-opik
```

## Configuration

```go
import (
    "github.com/agentplexus/omniobserve/llmops"
    _ "github.com/agentplexus/go-opik/llmops"
)

provider, err := llmops.Open("opik",
    llmops.WithAPIKey("your-opik-api-key"),
    llmops.WithProjectName("my-project"),
    llmops.WithWorkspace("my-workspace"),  // Optional
)
```

## Features

| Feature | Supported |
|---------|:---------:|
| Tracing | :white_check_mark: |
| Evaluation | :white_check_mark: |
| Prompts | :white_check_mark: |
| Datasets | :white_check_mark: |
| Experiments | :white_check_mark: |
| Streaming | :white_check_mark: |
| Distributed Tracing | :white_check_mark: |
| Cost Tracking | :white_check_mark: |

## Working with Prompts

Opik has full prompt management support:

```go
// Create a versioned prompt
prompt, _ := provider.CreatePrompt(ctx, "chat-template",
    `You are a helpful assistant. User: {{.query}}`,
    llmops.WithPromptDescription("Main chat template"),
)

// Get a prompt
prompt, _ := provider.GetPrompt(ctx, "chat-template")

// Render with variables
rendered := prompt.Render(map[string]any{"query": "Hello!"})
```

## Direct SDK Access

For Opik-specific features:

```go
import "github.com/agentplexus/go-opik"

client := opik.NewClient(opik.Config{
    APIKey: "your-api-key",
})
```
