// Package observops provides a unified interface for general service observability platforms.
// It abstracts common functionality across providers using OpenTelemetry as the core
// instrumentation layer.
//
// This package complements the llmops package (which handles LLM-specific observability)
// by providing general service observability for:
//   - Metrics (counters, gauges, histograms)
//   - Traces (distributed tracing with spans)
//   - Logs (structured logging correlated with traces)
//
// Supported backends include:
//   - OTLP (OpenTelemetry Protocol) - vendor-agnostic
//   - New Relic
//   - Datadog
//   - Prometheus (metrics only)
//   - Jaeger (traces only)
//
// # Quick Start
//
// Import the provider you want to use:
//
//	import (
//		"github.com/agentplexus/omniobserve/observops"
//		_ "github.com/agentplexus/omniobserve/observops/otlp"     // OTLP exporter
//		// or
//		_ "github.com/agentplexus/omniobserve/observops/newrelic" // New Relic
//		// or
//		_ "github.com/agentplexus/omniobserve/observops/datadog"  // Datadog
//	)
//
// Then open a provider:
//
//	provider, err := observops.Open("otlp",
//		observops.WithEndpoint("localhost:4317"),
//		observops.WithServiceName("my-service"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer provider.Shutdown(context.Background())
//
// Use metrics:
//
//	counter, _ := provider.Meter().Counter("requests_total",
//		observops.WithDescription("Total number of requests"),
//	)
//	counter.Add(ctx, 1, observops.WithAttributes(
//		observops.Attribute("method", "GET"),
//		observops.Attribute("path", "/api/users"),
//	))
//
// Use tracing:
//
//	ctx, span := provider.Tracer().Start(ctx, "ProcessRequest")
//	defer span.End()
//	span.SetAttributes(observops.Attribute("user.id", "123"))
//
// Use structured logging:
//
//	provider.Logger().Info(ctx, "Request processed",
//		observops.LogAttribute("user_id", "123"),
//		observops.LogAttribute("duration_ms", 45),
//	)
package observops

import (
	"context"
	"time"
)

// Provider is the main interface for general observability backends.
// It provides access to metrics, traces, and logs.
type Provider interface {
	// Name returns the provider name (e.g., "otlp", "newrelic", "datadog").
	Name() string

	// Meter returns a metric meter for creating and recording metrics.
	Meter() Meter

	// Tracer returns a tracer for creating spans.
	Tracer() Tracer

	// Logger returns a structured logger.
	Logger() Logger

	// Shutdown gracefully shuts down the provider, flushing any buffered data.
	Shutdown(ctx context.Context) error

	// ForceFlush forces any buffered telemetry to be exported.
	ForceFlush(ctx context.Context) error
}

// Meter provides methods for creating metric instruments.
type Meter interface {
	// Counter creates a counter metric (monotonically increasing).
	Counter(name string, opts ...MetricOption) (Counter, error)

	// UpDownCounter creates a counter that can increase or decrease.
	UpDownCounter(name string, opts ...MetricOption) (UpDownCounter, error)

	// Histogram creates a histogram for recording value distributions.
	Histogram(name string, opts ...MetricOption) (Histogram, error)

	// Gauge creates a gauge metric for recording current values.
	Gauge(name string, opts ...MetricOption) (Gauge, error)
}

// Counter is a metric that only increases.
type Counter interface {
	// Add increments the counter by the given value.
	Add(ctx context.Context, value float64, opts ...RecordOption)
}

// UpDownCounter is a metric that can increase or decrease.
type UpDownCounter interface {
	// Add adds the given value (can be negative).
	Add(ctx context.Context, value float64, opts ...RecordOption)
}

// Histogram records a distribution of values.
type Histogram interface {
	// Record records a value in the histogram.
	Record(ctx context.Context, value float64, opts ...RecordOption)
}

// Gauge records a current value that can go up or down.
type Gauge interface {
	// Record records the current gauge value.
	Record(ctx context.Context, value float64, opts ...RecordOption)
}

// Tracer provides methods for creating spans.
type Tracer interface {
	// Start creates a new span with the given name.
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)

	// SpanFromContext retrieves the current span from context.
	SpanFromContext(ctx context.Context) Span
}

// Span represents a unit of work in a trace.
type Span interface {
	// End marks the span as finished.
	End(opts ...SpanEndOption)

	// SetAttributes sets attributes on the span.
	SetAttributes(attrs ...KeyValue)

	// SetStatus sets the span status.
	SetStatus(code StatusCode, description string)

	// RecordError records an error on the span.
	RecordError(err error, opts ...EventOption)

	// AddEvent adds an event to the span.
	AddEvent(name string, opts ...EventOption)

	// SpanContext returns the span's context.
	SpanContext() SpanContext

	// IsRecording returns true if the span is recording.
	IsRecording() bool
}

// SpanContext contains identifying trace information about a span.
type SpanContext struct {
	TraceID    string
	SpanID     string
	TraceFlags byte
	Remote     bool
}

// StatusCode represents the status of a span.
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOK
	StatusCodeError
)

// Logger provides structured logging methods.
type Logger interface {
	// Debug logs a debug message.
	Debug(ctx context.Context, msg string, attrs ...LogAttribute)

	// Info logs an info message.
	Info(ctx context.Context, msg string, attrs ...LogAttribute)

	// Warn logs a warning message.
	Warn(ctx context.Context, msg string, attrs ...LogAttribute)

	// Error logs an error message.
	Error(ctx context.Context, msg string, attrs ...LogAttribute)
}

// LogAttribute represents a key-value pair for log entries.
type LogAttribute struct {
	Key   string
	Value any
}

// LogAttr creates a log attribute.
func LogAttr(key string, value any) LogAttribute {
	return LogAttribute{Key: key, Value: value}
}

// KeyValue represents a key-value pair for span/metric attributes.
type KeyValue struct {
	Key   string
	Value any
}

// Attribute creates a key-value attribute.
func Attribute(key string, value any) KeyValue {
	return KeyValue{Key: key, Value: value}
}

// CapabilityChecker allows querying provider capabilities.
type CapabilityChecker interface {
	// HasCapability checks if the provider supports a given capability.
	HasCapability(cap Capability) bool

	// Capabilities returns all supported capabilities.
	Capabilities() []Capability
}

// Capability represents a specific feature a provider may support.
type Capability string

const (
	// CapabilityMetrics indicates support for metrics.
	CapabilityMetrics Capability = "metrics"
	// CapabilityTraces indicates support for distributed tracing.
	CapabilityTraces Capability = "traces"
	// CapabilityLogs indicates support for structured logging.
	CapabilityLogs Capability = "logs"
	// CapabilityExemplars indicates support for metric exemplars.
	CapabilityExemplars Capability = "exemplars"
	// CapabilityResourceDetection indicates support for automatic resource detection.
	CapabilityResourceDetection Capability = "resource_detection"
	// CapabilityBatching indicates support for telemetry batching.
	CapabilityBatching Capability = "batching"
	// CapabilitySampling indicates support for trace sampling.
	CapabilitySampling Capability = "sampling"
)

// SpanKind represents the type of span.
type SpanKind int

const (
	SpanKindInternal SpanKind = iota
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// SeverityLevel represents log severity.
type SeverityLevel int

const (
	SeverityDebug SeverityLevel = iota
	SeverityInfo
	SeverityWarn
	SeverityError
)

// Resource represents the entity producing telemetry.
type Resource struct {
	ServiceName      string
	ServiceVersion   string
	ServiceNamespace string
	DeploymentEnv    string
	Attributes       map[string]string
}

// Config holds common configuration for providers.
type Config struct {
	// ServiceName is the name of the service.
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// Endpoint is the backend endpoint.
	Endpoint string

	// APIKey is the API key for authentication.
	APIKey string //nolint:gosec // G117: APIKey is intentionally stored for backend authentication

	// Insecure disables TLS.
	Insecure bool

	// Headers are additional headers to send.
	Headers map[string]string

	// Resource is the resource describing the service.
	Resource *Resource

	// BatchTimeout is the maximum time to wait before exporting.
	BatchTimeout time.Duration

	// BatchSize is the maximum number of items per batch.
	BatchSize int

	// Disabled disables telemetry collection.
	Disabled bool

	// Debug enables debug logging.
	Debug bool
}
