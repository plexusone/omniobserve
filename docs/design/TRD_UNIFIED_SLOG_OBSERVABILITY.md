# Technical Reference Document: Unified slog and Observability Integration

**Status**: Draft
**Author**: Claude Code
**Date**: 2026-03-03
**Version**: 0.1.0

## 1. Executive Summary

This document proposes extending OmniObserve to provide unified integration between Go's standard `*slog.Logger` and the observability stack (observops). This enables applications to:

1. Use injectable `*slog.Logger` via context (standard Go pattern)
2. Automatically correlate logs with distributed traces
3. Export logs to any observops-supported backend (OTLP, Datadog, Dynatrace, etc.)
4. Maintain local console output alongside remote observability

## 2. Problem Statement

### Current State

Applications face fragmented observability tooling:

| Concern | Tool | Integration |
|---------|------|-------------|
| Local logging | `*slog.Logger` | Manual |
| Distributed tracing | OTel SDK | Separate |
| Metrics | OTel/Prometheus | Separate |
| LLM observability | Langfuse/Opik | Separate |
| APM | Datadog/Dynatrace | Vendor-specific |

This leads to:
- Multiple instrumentation points
- Inconsistent context propagation
- Duplicate effort correlating logs with traces
- Vendor lock-in for each concern

### Desired State

Single instrumentation with unified context:

```go
// One provider, all signals
provider, _ := omniobserve.Open("otlp", ...)

// slog.Logger with automatic trace correlation
logger := provider.SlogHandler().Logger()

// All signals share context
ctx, span := provider.Tracer().Start(ctx, "handleRequest")
logger.InfoContext(ctx, "processing", "user_id", userID)  // trace_id auto-attached
provider.Meter().Counter("requests").Add(ctx, 1)
```

## 3. Architecture

### 3.1 Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application Code                          │
│                                                                   │
│   ctx = slogutil.ContextWithLogger(ctx, logger)                  │
│   logger.InfoContext(ctx, "message", "key", value)               │
└───────────────────────────┬───────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    omniobserve.SlogHandler                       │
│                                                                   │
│   Implements slog.Handler                                        │
│   - Extracts trace_id/span_id from context                       │
│   - Fans out to multiple destinations                            │
│   - Converts slog.Record to observops.LogRecord                  │
└───────────────────────────┬───────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
     ┌──────────┐    ┌──────────┐    ┌──────────┐
     │  Local   │    │ observops│    │  llmops  │
     │  Output  │    │  Logger  │    │  Trace   │
     │ (stdout) │    │  (OTLP)  │    │  Events  │
     └──────────┘    └──────────┘    └──────────┘
            │               │               │
            ▼               ▼               ▼
       Console         Datadog/         Langfuse/
                      Dynatrace/          Opik/
                        OTel            Phoenix
```

### 3.2 Package Structure

```
omniobserve/
├── sloghandler/              # NEW: slog.Handler integration
│   ├── handler.go            # Main slog.Handler implementation
│   ├── fanout.go             # Multi-destination handler
│   ├── options.go            # Configuration options
│   └── trace.go              # Trace context extraction
├── observops/
│   ├── observops.go          # Provider interface (existing)
│   ├── otlp/                  # OTLP provider (existing)
│   ├── datadog/               # Datadog provider (extend)
│   ├── dynatrace/             # NEW: Dynatrace provider
│   └── prometheus/            # NEW: Prometheus provider (metrics only)
├── llmops/
│   └── (existing)
└── omniobserve.go            # Unified entry point
```

## 4. Detailed Design

### 4.1 SlogHandler Implementation

```go
package sloghandler

import (
    "context"
    "log/slog"

    "github.com/plexusone/omniobserve/observops"
)

// Handler implements slog.Handler with observability integration.
type Handler struct {
    // Local output handler (console, file, etc.)
    local slog.Handler

    // Remote observability logger
    remote observops.Logger

    // Minimum level for remote export
    remoteLevel slog.Level

    // Whether to include trace context
    includeTrace bool

    // Attribute processors
    processors []AttributeProcessor

    // Group and attrs for WithGroup/WithAttrs
    groups []string
    attrs  []slog.Attr
}

// Config holds handler configuration.
type Config struct {
    // Local handler for console/file output
    LocalHandler slog.Handler

    // Remote logger from observops.Provider
    RemoteLogger observops.Logger

    // Minimum level for remote export (default: Info)
    RemoteLevel slog.Level

    // Whether to extract and include trace context (default: true)
    IncludeTraceContext bool

    // Attribute processors for filtering/transforming
    Processors []AttributeProcessor
}

// New creates a new Handler with the given configuration.
func New(cfg Config) *Handler {
    if cfg.RemoteLevel == 0 {
        cfg.RemoteLevel = slog.LevelInfo
    }
    return &Handler{
        local:        cfg.LocalHandler,
        remote:       cfg.RemoteLogger,
        remoteLevel:  cfg.RemoteLevel,
        includeTrace: cfg.IncludeTraceContext,
        processors:   cfg.Processors,
    }
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
    // Enabled if either local or remote would handle it
    localEnabled := h.local != nil && h.local.Enabled(ctx, level)
    remoteEnabled := h.remote != nil && level >= h.remoteLevel
    return localEnabled || remoteEnabled
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
    // Extract trace context if available
    var traceAttrs []slog.Attr
    if h.includeTrace {
        if sc := TraceContextFromContext(ctx); sc.IsValid() {
            traceAttrs = []slog.Attr{
                slog.String("trace_id", sc.TraceID),
                slog.String("span_id", sc.SpanID),
            }
        }
    }

    // Apply attribute processors
    attrs := h.collectAttrs(r)
    for _, p := range h.processors {
        attrs = p.Process(attrs)
    }

    // Handle locally
    if h.local != nil && h.local.Enabled(ctx, r.Level) {
        localRecord := r.Clone()
        localRecord.AddAttrs(traceAttrs...)
        if err := h.local.Handle(ctx, localRecord); err != nil {
            // Log error but don't fail
        }
    }

    // Handle remotely
    if h.remote != nil && r.Level >= h.remoteLevel {
        logAttrs := h.toLogAttributes(attrs, traceAttrs)
        switch r.Level {
        case slog.LevelDebug:
            h.remote.Debug(ctx, r.Message, logAttrs...)
        case slog.LevelInfo:
            h.remote.Info(ctx, r.Message, logAttrs...)
        case slog.LevelWarn:
            h.remote.Warn(ctx, r.Message, logAttrs...)
        case slog.LevelError:
            h.remote.Error(ctx, r.Message, logAttrs...)
        }
    }

    return nil
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
    h2 := *h
    h2.attrs = append(h2.attrs, attrs...)
    if h.local != nil {
        h2.local = h.local.WithAttrs(attrs)
    }
    return &h2
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
    h2 := *h
    h2.groups = append(h2.groups, name)
    if h.local != nil {
        h2.local = h.local.WithGroup(name)
    }
    return &h2
}

// Logger returns an *slog.Logger using this handler.
func (h *Handler) Logger() *slog.Logger {
    return slog.New(h)
}
```

### 4.2 Trace Context Extraction

```go
package sloghandler

import (
    "context"

    "github.com/plexusone/omniobserve/observops"
    "go.opentelemetry.io/otel/trace"
)

// TraceContext holds trace identification.
type TraceContext struct {
    TraceID string
    SpanID  string
}

// IsValid returns true if the trace context has valid IDs.
func (tc TraceContext) IsValid() bool {
    return tc.TraceID != "" && tc.SpanID != ""
}

// TraceContextFromContext extracts trace context from the given context.
// It checks both observops.Span and OTel span.
func TraceContextFromContext(ctx context.Context) TraceContext {
    // Try observops first
    if span := observops.SpanFromContext(ctx); span != nil {
        sc := span.SpanContext()
        if sc.TraceID != "" {
            return TraceContext{
                TraceID: sc.TraceID,
                SpanID:  sc.SpanID,
            }
        }
    }

    // Fall back to OTel
    if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
        sc := span.SpanContext()
        return TraceContext{
            TraceID: sc.TraceID().String(),
            SpanID:  sc.SpanID().String(),
        }
    }

    return TraceContext{}
}
```

### 4.3 Provider Integration

```go
package omniobserve

import (
    "log/slog"
    "os"

    "github.com/plexusone/omniobserve/observops"
    "github.com/plexusone/omniobserve/sloghandler"
)

// UnifiedProvider combines observops and llmops capabilities.
type UnifiedProvider struct {
    observops observops.Provider
    llmops    llmops.Provider  // optional
    slogHandler *sloghandler.Handler
}

// SlogHandler returns an slog.Handler that integrates with this provider.
func (p *UnifiedProvider) SlogHandler(opts ...SlogOption) slog.Handler {
    cfg := &slogConfig{
        localHandler: slog.NewJSONHandler(os.Stdout, nil),
        remoteLevel:  slog.LevelInfo,
        includeTrace: true,
    }
    for _, opt := range opts {
        opt(cfg)
    }

    return sloghandler.New(sloghandler.Config{
        LocalHandler:        cfg.localHandler,
        RemoteLogger:        p.observops.Logger(),
        RemoteLevel:         cfg.remoteLevel,
        IncludeTraceContext: cfg.includeTrace,
        Processors:          cfg.processors,
    })
}

// Logger returns an *slog.Logger using the integrated handler.
func (p *UnifiedProvider) Logger(opts ...SlogOption) *slog.Logger {
    return slog.New(p.SlogHandler(opts...))
}

// slog options
type slogConfig struct {
    localHandler slog.Handler
    remoteLevel  slog.Level
    includeTrace bool
    processors   []sloghandler.AttributeProcessor
}

type SlogOption func(*slogConfig)

// WithLocalHandler sets the local output handler.
func WithLocalHandler(h slog.Handler) SlogOption {
    return func(c *slogConfig) { c.localHandler = h }
}

// WithRemoteLevel sets minimum level for remote export.
func WithRemoteLevel(level slog.Level) SlogOption {
    return func(c *slogConfig) { c.remoteLevel = level }
}

// WithoutTraceContext disables automatic trace context inclusion.
func WithoutTraceContext() SlogOption {
    return func(c *slogConfig) { c.includeTrace = false }
}

// WithLocalOnly disables remote logging.
func WithLocalOnly() SlogOption {
    return func(c *slogConfig) { c.remoteLevel = slog.Level(100) } // effectively never
}
```

### 4.4 Usage Examples

#### Basic Usage

```go
package main

import (
    "context"
    "log/slog"

    "github.com/plexusone/omniobserve"
    _ "github.com/plexusone/omniobserve/observops/otlp"
)

func main() {
    // Create unified provider
    provider, _ := omniobserve.Open("otlp",
        omniobserve.WithEndpoint("localhost:4317"),
        omniobserve.WithServiceName("my-service"),
    )
    defer provider.Shutdown(context.Background())

    // Get integrated slog.Logger
    logger := provider.Logger()

    // Use with context
    ctx := context.Background()
    ctx, span := provider.Tracer().Start(ctx, "handleRequest")
    defer span.End()

    // Logs automatically include trace_id and span_id
    logger.InfoContext(ctx, "processing request",
        "user_id", "u123",
        "action", "login",
    )
    // Output: {"time":"...","level":"INFO","msg":"processing request",
    //          "user_id":"u123","action":"login",
    //          "trace_id":"abc123","span_id":"def456"}
}
```

#### With Context-Based Logger Injection

```go
package main

import (
    "context"
    "net/http"

    "github.com/plexusone/omniobserve"
    "github.com/grokify/mogo/log/slogutil"
)

func main() {
    provider, _ := omniobserve.Open("otlp", ...)
    baseLogger := provider.Logger()

    // HTTP middleware
    http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Start trace
        ctx, span := provider.Tracer().Start(ctx, "handleUsers")
        defer span.End()

        // Inject request-scoped logger
        reqLogger := baseLogger.With("request_id", r.Header.Get("X-Request-ID"))
        ctx = slogutil.ContextWithLogger(ctx, reqLogger)

        // In handlers/services, retrieve from context
        handleUsers(ctx)
    })
}

func handleUsers(ctx context.Context) {
    // Get logger from context (with fallback)
    logger := slogutil.LoggerFromContext(ctx, slog.Default())

    // Logs include request_id + trace_id automatically
    logger.InfoContext(ctx, "fetching users")
}
```

#### Multi-Backend Configuration

```go
// Export to both Datadog and local console
provider, _ := omniobserve.OpenMulti(
    omniobserve.Backend("datadog",
        omniobserve.WithAPIKey(os.Getenv("DD_API_KEY")),
        omniobserve.WithSite(datadog.SiteUS1),
    ),
    omniobserve.Backend("otlp",
        omniobserve.WithEndpoint("localhost:4317"),
    ),
)

// Logger outputs to all configured backends
logger := provider.Logger(
    omniobserve.WithLocalHandler(slog.NewTextHandler(os.Stdout, nil)),
)
```

## 5. New Provider Implementations

### 5.1 Dynatrace Provider

```go
package dynatrace

import (
    "github.com/plexusone/omniobserve/observops"
)

const (
    // Dynatrace endpoints
    EndpointUS    = "https://{tenant}.live.dynatrace.com/api/v2/otlp"
    EndpointEU    = "https://{tenant}.live.dynatrace.eu/api/v2/otlp"
    EndpointManaged = "https://{tenant}/e/{environment}/api/v2/otlp"
)

// Config extends observops.Config with Dynatrace-specific options.
type Config struct {
    observops.Config

    // Tenant is the Dynatrace tenant ID.
    Tenant string

    // Environment is the environment ID (for managed deployments).
    Environment string

    // Region selects the Dynatrace region.
    Region Region
}

type Region string

const (
    RegionUS      Region = "us"
    RegionEU      Region = "eu"
    RegionManaged Region = "managed"
)

func init() {
    observops.Register("dynatrace", New)
    observops.RegisterInfo(observops.ProviderInfo{
        Name:        "dynatrace",
        Description: "Dynatrace observability platform",
        Website:     "https://dynatrace.com",
        Capabilities: []observops.Capability{
            observops.CapabilityMetrics,
            observops.CapabilityTraces,
            observops.CapabilityLogs,
        },
    })
}

// New creates a new Dynatrace provider.
func New(opts ...observops.ClientOption) (observops.Provider, error) {
    cfg := &Config{}
    observops.ApplyOptions(cfg, opts...)

    // Build endpoint from tenant/region
    endpoint := buildEndpoint(cfg)

    // Use OTLP provider with Dynatrace endpoint
    return otlp.NewWithEndpoint(endpoint,
        observops.WithHeaders(map[string]string{
            "Authorization": "Api-Token " + cfg.APIKey,
        }),
    )
}
```

### 5.2 Prometheus Provider (Metrics Only)

```go
package prometheus

import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/plexusone/omniobserve/observops"
)

func init() {
    observops.Register("prometheus", New)
    observops.RegisterInfo(observops.ProviderInfo{
        Name:        "prometheus",
        Description: "Prometheus metrics endpoint",
        Capabilities: []observops.Capability{
            observops.CapabilityMetrics,
        },
    })
}

// Provider implements observops.Provider for Prometheus.
type Provider struct {
    registry *prometheus.Registry
    meters   map[string]*promMeter
}

// Meter returns the Prometheus meter.
func (p *Provider) Meter() observops.Meter {
    return &promMeter{registry: p.registry}
}

// Tracer returns a no-op tracer (Prometheus doesn't support tracing).
func (p *Provider) Tracer() observops.Tracer {
    return observops.NoopTracer()
}

// Logger returns a no-op logger (Prometheus doesn't support logging).
func (p *Provider) Logger() observops.Logger {
    return observops.NoopLogger()
}

// Handler returns an http.Handler for the /metrics endpoint.
func (p *Provider) Handler() http.Handler {
    return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}
```

## 6. Migration Path

### Phase 1: sloghandler Package
- Implement `sloghandler.Handler`
- Add trace context extraction
- Unit tests for handler behavior

### Phase 2: Provider Integration
- Add `SlogHandler()` and `Logger()` to observops.Provider
- Update OTLP provider with slog integration
- Integration tests

### Phase 3: Additional Backends
- Implement Dynatrace provider
- Implement Prometheus provider
- Complete Datadog provider

### Phase 4: Unified Entry Point
- Create `omniobserve.Open()` unified entry point
- Add multi-backend support
- Documentation and examples

## 7. Compatibility

### Go Version
- Requires Go 1.21+ (slog in stdlib)

### Breaking Changes
- None - all additions are backward compatible

### Dependencies
- `log/slog` (stdlib)
- `go.opentelemetry.io/otel` (existing)

## 8. Testing Strategy

### Unit Tests
- Handler level enable/disable logic
- Trace context extraction
- Attribute processing

### Integration Tests
- End-to-end with OTLP collector
- Multi-backend fanout
- Context propagation through middleware

### Benchmarks
- Handler overhead vs plain slog
- Memory allocation per log entry
- Fanout to multiple backends

## 9. Open Questions

1. **Should llmops traces also be correlated with observops traces?**
   - Option A: Unified context (both share trace_id)
   - Option B: Separate but linkable (llmops has parent observops span)

2. **How to handle high-volume debug logs?**
   - Sampling at slog level?
   - Separate level thresholds for local vs remote?

3. **Should we support slog groups in remote export?**
   - OTLP logs have flat attributes
   - Could flatten: `group.subgroup.key = value`

## 10. References

- [Go slog package](https://pkg.go.dev/log/slog)
- [OpenTelemetry Logs Data Model](https://opentelemetry.io/docs/specs/otel/logs/data-model/)
- [Datadog OTLP Ingestion](https://docs.datadoghq.com/opentelemetry/)
- [Dynatrace OpenTelemetry](https://docs.dynatrace.com/docs/extend-dynatrace/opentelemetry)
