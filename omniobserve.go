// Package omniobserve provides a unified interface for LLM, ML, and general service observability platforms.
//
// This library abstracts common functionality across providers like:
//   - Comet Opik
//   - Arize Phoenix
//   - Langfuse
//   - New Relic
//   - Datadog
//   - OpenTelemetry (OTLP)
//
// # Quick Start - LLM Observability
//
// Import the provider you want to use:
//
//	import (
//		"github.com/plexusone/omniobserve/llmops"
//		_ "github.com/agentplexus/go-opik/llmops"             // Register Opik
//		// or
//		_ "github.com/plexusone/omniobserve/llmops/langfuse" // Register Langfuse
//		// or
//		_ "github.com/agentplexus/go-phoenix/llmops"           // Register Phoenix
//	)
//
// Then open a provider:
//
//	provider, err := llmops.Open("opik", llmops.WithAPIKey("..."))
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer provider.Close()
//
// Start tracing:
//
//	ctx, trace, _ := provider.StartTrace(ctx, "my-workflow")
//	defer trace.End()
//
//	ctx, span, _ := provider.StartSpan(ctx, "llm-call",
//		llmops.WithSpanType(llmops.SpanTypeLLM),
//		llmops.WithModel("gpt-4"),
//	)
//	defer span.End()
//
// # Quick Start - General Service Observability
//
// Import the provider you want to use:
//
//	import (
//		"github.com/plexusone/omniobserve/observops"
//		_ "github.com/plexusone/omniobserve/observops/otlp"     // OTLP exporter
//		// or
//		_ "github.com/plexusone/omniobserve/observops/newrelic" // New Relic
//		// or
//		_ "github.com/plexusone/omniobserve/observops/datadog"  // Datadog
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
// Use metrics, traces, and logs:
//
//	// Metrics
//	counter, _ := provider.Meter().Counter("requests_total")
//	counter.Add(ctx, 1, observops.WithAttributes(observops.Attribute("method", "GET")))
//
//	// Traces
//	ctx, span := provider.Tracer().Start(ctx, "ProcessRequest")
//	defer span.End()
//
//	// Logs
//	provider.Logger().Info(ctx, "Request processed", observops.LogAttr("user_id", "123"))
//
// # Architecture
//
// The library is organized into three main packages:
//
//   - llmops: LLM observability (traces, spans, evaluations, prompts)
//   - mlops: ML operations (experiments, model registry, artifacts)
//   - observops: General service observability (metrics, traces, logs via OpenTelemetry)
//
// Each package defines interfaces that providers implement. Provider-specific
// implementations are in subpackages (e.g., llmops/opik, observops/newrelic).
//
// # LLM Observability Providers
//
//   - opik: Comet Opik (open-source, self-hosted)
//   - langfuse: Langfuse (open-source, cloud & self-hosted)
//   - phoenix: Arize Phoenix (open-source, uses OpenTelemetry)
//
// # General Observability Providers
//
//   - otlp: OpenTelemetry Protocol (vendor-agnostic)
//   - newrelic: New Relic
//   - datadog: Datadog
//
// # Features
//
// LLM Observability:
//   - Trace/span creation and context propagation
//   - Input/output capture
//   - Token usage and cost tracking
//   - Feedback scores and evaluations
//   - Dataset management
//   - Prompt versioning (provider-dependent)
//
// General Observability:
//   - Metrics (counters, gauges, histograms)
//   - Distributed tracing with spans
//   - Structured logging with trace correlation
//   - Vendor-agnostic via OpenTelemetry
//
// # SDK Access
//
// For provider-specific features, you can use the underlying SDKs directly:
//
//	import "github.com/agentplexus/go-opik"    // Opik SDK
//	import "github.com/agentplexus/go-phoenix" // Phoenix SDK
//	import "github.com/plexusone/omniobserve/sdk/langfuse"
package omniobserve

import (
	"github.com/plexusone/omniobserve/llmops"
	"github.com/plexusone/omniobserve/mlops"
	"github.com/plexusone/omniobserve/observops"
)

// Version is the library version.
const Version = "0.1.0"

// Re-export commonly used types for convenience.
type (
	// Provider is an alias for llmops.Provider.
	Provider = llmops.Provider

	// Trace is an alias for llmops.Trace.
	Trace = llmops.Trace

	// Span is an alias for llmops.Span.
	Span = llmops.Span

	// TokenUsage is an alias for llmops.TokenUsage.
	TokenUsage = llmops.TokenUsage

	// SpanType is an alias for llmops.SpanType.
	SpanType = llmops.SpanType

	// EvalInput is an alias for llmops.EvalInput.
	EvalInput = llmops.EvalInput

	// EvalResult is an alias for llmops.EvalResult.
	EvalResult = llmops.EvalResult

	// MLProvider is an alias for mlops.Provider.
	MLProvider = mlops.Provider

	// Experiment is an alias for mlops.Experiment.
	Experiment = mlops.Experiment

	// Run is an alias for mlops.Run.
	Run = mlops.Run

	// Model is an alias for mlops.Model.
	Model = mlops.Model
)

// Span type constants.
const (
	SpanTypeGeneral   = llmops.SpanTypeGeneral
	SpanTypeLLM       = llmops.SpanTypeLLM
	SpanTypeTool      = llmops.SpanTypeTool
	SpanTypeRetrieval = llmops.SpanTypeRetrieval
	SpanTypeAgent     = llmops.SpanTypeAgent
	SpanTypeChain     = llmops.SpanTypeChain
	SpanTypeGuardrail = llmops.SpanTypeGuardrail
)

// OpenLLMOps opens an LLM observability provider.
// This is a convenience function that wraps llmops.Open.
func OpenLLMOps(name string, opts ...llmops.ClientOption) (llmops.Provider, error) {
	return llmops.Open(name, opts...)
}

// Providers returns the names of registered LLM providers.
func Providers() []string {
	return llmops.Providers()
}

// Re-export option functions for convenience.
var (
	// Client options
	WithAPIKey      = llmops.WithAPIKey
	WithEndpoint    = llmops.WithEndpoint
	WithWorkspace   = llmops.WithWorkspace
	WithProjectName = llmops.WithProjectName
	WithHTTPClient  = llmops.WithHTTPClient
	WithTimeout     = llmops.WithTimeout
	WithDisabled    = llmops.WithDisabled
	WithDebug       = llmops.WithDebug

	// Trace options
	WithTraceProject  = llmops.WithTraceProject
	WithTraceInput    = llmops.WithTraceInput
	WithTraceOutput   = llmops.WithTraceOutput
	WithTraceMetadata = llmops.WithTraceMetadata
	WithTraceTags     = llmops.WithTraceTags
	WithThreadID      = llmops.WithThreadID

	// Span options
	WithSpanType     = llmops.WithSpanType
	WithSpanInput    = llmops.WithSpanInput
	WithSpanOutput   = llmops.WithSpanOutput
	WithSpanMetadata = llmops.WithSpanMetadata
	WithSpanTags     = llmops.WithSpanTags
	WithModel        = llmops.WithModel
	WithProvider     = llmops.WithProvider
	WithTokenUsage   = llmops.WithTokenUsage

	// End options
	WithEndOutput   = llmops.WithEndOutput
	WithEndMetadata = llmops.WithEndMetadata
	WithEndError    = llmops.WithEndError
)

// =============================================================================
// General Service Observability (observops)
// =============================================================================

// ObservopsProvider is an alias for observops.Provider.
type ObservopsProvider = observops.Provider

// Meter is an alias for observops.Meter.
type Meter = observops.Meter

// Tracer is an alias for observops.Tracer.
type Tracer = observops.Tracer

// Logger is an alias for observops.Logger.
type Logger = observops.Logger

// ObservopsSpan is an alias for observops.Span.
type ObservopsSpan = observops.Span

// Counter is an alias for observops.Counter.
type Counter = observops.Counter

// Histogram is an alias for observops.Histogram.
type Histogram = observops.Histogram

// Gauge is an alias for observops.Gauge.
type Gauge = observops.Gauge

// OpenObservops opens a general observability provider.
// This is a convenience function that wraps observops.Open.
func OpenObservops(name string, opts ...observops.ClientOption) (observops.Provider, error) {
	return observops.Open(name, opts...)
}

// ObservopsProviders returns the names of registered general observability providers.
func ObservopsProviders() []string {
	return observops.Providers()
}

// Re-export observops option functions for convenience.
var (
	// Observops client options
	ObsWithServiceName    = observops.WithServiceName
	ObsWithServiceVersion = observops.WithServiceVersion
	ObsWithEndpoint       = observops.WithEndpoint
	ObsWithAPIKey         = observops.WithAPIKey
	ObsWithInsecure       = observops.WithInsecure
	ObsWithHeaders        = observops.WithHeaders
	ObsWithResource       = observops.WithResource
	ObsWithBatchTimeout   = observops.WithBatchTimeout
	ObsWithBatchSize      = observops.WithBatchSize
	ObsWithDisabled       = observops.WithDisabled
	ObsWithDebug          = observops.WithDebug

	// Metric options
	ObsWithDescription = observops.WithDescription
	ObsWithUnit        = observops.WithUnit

	// Record options
	ObsWithAttributes = observops.WithAttributes

	// Span options
	ObsWithSpanKind       = observops.WithSpanKind
	ObsWithSpanAttributes = observops.WithSpanAttributes
	ObsWithSpanLinks      = observops.WithSpanLinks

	// Attribute helper
	ObsAttribute = observops.Attribute
	ObsLogAttr   = observops.LogAttr
)
