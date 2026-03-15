// Package omniobserve provides a unified entry point for observability,
// combining metrics, traces, and logs with HTTP middleware and context utilities.
package omniobserve

import (
	"context"
	"log/slog"
	"sync"

	"github.com/plexusone/omniobserve/observops"
)

// Observability provides a unified interface for metrics, traces, and logs.
// It wraps an observops.Provider with convenience methods and middleware support.
type Observability struct {
	provider    observops.Provider
	logger      *slog.Logger
	config      *Config
	mu          sync.RWMutex
	initialized bool
}

// New creates a new Observability instance with the specified provider and options.
func New(providerName string, opts ...Option) (*Observability, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Build provider options
	providerOpts := []observops.ClientOption{
		observops.WithServiceName(cfg.ServiceName),
	}
	if cfg.ServiceVersion != "" {
		providerOpts = append(providerOpts, observops.WithServiceVersion(cfg.ServiceVersion))
	}
	if cfg.Endpoint != "" {
		providerOpts = append(providerOpts, observops.WithEndpoint(cfg.Endpoint))
	}
	if cfg.APIKey != "" {
		providerOpts = append(providerOpts, observops.WithAPIKey(cfg.APIKey))
	}
	if cfg.Insecure {
		providerOpts = append(providerOpts, observops.WithInsecure())
	}
	if cfg.Disabled {
		providerOpts = append(providerOpts, observops.WithDisabled())
	}
	if cfg.Debug {
		providerOpts = append(providerOpts, observops.WithDebug())
	}
	for k, v := range cfg.Headers {
		providerOpts = append(providerOpts, observops.WithHeaders(map[string]string{k: v}))
	}
	providerOpts = append(providerOpts, cfg.ProviderOptions...)

	// Open the provider
	provider, err := observops.Open(providerName, providerOpts...)
	if err != nil {
		return nil, err
	}

	o := &Observability{
		provider:    provider,
		config:      cfg,
		initialized: true,
	}

	// Create slog handler
	slogOpts := []observops.SlogOption{}
	if cfg.LocalHandler != nil {
		slogOpts = append(slogOpts, observops.WithSlogLocalHandler(cfg.LocalHandler))
	}
	if cfg.SlogLevel != 0 {
		slogOpts = append(slogOpts, observops.WithSlogRemoteLevel(int(cfg.SlogLevel)))
	}

	handler := provider.SlogHandler(slogOpts...)
	o.logger = slog.New(handler)

	return o, nil
}

// NewWithProvider creates a new Observability instance with an existing provider.
func NewWithProvider(provider observops.Provider, opts ...Option) *Observability {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	o := &Observability{
		provider:    provider,
		config:      cfg,
		initialized: true,
	}

	slogOpts := []observops.SlogOption{}
	if cfg.LocalHandler != nil {
		slogOpts = append(slogOpts, observops.WithSlogLocalHandler(cfg.LocalHandler))
	}
	if cfg.SlogLevel != 0 {
		slogOpts = append(slogOpts, observops.WithSlogRemoteLevel(int(cfg.SlogLevel)))
	}

	handler := provider.SlogHandler(slogOpts...)
	o.logger = slog.New(handler)

	return o
}

// Provider returns the underlying observops.Provider.
func (o *Observability) Provider() observops.Provider {
	return o.provider
}

// Logger returns the configured slog.Logger with trace context support.
func (o *Observability) Logger() *slog.Logger {
	return o.logger
}

// Tracer returns the tracer for creating spans.
func (o *Observability) Tracer() observops.Tracer {
	return o.provider.Tracer()
}

// Meter returns the meter for creating metrics.
func (o *Observability) Meter() observops.Meter {
	return o.provider.Meter()
}

// StartSpan starts a new span with the given name and options.
// Returns the span and a context containing the span.
func (o *Observability) StartSpan(ctx context.Context, name string, opts ...observops.SpanOption) (context.Context, observops.Span) {
	return o.provider.Tracer().Start(ctx, name, opts...)
}

// LoggerFromContext returns a logger that includes trace context from the span in ctx.
func (o *Observability) LoggerFromContext(ctx context.Context) *slog.Logger {
	span := o.provider.Tracer().SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return o.logger
	}

	sc := span.SpanContext()
	return o.logger.With(
		slog.String("trace_id", sc.TraceID),
		slog.String("span_id", sc.SpanID),
	)
}

// Shutdown gracefully shuts down the observability provider.
func (o *Observability) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.initialized {
		return nil
	}
	o.initialized = false

	return o.provider.Shutdown(ctx)
}

// ForceFlush forces an immediate flush of all buffered telemetry data.
func (o *Observability) ForceFlush(ctx context.Context) error {
	return o.provider.ForceFlush(ctx)
}

// Counter creates or retrieves a counter metric.
func (o *Observability) Counter(name string, opts ...observops.MetricOption) (observops.Counter, error) {
	return o.provider.Meter().Counter(name, opts...)
}

// Histogram creates or retrieves a histogram metric.
func (o *Observability) Histogram(name string, opts ...observops.MetricOption) (observops.Histogram, error) {
	return o.provider.Meter().Histogram(name, opts...)
}

// Gauge creates or retrieves a gauge metric.
func (o *Observability) Gauge(name string, opts ...observops.MetricOption) (observops.Gauge, error) {
	return o.provider.Meter().Gauge(name, opts...)
}

// UpDownCounter creates or retrieves an up-down counter metric.
func (o *Observability) UpDownCounter(name string, opts ...observops.MetricOption) (observops.UpDownCounter, error) {
	return o.provider.Meter().UpDownCounter(name, opts...)
}
