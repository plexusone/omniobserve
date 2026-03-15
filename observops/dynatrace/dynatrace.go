// Package dynatrace provides a Dynatrace observability provider for observops.
// It uses OpenTelemetry Protocol (OTLP) to send telemetry data to Dynatrace.
//
// # Usage
//
//	import (
//		"github.com/plexusone/omniobserve/observops"
//		_ "github.com/plexusone/omniobserve/observops/dynatrace"
//	)
//
//	provider, err := observops.Open("dynatrace",
//		observops.WithEndpoint("https://{environment-id}.live.dynatrace.com/api/v2/otlp"),
//		observops.WithAPIKey("dt0c01.XXXXXXXX.YYYYYYYY"),
//		observops.WithServiceName("my-service"),
//	)
//
// # Configuration
//
// Required:
//   - API Token: A Dynatrace API token with the following scopes:
//   - metrics.ingest (for metrics)
//   - traces.ingest (for traces)
//   - logs.ingest (for logs)
//   - Endpoint: Your Dynatrace OTLP endpoint
//   - Service Name: The name of your service
//
// # Endpoint Formats
//
// For Dynatrace SaaS:
//
//	https://{environment-id}.live.dynatrace.com/api/v2/otlp
//
// For Dynatrace Managed:
//
//	https://{your-domain}/e/{environment-id}/api/v2/otlp
//
// For ActiveGate:
//
//	https://{activegate-address}:9999/e/{environment-id}/api/v2/otlp
//
// # Environment Variables
//
// You can also configure via environment variables:
//   - DT_API_TOKEN: Your Dynatrace API token
//   - OTEL_EXPORTER_OTLP_ENDPOINT: Your Dynatrace OTLP endpoint
//   - OTEL_SERVICE_NAME: Service name
//
// # Token Scopes
//
// Create an API token in Dynatrace with the following scopes:
//   - openTelemetryTrace.ingest (for traces)
//   - metrics.ingest (for metrics)
//   - logs.ingest (for logs)
package dynatrace

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/plexusone/omniobserve/observops"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	observops.Register("dynatrace", New)
	observops.RegisterInfo(observops.ProviderInfo{
		Name:        "dynatrace",
		Description: "Dynatrace observability platform",
		Website:     "https://www.dynatrace.com",
		OpenSource:  false,
		SelfHosted:  true, // Managed option available
		Capabilities: []observops.Capability{
			observops.CapabilityMetrics,
			observops.CapabilityTraces,
			observops.CapabilityLogs,
			observops.CapabilityBatching,
			observops.CapabilitySampling,
			observops.CapabilityResourceDetection,
		},
	})
}

// Provider implements observops.Provider for Dynatrace.
type Provider struct {
	cfg            *observops.Config
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	logger         *dtLogger
	shutdown       bool
	mu             sync.RWMutex
}

// New creates a new Dynatrace provider.
func New(opts ...observops.ClientOption) (observops.Provider, error) {
	cfg := observops.ApplyOptions(opts...)

	if cfg.Disabled {
		return newNoopProvider(), nil
	}

	if cfg.APIKey == "" {
		return nil, observops.ErrMissingAPIKey
	}

	if cfg.ServiceName == "" {
		return nil, observops.ErrMissingServiceName
	}

	if cfg.Endpoint == "" {
		return nil, observops.ErrMissingEndpoint
	}

	// Set up headers with API token
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	cfg.Headers["Authorization"] = "Api-Token " + cfg.APIKey

	p := &Provider{
		cfg: cfg,
	}

	ctx := context.Background()

	// Build resource
	res := p.buildResource()

	// Initialize trace exporter
	if err := p.initTracing(ctx, res); err != nil {
		return nil, observops.WrapError("dynatrace", "initTracing", err)
	}

	// Initialize metric exporter
	if err := p.initMetrics(ctx, res); err != nil {
		_ = p.tracerProvider.Shutdown(ctx)
		return nil, observops.WrapError("dynatrace", "initMetrics", err)
	}

	// Initialize logger
	p.logger = &dtLogger{
		tracer: p.tracer,
	}

	return p, nil
}

func (p *Provider) buildResource() *resource.Resource {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(p.cfg.ServiceName),
	}

	if p.cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersionKey.String(p.cfg.ServiceVersion))
	}

	if p.cfg.Resource != nil {
		if p.cfg.Resource.ServiceNamespace != "" {
			attrs = append(attrs, semconv.ServiceNamespaceKey.String(p.cfg.Resource.ServiceNamespace))
		}
		if p.cfg.Resource.DeploymentEnv != "" {
			attrs = append(attrs, attribute.String("deployment.environment", p.cfg.Resource.DeploymentEnv))
		}
		for k, v := range p.cfg.Resource.Attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
	}

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		attrs...,
	)
}

// parseEndpoint extracts the base URL from the OTLP endpoint.
// Dynatrace endpoint format: https://{env}.live.dynatrace.com/api/v2/otlp
// We need to strip /api/v2/otlp for the HTTP exporter as it appends paths.
func parseEndpoint(endpoint string) string {
	// Remove trailing slash
	endpoint = strings.TrimSuffix(endpoint, "/")

	// Remove /api/v2/otlp suffix if present (exporter adds /v1/traces, /v1/metrics)
	endpoint = strings.TrimSuffix(endpoint, "/api/v2/otlp")

	// Remove protocol prefix for the exporter (it uses WithEndpoint which expects host:port or host)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	return endpoint
}

func (p *Provider) initTracing(ctx context.Context, res *resource.Resource) error {
	endpoint := parseEndpoint(p.cfg.Endpoint)

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithHeaders(p.cfg.Headers),
		otlptracehttp.WithURLPath("/api/v2/otlp/v1/traces"),
	}

	if p.cfg.Insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return err
	}

	batchOpts := []sdktrace.BatchSpanProcessorOption{}
	if p.cfg.BatchTimeout > 0 {
		batchOpts = append(batchOpts, sdktrace.WithBatchTimeout(p.cfg.BatchTimeout))
	}
	if p.cfg.BatchSize > 0 {
		batchOpts = append(batchOpts, sdktrace.WithMaxExportBatchSize(p.cfg.BatchSize))
	}

	p.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter, batchOpts...),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(p.tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	p.tracer = p.tracerProvider.Tracer(p.cfg.ServiceName)

	return nil
}

func (p *Provider) initMetrics(ctx context.Context, res *resource.Resource) error {
	endpoint := parseEndpoint(p.cfg.Endpoint)

	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithHeaders(p.cfg.Headers),
		otlpmetrichttp.WithURLPath("/api/v2/otlp/v1/metrics"),
	}

	if p.cfg.Insecure {
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}

	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return err
	}

	readerOpts := []sdkmetric.PeriodicReaderOption{}
	if p.cfg.BatchTimeout > 0 {
		readerOpts = append(readerOpts, sdkmetric.WithInterval(p.cfg.BatchTimeout))
	}

	p.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, readerOpts...)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(p.meterProvider)

	p.meter = p.meterProvider.Meter(p.cfg.ServiceName)

	return nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "dynatrace"
}

// Meter returns the metric meter.
func (p *Provider) Meter() observops.Meter {
	return &dtMeter{meter: p.meter}
}

// Tracer returns the tracer.
func (p *Provider) Tracer() observops.Tracer {
	return &dtTracer{tracer: p.tracer}
}

// Logger returns the structured logger.
func (p *Provider) Logger() observops.Logger {
	return p.logger
}

// SlogHandler returns an slog.Handler that integrates with this provider.
// The handler sends logs to Dynatrace while optionally also forwarding
// to a local handler for console/file output.
func (p *Provider) SlogHandler(opts ...observops.SlogOption) slog.Handler {
	cfg := observops.ApplySlogOptions(opts...)
	if cfg.Disabled {
		return observops.NoopSlogHandler()
	}
	return observops.NewLoggerSlogHandler(p.logger, cfg)
}

// Shutdown gracefully shuts down the provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.shutdown {
		return observops.ErrShutdown
	}
	p.shutdown = true

	var errs []error

	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// ForceFlush forces any buffered telemetry to be exported.
func (p *Provider) ForceFlush(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.shutdown {
		return observops.ErrShutdown
	}

	var errs []error

	if p.tracerProvider != nil {
		if err := p.tracerProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if p.meterProvider != nil {
		if err := p.meterProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// dtMeter wraps an OpenTelemetry meter.
type dtMeter struct {
	meter metric.Meter
}

func (m *dtMeter) Counter(name string, opts ...observops.MetricOption) (observops.Counter, error) {
	desc := observops.GetDescription(opts...)
	unit := observops.GetUnit(opts...)

	counterOpts := []metric.Float64CounterOption{}
	if desc != "" {
		counterOpts = append(counterOpts, metric.WithDescription(desc))
	}
	if unit != "" {
		counterOpts = append(counterOpts, metric.WithUnit(unit))
	}

	counter, err := m.meter.Float64Counter(name, counterOpts...)
	if err != nil {
		return nil, err
	}
	return &dtCounter{counter: counter}, nil
}

func (m *dtMeter) UpDownCounter(name string, opts ...observops.MetricOption) (observops.UpDownCounter, error) {
	desc := observops.GetDescription(opts...)
	unit := observops.GetUnit(opts...)

	counterOpts := []metric.Float64UpDownCounterOption{}
	if desc != "" {
		counterOpts = append(counterOpts, metric.WithDescription(desc))
	}
	if unit != "" {
		counterOpts = append(counterOpts, metric.WithUnit(unit))
	}

	counter, err := m.meter.Float64UpDownCounter(name, counterOpts...)
	if err != nil {
		return nil, err
	}
	return &dtUpDownCounter{counter: counter}, nil
}

func (m *dtMeter) Histogram(name string, opts ...observops.MetricOption) (observops.Histogram, error) {
	desc := observops.GetDescription(opts...)
	unit := observops.GetUnit(opts...)

	histOpts := []metric.Float64HistogramOption{}
	if desc != "" {
		histOpts = append(histOpts, metric.WithDescription(desc))
	}
	if unit != "" {
		histOpts = append(histOpts, metric.WithUnit(unit))
	}

	hist, err := m.meter.Float64Histogram(name, histOpts...)
	if err != nil {
		return nil, err
	}
	return &dtHistogram{histogram: hist}, nil
}

func (m *dtMeter) Gauge(name string, opts ...observops.MetricOption) (observops.Gauge, error) {
	desc := observops.GetDescription(opts...)
	unit := observops.GetUnit(opts...)

	gaugeOpts := []metric.Float64GaugeOption{}
	if desc != "" {
		gaugeOpts = append(gaugeOpts, metric.WithDescription(desc))
	}
	if unit != "" {
		gaugeOpts = append(gaugeOpts, metric.WithUnit(unit))
	}

	gauge, err := m.meter.Float64Gauge(name, gaugeOpts...)
	if err != nil {
		return nil, err
	}
	return &dtGauge{gauge: gauge}, nil
}

type dtCounter struct {
	counter metric.Float64Counter
}

func (c *dtCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

type dtUpDownCounter struct {
	counter metric.Float64UpDownCounter
}

func (c *dtUpDownCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

type dtHistogram struct {
	histogram metric.Float64Histogram
}

func (h *dtHistogram) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}

type dtGauge struct {
	gauge metric.Float64Gauge
}

func (g *dtGauge) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	g.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// dtTracer wraps an OpenTelemetry tracer.
type dtTracer struct {
	tracer trace.Tracer
}

func (t *dtTracer) Start(ctx context.Context, name string, opts ...observops.SpanOption) (context.Context, observops.Span) {
	spanOpts := []trace.SpanStartOption{}

	kind := observops.GetSpanKind(opts...)
	spanOpts = append(spanOpts, trace.WithSpanKind(toOtelSpanKind(kind)))

	attrs := observops.GetSpanAttributes(opts...)
	if len(attrs) > 0 {
		spanOpts = append(spanOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}

	links := observops.GetSpanLinks(opts...)
	if len(links) > 0 {
		otelLinks := make([]trace.Link, len(links))
		for i, link := range links {
			otelLinks[i] = trace.Link{
				SpanContext: toOtelSpanContext(link),
			}
		}
		spanOpts = append(spanOpts, trace.WithLinks(otelLinks...))
	}

	ctx, span := t.tracer.Start(ctx, name, spanOpts...)
	return ctx, &dtSpan{span: span}
}

func (t *dtTracer) SpanFromContext(ctx context.Context) observops.Span {
	span := trace.SpanFromContext(ctx)
	return &dtSpan{span: span}
}

type dtSpan struct {
	span trace.Span
}

func (s *dtSpan) End(opts ...observops.SpanEndOption) {
	endOpts := []trace.SpanEndOption{}

	ts := observops.GetEndTimestamp(opts...)
	if ts != nil {
		endOpts = append(endOpts, trace.WithTimestamp(*ts))
	}

	s.span.End(endOpts...)
}

func (s *dtSpan) SetAttributes(attrs ...observops.KeyValue) {
	s.span.SetAttributes(toOtelAttributes(attrs)...)
}

func (s *dtSpan) SetStatus(code observops.StatusCode, description string) {
	s.span.SetStatus(toOtelStatusCode(code), description)
}

func (s *dtSpan) RecordError(err error, opts ...observops.EventOption) {
	eventOpts := []trace.EventOption{}

	attrs := observops.GetEventAttributes(opts...)
	if len(attrs) > 0 {
		eventOpts = append(eventOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}

	ts := observops.GetEventTimestamp(opts...)
	if ts != nil {
		eventOpts = append(eventOpts, trace.WithTimestamp(*ts))
	}

	s.span.RecordError(err, eventOpts...)
}

func (s *dtSpan) AddEvent(name string, opts ...observops.EventOption) {
	eventOpts := []trace.EventOption{}

	attrs := observops.GetEventAttributes(opts...)
	if len(attrs) > 0 {
		eventOpts = append(eventOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}

	ts := observops.GetEventTimestamp(opts...)
	if ts != nil {
		eventOpts = append(eventOpts, trace.WithTimestamp(*ts))
	}

	s.span.AddEvent(name, eventOpts...)
}

func (s *dtSpan) SpanContext() observops.SpanContext {
	sc := s.span.SpanContext()
	return observops.SpanContext{
		TraceID:    sc.TraceID().String(),
		SpanID:     sc.SpanID().String(),
		TraceFlags: byte(sc.TraceFlags()),
		Remote:     sc.IsRemote(),
	}
}

func (s *dtSpan) IsRecording() bool {
	return s.span.IsRecording()
}

// dtLogger provides structured logging with trace correlation.
type dtLogger struct {
	tracer trace.Tracer
}

func (l *dtLogger) log(ctx context.Context, level observops.SeverityLevel, msg string, attrs ...observops.LogAttribute) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		eventAttrs := make([]attribute.KeyValue, 0, len(attrs)+2)
		eventAttrs = append(eventAttrs, attribute.String("log.level", levelName(level)))
		eventAttrs = append(eventAttrs, attribute.String("log.message", msg))
		for _, attr := range attrs {
			eventAttrs = append(eventAttrs, toOtelAttribute(attr.Key, attr.Value))
		}
		span.AddEvent("log", trace.WithAttributes(eventAttrs...))
	}
}

func (l *dtLogger) Debug(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityDebug, msg, attrs...)
}

func (l *dtLogger) Info(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityInfo, msg, attrs...)
}

func (l *dtLogger) Warn(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityWarn, msg, attrs...)
}

func (l *dtLogger) Error(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityError, msg, attrs...)
}

func levelName(level observops.SeverityLevel) string {
	switch level {
	case observops.SeverityDebug:
		return "DEBUG"
	case observops.SeverityInfo:
		return "INFO"
	case observops.SeverityWarn:
		return "WARN"
	case observops.SeverityError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Conversion helpers

func toOtelAttributes(kvs []observops.KeyValue) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, len(kvs))
	for i, kv := range kvs {
		attrs[i] = toOtelAttribute(kv.Key, kv.Value)
	}
	return attrs
}

func toOtelAttribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	case time.Duration:
		return attribute.Int64(key, v.Milliseconds())
	default:
		return attribute.String(key, "")
	}
}

func toOtelSpanKind(kind observops.SpanKind) trace.SpanKind {
	switch kind {
	case observops.SpanKindServer:
		return trace.SpanKindServer
	case observops.SpanKindClient:
		return trace.SpanKindClient
	case observops.SpanKindProducer:
		return trace.SpanKindProducer
	case observops.SpanKindConsumer:
		return trace.SpanKindConsumer
	default:
		return trace.SpanKindInternal
	}
}

func toOtelStatusCode(code observops.StatusCode) codes.Code {
	switch code {
	case observops.StatusCodeOK:
		return codes.Ok
	case observops.StatusCodeError:
		return codes.Error
	default:
		return codes.Unset
	}
}

func toOtelSpanContext(sc observops.SpanContext) trace.SpanContext {
	traceID, _ := trace.TraceIDFromHex(sc.TraceID)
	spanID, _ := trace.SpanIDFromHex(sc.SpanID)
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.TraceFlags(sc.TraceFlags),
		Remote:     sc.Remote,
	})
}
