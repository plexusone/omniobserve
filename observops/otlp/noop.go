package otlp

import (
	"context"
	"log/slog"

	"github.com/plexusone/omniobserve/observops"
)

// noopProvider is a provider that does nothing.
// Used when observability is disabled.
type noopProvider struct{}

func newNoopProvider() observops.Provider {
	return &noopProvider{}
}

func (p *noopProvider) Name() string { return "otlp-noop" }

func (p *noopProvider) Meter() observops.Meter { return &noopMeter{} }

func (p *noopProvider) Tracer() observops.Tracer { return &noopTracer{} }

func (p *noopProvider) Logger() observops.Logger { return &noopLogger{} }

func (p *noopProvider) SlogHandler(opts ...observops.SlogOption) slog.Handler {
	return observops.NoopSlogHandler()
}

func (p *noopProvider) Shutdown(ctx context.Context) error { return nil }

func (p *noopProvider) ForceFlush(ctx context.Context) error { return nil }

// noopMeter is a meter that does nothing.
type noopMeter struct{}

func (m *noopMeter) Counter(name string, opts ...observops.MetricOption) (observops.Counter, error) {
	return &noopCounter{}, nil
}

func (m *noopMeter) UpDownCounter(name string, opts ...observops.MetricOption) (observops.UpDownCounter, error) {
	return &noopUpDownCounter{}, nil
}

func (m *noopMeter) Histogram(name string, opts ...observops.MetricOption) (observops.Histogram, error) {
	return &noopHistogram{}, nil
}

func (m *noopMeter) Gauge(name string, opts ...observops.MetricOption) (observops.Gauge, error) {
	return &noopGauge{}, nil
}

type noopCounter struct{}

func (c *noopCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {}

type noopUpDownCounter struct{}

func (c *noopUpDownCounter) Add(ctx context.Context, value float64, opts ...observops.RecordOption) {}

type noopHistogram struct{}

func (h *noopHistogram) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {}

type noopGauge struct{}

func (g *noopGauge) Record(ctx context.Context, value float64, opts ...observops.RecordOption) {}

// noopTracer is a tracer that does nothing.
type noopTracer struct{}

func (t *noopTracer) Start(ctx context.Context, name string, opts ...observops.SpanOption) (context.Context, observops.Span) {
	return ctx, &noopSpan{}
}

func (t *noopTracer) SpanFromContext(ctx context.Context) observops.Span {
	return &noopSpan{}
}

type noopSpan struct{}

func (s *noopSpan) End(opts ...observops.SpanEndOption) {}

func (s *noopSpan) SetAttributes(attrs ...observops.KeyValue) {}

func (s *noopSpan) SetStatus(code observops.StatusCode, description string) {}

func (s *noopSpan) RecordError(err error, opts ...observops.EventOption) {}

func (s *noopSpan) AddEvent(name string, opts ...observops.EventOption) {}

func (s *noopSpan) SpanContext() observops.SpanContext {
	return observops.SpanContext{}
}

func (s *noopSpan) IsRecording() bool { return false }

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (l *noopLogger) Debug(ctx context.Context, msg string, attrs ...observops.LogAttribute) {}

func (l *noopLogger) Info(ctx context.Context, msg string, attrs ...observops.LogAttribute) {}

func (l *noopLogger) Warn(ctx context.Context, msg string, attrs ...observops.LogAttribute) {}

func (l *noopLogger) Error(ctx context.Context, msg string, attrs ...observops.LogAttribute) {}
