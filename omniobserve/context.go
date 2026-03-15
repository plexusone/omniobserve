package omniobserve

import (
	"context"
	"log/slog"

	"github.com/plexusone/omniobserve/observops"
)

// Context keys for storing observability components.
type contextKey int

const (
	observabilityKey contextKey = iota
	loggerKey
	spanKey
)

// ContextWithObservability returns a new context with the Observability instance.
func ContextWithObservability(ctx context.Context, o *Observability) context.Context {
	return context.WithValue(ctx, observabilityKey, o)
}

// ObservabilityFromContext retrieves the Observability instance from the context.
// Returns nil if not found.
func ObservabilityFromContext(ctx context.Context) *Observability {
	o, _ := ctx.Value(observabilityKey).(*Observability)
	return o
}

// ContextWithLogger returns a new context with the slog.Logger.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext retrieves the slog.Logger from the context.
// If no logger is found, it checks for an Observability instance and returns its logger.
// If nothing is found, returns slog.Default().
func LoggerFromContext(ctx context.Context) *slog.Logger {
	// First, check for a directly stored logger
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}

	// Check for Observability instance
	if o := ObservabilityFromContext(ctx); o != nil {
		return o.LoggerFromContext(ctx)
	}

	// Fall back to default
	return slog.Default()
}

// ContextWithSpan returns a new context with the span.
// This is typically called by middleware after starting a span.
func ContextWithSpan(ctx context.Context, span observops.Span) context.Context {
	return context.WithValue(ctx, spanKey, span)
}

// SpanFromContext retrieves the span from the context.
// Returns nil if not found.
func SpanFromContext(ctx context.Context) observops.Span {
	span, _ := ctx.Value(spanKey).(*wrappedSpan)
	if span != nil {
		return span.Span
	}

	// Also check the Observability's tracer
	if o := ObservabilityFromContext(ctx); o != nil {
		return o.Tracer().SpanFromContext(ctx)
	}

	return nil
}

// wrappedSpan wraps an observops.Span for context storage.
type wrappedSpan struct {
	observops.Span
}

// L is a shorthand for LoggerFromContext.
func L(ctx context.Context) *slog.Logger {
	return LoggerFromContext(ctx)
}

// S is a shorthand for SpanFromContext.
func S(ctx context.Context) observops.Span {
	return SpanFromContext(ctx)
}

// O is a shorthand for ObservabilityFromContext.
func O(ctx context.Context) *Observability {
	return ObservabilityFromContext(ctx)
}

// WithTraceContext returns a logger with trace context attributes from the current span.
func WithTraceContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	span := SpanFromContext(ctx)
	if span == nil {
		return logger
	}

	sc := span.SpanContext()
	if sc.TraceID == "" {
		return logger
	}

	return logger.With(
		slog.String("trace_id", sc.TraceID),
		slog.String("span_id", sc.SpanID),
	)
}

// Trace starts a new span and executes the function within that span context.
// The span is automatically ended when the function returns.
// If an error is returned, it is recorded on the span.
func Trace(ctx context.Context, name string, fn func(ctx context.Context) error, opts ...observops.SpanOption) error {
	o := ObservabilityFromContext(ctx)
	if o == nil {
		return fn(ctx)
	}

	ctx, span := o.StartSpan(ctx, name, opts...)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(observops.StatusCodeError, err.Error())
	}

	return err
}

// TraceFunc is like Trace but for functions that return a value and an error.
func TraceFunc[T any](ctx context.Context, name string, fn func(ctx context.Context) (T, error), opts ...observops.SpanOption) (T, error) {
	o := ObservabilityFromContext(ctx)
	if o == nil {
		return fn(ctx)
	}

	ctx, span := o.StartSpan(ctx, name, opts...)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(observops.StatusCodeError, err.Error())
	}

	return result, err
}
