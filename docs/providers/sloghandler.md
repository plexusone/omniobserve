# sloghandler Package

The `sloghandler` package provides `slog.Handler` implementations for unified logging with trace correlation and multi-destination output.

## Installation

```go
import "github.com/plexusone/omniobserve/sloghandler"
```

## Handlers

### Dual Handler

Routes logs to both local and remote handlers with independent level filtering.

```go
// Create local handler (console output)
local := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})

// Create remote handler (e.g., from observops provider)
remote := provider.SlogHandler()

// Combine: all logs go local, only Warn+ go remote
handler := sloghandler.Dual(local, remote,
    sloghandler.WithRemoteLevel(slog.LevelWarn),
)

logger := slog.New(handler)
logger.Debug("debug")  // local only
logger.Info("info")    // local only
logger.Warn("warn")    // local + remote
logger.Error("error")  // local + remote
```

### LocalOnly Handler

Wraps a local handler with trace context injection.

```go
handler := sloghandler.LocalOnly(
    slog.NewJSONHandler(os.Stdout, nil),
)

logger := slog.New(handler)

// With trace context
ctx, span := tracer.Start(ctx, "operation")
defer span.End()

logger.InfoContext(ctx, "message")
// Output includes trace_id and span_id
```

### RemoteOnly Handler

Sends logs only to a remote handler.

```go
handler := sloghandler.RemoteOnly(
    provider.SlogHandler(),
)
```

### Fanout Handler

Sends logs to multiple handlers simultaneously.

```go
h1 := slog.NewJSONHandler(os.Stdout, nil)
h2 := slog.NewTextHandler(logFile, nil)
h3 := provider.SlogHandler()

fanout := sloghandler.NewFanout([]slog.Handler{h1, h2, h3})
logger := slog.New(fanout)
```

#### Async Fanout

For non-blocking writes to slow handlers:

```go
fanout := sloghandler.NewFanout(
    []slog.Handler{h1, h2, h3},
    sloghandler.WithAsync(),
)
```

### Tee (Two-Way Fanout)

Convenience function for two handlers:

```go
handler := sloghandler.Tee(h1, h2)
// or async:
handler := sloghandler.TeeAsync(h1, h2)
```

## Configuration Options

### WithRemoteLevel

Sets the minimum level for remote logging:

```go
handler := sloghandler.Dual(local, remote,
    sloghandler.WithRemoteLevel(slog.LevelWarn),
)
```

### WithoutTraceContext

Disables automatic trace context injection:

```go
handler := sloghandler.LocalOnly(h,
    sloghandler.WithoutTraceContext(),
)
```

### WithTraceIDKey / WithSpanIDKey

Customizes the trace context attribute keys:

```go
handler := sloghandler.LocalOnly(h,
    sloghandler.WithTraceIDKey("traceId"),
    sloghandler.WithSpanIDKey("spanId"),
)
```

### WithProcessor

Adds attribute processors for redaction or transformation:

```go
// Redact sensitive fields
redactor := sloghandler.RedactProcessor("password", "secret", "token")

handler := sloghandler.LocalOnly(h,
    sloghandler.WithProcessor(redactor),
)

logger.Info("login", "username", "john", "password", "secret123")
// password value becomes "[REDACTED]"
```

## Trace Context

The handler automatically extracts trace context from OpenTelemetry spans:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("my-service")
ctx, span := tracer.Start(context.Background(), "operation")
defer span.End()

logger.InfoContext(ctx, "processing request")
// Output: {"msg":"processing request","trace_id":"abc123","span_id":"def456",...}
```

## Custom Trace Context Extractor

For non-OTel tracing systems:

```go
extractor := func(ctx context.Context) sloghandler.TraceContext {
    // Extract from your tracing system
    return sloghandler.TraceContext{
        TraceID: getTraceID(ctx),
        SpanID:  getSpanID(ctx),
    }
}

handler := sloghandler.LocalOnly(h,
    sloghandler.WithTraceContextExtractor(extractor),
)
```

## Integration with observops

The easiest way to use sloghandler is via an observops provider:

```go
import (
    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/otlp"
)

provider, _ := observops.Open("otlp",
    observops.WithEndpoint("localhost:4317"),
    observops.WithServiceName("my-service"),
)

// Provider returns an slog.Handler that:
// - Sends logs to the OTLP backend
// - Optionally outputs to a local handler
// - Automatically includes trace context
handler := provider.SlogHandler(
    observops.WithSlogLocalHandler(slog.NewJSONHandler(os.Stdout, nil)),
    observops.WithSlogRemoteLevel(int(slog.LevelWarn)),
)

slog.SetDefault(slog.New(handler))
```

## Performance

Benchmarks on typical hardware:

| Handler | ns/op | B/op | allocs/op |
|---------|-------|------|-----------|
| LocalOnly | ~700 | 400 | 1 |
| LocalOnly + TraceContext | ~700 | 400 | 1 |
| Fanout (2 handlers) | ~1000 | 800 | 0 |
| Fanout (async, 2 handlers) | ~1100 | 850 | 2 |

## Best Practices

1. **Use context-aware logging**: Always use `InfoContext`, `WarnContext`, etc. to enable trace correlation.

2. **Set appropriate remote levels**: Don't send Debug/Info to remote backends unless necessary.

3. **Redact sensitive data**: Use processors to redact passwords, tokens, and PII.

4. **Consider async for slow handlers**: Use `WithAsync()` for handlers that may block (network, file I/O).

5. **Reuse handlers**: Create handlers once and reuse them across the application.

## Example: Complete Setup

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/plexusone/omniobserve/observops"
    "github.com/plexusone/omniobserve/sloghandler"
    _ "github.com/plexusone/omniobserve/observops/otlp"
)

func main() {
    // Create observops provider
    provider, err := observops.Open("otlp",
        observops.WithEndpoint("localhost:4317"),
        observops.WithServiceName("my-service"),
        observops.WithInsecure(),
    )
    if err != nil {
        panic(err)
    }
    defer provider.Shutdown(context.Background())

    // Create local handler with pretty output
    local := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })

    // Create remote handler from provider
    remote := provider.SlogHandler()

    // Create dual handler with redaction
    redactor := sloghandler.RedactProcessor("password", "token", "secret")
    handler := sloghandler.Dual(local, remote,
        sloghandler.WithRemoteLevel(slog.LevelInfo),
        sloghandler.WithProcessor(redactor),
    )

    // Set as default logger
    slog.SetDefault(slog.New(handler))

    // Use throughout application
    ctx, span := provider.Tracer().Start(context.Background(), "main")
    defer span.End()

    slog.InfoContext(ctx, "application started",
        "version", "1.0.0",
        "password", "should-be-redacted",
    )
}
```
