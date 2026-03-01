# slog Provider

The slog provider logs trace events to Go's standard `log/slog` package. It's useful for local development, debugging, or as a fallback when no observability platform is configured.

## Installation

The slog provider is included in OmniObserve:

```bash
go get github.com/plexusone/omniobserve
```

## Configuration

```go
import (
    "log/slog"

    "github.com/plexusone/omniobserve/llmops"
    _ "github.com/plexusone/omniobserve/llmops/slog"
)

// Use default slog logger
provider, err := llmops.Open("slog",
    llmops.WithProjectName("my-project"),
)

// Or use a custom logger
provider, err := llmops.Open("slog",
    llmops.WithLogger(slog.Default()),
    llmops.WithProjectName("my-project"),
)
```

## Features

| Feature | Supported |
|---------|:---------:|
| Tracing | :white_check_mark: |
| Evaluation | :x: |
| Prompts | :x: |
| Datasets | :x: |
| Experiments | :x: |
| Streaming | :x: |

## Log Output

The slog provider logs trace and span events with structured attributes:

```
INFO trace started trace_id=abc123 name=chat-workflow project=my-project
INFO span started span_id=def456 trace_id=abc123 name=llm-call type=llm model=gpt-4
INFO span ended span_id=def456 trace_id=abc123 name=llm-call duration=1.5s prompt_tokens=10 completion_tokens=8
INFO trace ended trace_id=abc123 name=chat-workflow duration=2.1s
```

## Use Cases

- **Local Development**: See traces in your terminal without external services
- **Debugging**: Understand trace flow during development
- **Fallback**: Default provider when no other is configured
- **Testing**: Verify tracing instrumentation in tests

## Custom Logger

Configure with a JSON logger for structured output:

```go
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

provider, _ := llmops.Open("slog",
    llmops.WithLogger(logger),
)
```
