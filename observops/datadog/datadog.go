// Package datadog provides a Datadog observability provider for observops.
// It uses OpenTelemetry Protocol (OTLP) to send telemetry data to Datadog
// via the Datadog Agent or directly to Datadog's OTLP endpoints.
//
// # Usage
//
//	import (
//		"github.com/plexusone/omniobserve/observops"
//		_ "github.com/plexusone/omniobserve/observops/datadog"
//	)
//
//	provider, err := observops.Open("datadog",
//		observops.WithEndpoint("localhost:4317"), // Datadog Agent OTLP endpoint
//		observops.WithServiceName("my-service"),
//	)
//
// # Configuration
//
// When using the Datadog Agent (recommended):
//   - Endpoint: localhost:4317 (default) for gRPC or localhost:4318 for HTTP
//   - Ensure the Datadog Agent has OTLP ingestion enabled
//
// When sending directly to Datadog:
//   - Endpoint: Use Datadog's intake endpoint for your region
//   - API Key: Your Datadog API key
//
// # Environment Variables
//
// Standard OpenTelemetry environment variables are respected:
//   - OTEL_EXPORTER_OTLP_ENDPOINT
//   - OTEL_SERVICE_NAME
//   - DD_SITE (for direct ingestion: datadoghq.com, datadoghq.eu, etc.)
//   - DD_API_KEY (for direct ingestion)
//
// # Datadog Agent Configuration
//
// Enable OTLP ingestion in your datadog.yaml:
//
//	otlp_config:
//	  receiver:
//	    protocols:
//	      grpc:
//	        endpoint: 0.0.0.0:4317
//	      http:
//	        endpoint: 0.0.0.0:4318
package datadog

import (
	"context"
	"sync"
	"time"

	"github.com/plexusone/omniobserve/observops"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// DefaultAgentEndpoint is the default Datadog Agent OTLP endpoint.
	DefaultAgentEndpoint = "localhost:4317"
)

// Site represents a Datadog datacenter site.
type Site string

const (
	SiteUS1 Site = "datadoghq.com"
	SiteUS3 Site = "us3.datadoghq.com"
	SiteUS5 Site = "us5.datadoghq.com"
	SiteEU1 Site = "datadoghq.eu"
	SiteAP1 Site = "ap1.datadoghq.com"
)

func init() {
	observops.Register("datadog", New)
	observops.RegisterInfo(observops.ProviderInfo{
		Name:        "datadog",
		Description: "Datadog observability platform",
		Website:     "https://www.datadoghq.com",
		OpenSource:  false,
		SelfHosted:  false,
		Capabilities: []observops.Capability{
			observops.CapabilityMetrics,
			observops.CapabilityTraces,
			observops.CapabilityLogs,
			observops.CapabilityBatching,
			observops.CapabilitySampling,
		},
	})
}

// WithDatadogSite sets the Datadog site for direct ingestion.
func WithDatadogSite(site Site) observops.ClientOption {
	return func(c *observops.Config) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers["x-dd-site"] = string(site)
	}
}

// WithDatadogEnv sets the Datadog environment tag.
func WithDatadogEnv(env string) observops.ClientOption {
	return func(c *observops.Config) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers["x-dd-env"] = env
	}
}

// WithDatadogVersion sets the Datadog version tag.
func WithDatadogVersion(version string) observops.ClientOption {
	return func(c *observops.Config) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers["x-dd-version"] = version
	}
}

// Provider implements observops.Provider for Datadog.
type Provider struct {
	cfg            *observops.Config
	site           Site
	env            string
	version        string
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	logger         *ddLogger
	shutdown       bool
	mu             sync.RWMutex
}

// New creates a new Datadog provider.
func New(opts ...observops.ClientOption) (observops.Provider, error) {
	cfg := observops.ApplyOptions(opts...)

	if cfg.Disabled {
		return newNoopProvider(), nil
	}

	if cfg.ServiceName == "" {
		return nil, observops.ErrMissingServiceName
	}

	// Extract Datadog-specific config from headers
	var site Site
	var env, version string
	if cfg.Headers != nil {
		if s, ok := cfg.Headers["x-dd-site"]; ok {
			site = Site(s)
			delete(cfg.Headers, "x-dd-site")
		}
		if e, ok := cfg.Headers["x-dd-env"]; ok {
			env = e
			delete(cfg.Headers, "x-dd-env")
		}
		if v, ok := cfg.Headers["x-dd-version"]; ok {
			version = v
			delete(cfg.Headers, "x-dd-version")
		}
	}

	// Set default endpoint (Datadog Agent)
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultAgentEndpoint
	}

	// For direct ingestion, set up API key header
	if cfg.APIKey != "" {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		cfg.Headers["DD-API-KEY"] = cfg.APIKey
	}

	p := &Provider{
		cfg:     cfg,
		site:    site,
		env:     env,
		version: version,
	}

	ctx := context.Background()

	// Build resource with Datadog-specific attributes
	res := p.buildResource()

	// Initialize trace exporter
	if err := p.initTracing(ctx, res); err != nil {
		return nil, observops.WrapError("datadog", "initTracing", err)
	}

	// Initialize metric exporter
	if err := p.initMetrics(ctx, res); err != nil {
		_ = p.tracerProvider.Shutdown(ctx)
		return nil, observops.WrapError("datadog", "initMetrics", err)
	}

	// Initialize logger
	p.logger = &ddLogger{
		tracer: p.tracer,
	}

	return p, nil
}

func (p *Provider) buildResource() *resource.Resource {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(p.cfg.ServiceName),
	}

	// Service version
	if p.cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersionKey.String(p.cfg.ServiceVersion))
	} else if p.version != "" {
		attrs = append(attrs, semconv.ServiceVersionKey.String(p.version))
	}

	// Datadog-specific attributes
	if p.env != "" {
		attrs = append(attrs, attribute.String("deployment.environment", p.env))
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

func (p *Provider) initTracing(ctx context.Context, res *resource.Resource) error {
	traceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(p.cfg.Endpoint),
	}

	if p.cfg.Insecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
	}

	if len(p.cfg.Headers) > 0 {
		traceOpts = append(traceOpts, otlptracegrpc.WithHeaders(p.cfg.Headers))
	}

	traceExporter, err := otlptracegrpc.New(ctx, traceOpts...)
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
	metricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(p.cfg.Endpoint),
	}

	if p.cfg.Insecure {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	}

	if len(p.cfg.Headers) > 0 {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithHeaders(p.cfg.Headers))
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, metricOpts...)
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
	return "datadog"
}

// Meter returns the metric meter.
func (p *Provider) Meter() observops.Meter {
	return &ddMeter{meter: p.meter}
}

// Tracer returns the tracer.
func (p *Provider) Tracer() observops.Tracer {
	return &ddTracer{tracer: p.tracer}
}

// Logger returns the structured logger.
func (p *Provider) Logger() observops.Logger {
	return p.logger
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

// ddMeter wraps the OpenTelemetry meter for Datadog.
type ddMeter struct {
	meter metric.Meter
}

func (m *ddMeter) Counter(name string, opts ...observops.MetricOption) (observops.Counter, error) {
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
	return &ddCounter{counter: counter}, nil
}

func (m *ddMeter) UpDownCounter(name string, opts ...observops.MetricOption) (observops.UpDownCounter, error) {
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
	return &ddUpDownCounter{counter: counter}, nil
}

func (m *ddMeter) Histogram(name string, opts ...observops.MetricOption) (observops.Histogram, error) {
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
	return &ddHistogram{histogram: hist}, nil
}

func (m *ddMeter) Gauge(name string, opts ...observops.MetricOption) (observops.Gauge, error) {
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
	return &ddGauge{gauge: gauge}, nil
}

type ddCounter struct {
	counter metric.Float64Counter
}

func (c *ddCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

type ddUpDownCounter struct {
	counter metric.Float64UpDownCounter
}

func (c *ddUpDownCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

type ddHistogram struct {
	histogram metric.Float64Histogram
}

func (h *ddHistogram) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}

type ddGauge struct {
	gauge metric.Float64Gauge
}

func (g *ddGauge) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	g.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// ddTracer wraps the OpenTelemetry tracer for Datadog.
type ddTracer struct {
	tracer trace.Tracer
}

func (t *ddTracer) Start(ctx context.Context, name string, opts ...observops.SpanOption) (context.Context, observops.Span) {
	spanOpts := []trace.SpanStartOption{}

	kind := observops.GetSpanKind(opts...)
	spanOpts = append(spanOpts, trace.WithSpanKind(toOtelSpanKind(kind)))

	attrs := observops.GetSpanAttributes(opts...)
	if len(attrs) > 0 {
		spanOpts = append(spanOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}

	ctx, span := t.tracer.Start(ctx, name, spanOpts...)
	return ctx, &ddSpan{span: span}
}

func (t *ddTracer) SpanFromContext(ctx context.Context) observops.Span {
	span := trace.SpanFromContext(ctx)
	return &ddSpan{span: span}
}

type ddSpan struct {
	span trace.Span
}

func (s *ddSpan) End(opts ...observops.SpanEndOption) {
	endOpts := []trace.SpanEndOption{}
	ts := observops.GetEndTimestamp(opts...)
	if ts != nil {
		endOpts = append(endOpts, trace.WithTimestamp(*ts))
	}
	s.span.End(endOpts...)
}

func (s *ddSpan) SetAttributes(attrs ...observops.KeyValue) {
	s.span.SetAttributes(toOtelAttributes(attrs)...)
}

func (s *ddSpan) SetStatus(code observops.StatusCode, description string) {
	s.span.SetStatus(toOtelStatusCode(code), description)
}

func (s *ddSpan) RecordError(err error, opts ...observops.EventOption) {
	eventOpts := []trace.EventOption{}
	attrs := observops.GetEventAttributes(opts...)
	if len(attrs) > 0 {
		eventOpts = append(eventOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}
	s.span.RecordError(err, eventOpts...)
}

func (s *ddSpan) AddEvent(name string, opts ...observops.EventOption) {
	eventOpts := []trace.EventOption{}
	attrs := observops.GetEventAttributes(opts...)
	if len(attrs) > 0 {
		eventOpts = append(eventOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}
	s.span.AddEvent(name, eventOpts...)
}

func (s *ddSpan) SpanContext() observops.SpanContext {
	sc := s.span.SpanContext()
	return observops.SpanContext{
		TraceID:    sc.TraceID().String(),
		SpanID:     sc.SpanID().String(),
		TraceFlags: byte(sc.TraceFlags()),
		Remote:     sc.IsRemote(),
	}
}

func (s *ddSpan) IsRecording() bool {
	return s.span.IsRecording()
}

// ddLogger provides structured logging with trace correlation for Datadog.
type ddLogger struct {
	tracer trace.Tracer
}

func (l *ddLogger) log(ctx context.Context, level observops.SeverityLevel, msg string, attrs ...observops.LogAttribute) {
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

func (l *ddLogger) Debug(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityDebug, msg, attrs...)
}

func (l *ddLogger) Info(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityInfo, msg, attrs...)
}

func (l *ddLogger) Warn(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityWarn, msg, attrs...)
}

func (l *ddLogger) Error(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
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
