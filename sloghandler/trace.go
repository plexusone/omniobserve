// Package sloghandler provides an slog.Handler implementation that integrates
// with OmniObserve's observability stack, enabling automatic trace correlation
// and multi-backend log export.
package sloghandler

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TraceContext holds trace identification extracted from context.
type TraceContext struct {
	// TraceID is the trace identifier.
	TraceID string

	// SpanID is the span identifier.
	SpanID string

	// TraceFlags contains trace flags (e.g., sampled).
	TraceFlags byte

	// Remote indicates if the span context was propagated from a remote parent.
	Remote bool
}

// IsValid returns true if the trace context has valid IDs.
func (tc TraceContext) IsValid() bool {
	return tc.TraceID != "" && tc.SpanID != ""
}

// TraceContextExtractor extracts trace context from a context.Context.
type TraceContextExtractor func(ctx context.Context) TraceContext

// DefaultTraceContextExtractor extracts trace context using OpenTelemetry.
func DefaultTraceContextExtractor(ctx context.Context) TraceContext {
	if ctx == nil {
		return TraceContext{}
	}

	// Extract from OTel span
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return TraceContext{}
	}

	sc := span.SpanContext()
	if !sc.IsValid() {
		return TraceContext{}
	}

	return TraceContext{
		TraceID:    sc.TraceID().String(),
		SpanID:     sc.SpanID().String(),
		TraceFlags: byte(sc.TraceFlags()),
		Remote:     sc.IsRemote(),
	}
}

// observopsSpanContexter is an interface that matches observops.Span.SpanContext().
// We use an interface to avoid import cycles.
type observopsSpanContexter interface {
	SpanContext() struct {
		TraceID    string
		SpanID     string
		TraceFlags byte
		Remote     bool
	}
}

// observopsSpanKey is the context key used by observops to store spans.
// This matches the key used in observops package.
type observopsSpanKey struct{}

// ObservopsTraceContextExtractor extracts trace context from observops spans.
// Falls back to OTel if no observops span is found.
func ObservopsTraceContextExtractor(ctx context.Context) TraceContext {
	if ctx == nil {
		return TraceContext{}
	}

	// Try observops span first
	if span, ok := ctx.Value(observopsSpanKey{}).(observopsSpanContexter); ok && span != nil {
		sc := span.SpanContext()
		if sc.TraceID != "" {
			return TraceContext{
				TraceID:    sc.TraceID,
				SpanID:     sc.SpanID,
				TraceFlags: sc.TraceFlags,
				Remote:     sc.Remote,
			}
		}
	}

	// Fall back to OTel
	return DefaultTraceContextExtractor(ctx)
}

// ChainedTraceContextExtractor chains multiple extractors, returning the first valid result.
func ChainedTraceContextExtractor(extractors ...TraceContextExtractor) TraceContextExtractor {
	return func(ctx context.Context) TraceContext {
		for _, ext := range extractors {
			if tc := ext(ctx); tc.IsValid() {
				return tc
			}
		}
		return TraceContext{}
	}
}
