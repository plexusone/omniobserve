// Package newrelic provides a New Relic observability provider for observops.
// It uses OpenTelemetry Protocol (OTLP) to send telemetry data to New Relic.
//
// # Usage
//
//	import (
//		"github.com/plexusone/omniobserve/observops"
//		_ "github.com/plexusone/omniobserve/observops/newrelic"
//	)
//
//	provider, err := observops.Open("newrelic",
//		observops.WithAPIKey("YOUR_NEW_RELIC_LICENSE_KEY"),
//		observops.WithServiceName("my-service"),
//	)
//
// # Configuration
//
// Required:
//   - API Key: Your New Relic license key (Ingest - License type)
//   - Service Name: The name of your service
//
// Optional:
//   - Endpoint: Defaults to otlp.nr-data.net:4317 (US) or otlp.eu01.nr-data.net:4317 (EU)
//
// # Environment Variables
//
// You can also configure via environment variables:
//   - NEW_RELIC_LICENSE_KEY: Your New Relic license key
//   - OTEL_SERVICE_NAME: Service name
//   - OTEL_EXPORTER_OTLP_ENDPOINT: Custom endpoint (optional)
//
// # Regions
//
// Use WithNewRelicRegion to specify your account region:
//   - "us" (default): US datacenter
//   - "eu": EU datacenter
package newrelic

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
	// DefaultUSEndpoint is the default New Relic OTLP endpoint for US accounts.
	DefaultUSEndpoint = "otlp.nr-data.net:4317"
	// DefaultEUEndpoint is the default New Relic OTLP endpoint for EU accounts.
	DefaultEUEndpoint = "otlp.eu01.nr-data.net:4317"
)

// Region represents a New Relic datacenter region.
type Region string

const (
	RegionUS Region = "us"
	RegionEU Region = "eu"
)

func init() {
	observops.Register("newrelic", New)
	observops.RegisterInfo(observops.ProviderInfo{
		Name:        "newrelic",
		Description: "New Relic observability platform",
		Website:     "https://newrelic.com",
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

// Config holds New Relic-specific configuration.
type Config struct {
	*observops.Config
	Region Region
}

// Option configures New Relic-specific settings.
type Option func(*Config)

// WithNewRelicRegion sets the New Relic datacenter region.
func WithNewRelicRegion(region Region) observops.ClientOption {
	return func(c *observops.Config) {
		// Store region in headers as a marker
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers["x-nr-region"] = string(region)
	}
}

// Provider implements observops.Provider for New Relic.
type Provider struct {
	cfg            *observops.Config
	region         Region
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	logger         *nrLogger
	shutdown       bool
	mu             sync.RWMutex
}

// New creates a new New Relic provider.
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

	// Determine region
	region := RegionUS
	if cfg.Headers != nil {
		if r, ok := cfg.Headers["x-nr-region"]; ok {
			region = Region(r)
			delete(cfg.Headers, "x-nr-region")
		}
	}

	// Set default endpoint based on region
	if cfg.Endpoint == "" {
		switch region {
		case RegionEU:
			cfg.Endpoint = DefaultEUEndpoint
		default:
			cfg.Endpoint = DefaultUSEndpoint
		}
	}

	// Set up headers with API key
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	cfg.Headers["api-key"] = cfg.APIKey

	p := &Provider{
		cfg:    cfg,
		region: region,
	}

	ctx := context.Background()

	// Build resource
	res := p.buildResource()

	// Initialize trace exporter
	if err := p.initTracing(ctx, res); err != nil {
		return nil, observops.WrapError("newrelic", "initTracing", err)
	}

	// Initialize metric exporter
	if err := p.initMetrics(ctx, res); err != nil {
		_ = p.tracerProvider.Shutdown(ctx)
		return nil, observops.WrapError("newrelic", "initMetrics", err)
	}

	// Initialize logger
	p.logger = &nrLogger{
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

func (p *Provider) initTracing(ctx context.Context, res *resource.Resource) error {
	traceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(p.cfg.Endpoint),
		otlptracegrpc.WithHeaders(p.cfg.Headers),
	}

	if p.cfg.Insecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
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
		otlpmetricgrpc.WithHeaders(p.cfg.Headers),
	}

	if p.cfg.Insecure {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
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
	return "newrelic"
}

// Meter returns the metric meter.
func (p *Provider) Meter() observops.Meter {
	return &nrMeter{meter: p.meter}
}

// Tracer returns the tracer.
func (p *Provider) Tracer() observops.Tracer {
	return &nrTracer{tracer: p.tracer}
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

// nrMeter wraps the OpenTelemetry meter for New Relic.
type nrMeter struct {
	meter metric.Meter
}

func (m *nrMeter) Counter(name string, opts ...observops.MetricOption) (observops.Counter, error) {
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
	return &nrCounter{counter: counter}, nil
}

func (m *nrMeter) UpDownCounter(name string, opts ...observops.MetricOption) (observops.UpDownCounter, error) {
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
	return &nrUpDownCounter{counter: counter}, nil
}

func (m *nrMeter) Histogram(name string, opts ...observops.MetricOption) (observops.Histogram, error) {
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
	return &nrHistogram{histogram: hist}, nil
}

func (m *nrMeter) Gauge(name string, opts ...observops.MetricOption) (observops.Gauge, error) {
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
	return &nrGauge{gauge: gauge}, nil
}

type nrCounter struct {
	counter metric.Float64Counter
}

func (c *nrCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

type nrUpDownCounter struct {
	counter metric.Float64UpDownCounter
}

func (c *nrUpDownCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

type nrHistogram struct {
	histogram metric.Float64Histogram
}

func (h *nrHistogram) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}

type nrGauge struct {
	gauge metric.Float64Gauge
}

func (g *nrGauge) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {
	attrs := toOtelAttributes(observops.GetAttributes(opts...))
	g.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// nrTracer wraps the OpenTelemetry tracer for New Relic.
type nrTracer struct {
	tracer trace.Tracer
}

func (t *nrTracer) Start(ctx context.Context, name string, opts ...observops.SpanOption) (context.Context, observops.Span) {
	spanOpts := []trace.SpanStartOption{}

	kind := observops.GetSpanKind(opts...)
	spanOpts = append(spanOpts, trace.WithSpanKind(toOtelSpanKind(kind)))

	attrs := observops.GetSpanAttributes(opts...)
	if len(attrs) > 0 {
		spanOpts = append(spanOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}

	ctx, span := t.tracer.Start(ctx, name, spanOpts...)
	return ctx, &nrSpan{span: span}
}

func (t *nrTracer) SpanFromContext(ctx context.Context) observops.Span {
	span := trace.SpanFromContext(ctx)
	return &nrSpan{span: span}
}

type nrSpan struct {
	span trace.Span
}

func (s *nrSpan) End(opts ...observops.SpanEndOption) {
	endOpts := []trace.SpanEndOption{}
	ts := observops.GetEndTimestamp(opts...)
	if ts != nil {
		endOpts = append(endOpts, trace.WithTimestamp(*ts))
	}
	s.span.End(endOpts...)
}

func (s *nrSpan) SetAttributes(attrs ...observops.KeyValue) {
	s.span.SetAttributes(toOtelAttributes(attrs)...)
}

func (s *nrSpan) SetStatus(code observops.StatusCode, description string) {
	s.span.SetStatus(toOtelStatusCode(code), description)
}

func (s *nrSpan) RecordError(err error, opts ...observops.EventOption) {
	eventOpts := []trace.EventOption{}
	attrs := observops.GetEventAttributes(opts...)
	if len(attrs) > 0 {
		eventOpts = append(eventOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}
	s.span.RecordError(err, eventOpts...)
}

func (s *nrSpan) AddEvent(name string, opts ...observops.EventOption) {
	eventOpts := []trace.EventOption{}
	attrs := observops.GetEventAttributes(opts...)
	if len(attrs) > 0 {
		eventOpts = append(eventOpts, trace.WithAttributes(toOtelAttributes(attrs)...))
	}
	s.span.AddEvent(name, eventOpts...)
}

func (s *nrSpan) SpanContext() observops.SpanContext {
	sc := s.span.SpanContext()
	return observops.SpanContext{
		TraceID:    sc.TraceID().String(),
		SpanID:     sc.SpanID().String(),
		TraceFlags: byte(sc.TraceFlags()),
		Remote:     sc.IsRemote(),
	}
}

func (s *nrSpan) IsRecording() bool {
	return s.span.IsRecording()
}

// nrLogger provides structured logging with trace correlation for New Relic.
type nrLogger struct {
	tracer trace.Tracer
}

func (l *nrLogger) log(ctx context.Context, level observops.SeverityLevel, msg string, attrs ...observops.LogAttribute) {
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

func (l *nrLogger) Debug(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityDebug, msg, attrs...)
}

func (l *nrLogger) Info(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityInfo, msg, attrs...)
}

func (l *nrLogger) Warn(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
	l.log(ctx, observops.SeverityWarn, msg, attrs...)
}

func (l *nrLogger) Error(ctx context.Context, msg string, attrs ...observops.LogAttribute) {
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
