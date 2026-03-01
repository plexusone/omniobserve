package omnillm

import (
	"context"

	"github.com/plexusone/omniobserve/llmops"
)

// contextKey is a private type used for storing spans in context.
type contextKey struct{}

// traceContextKey is a private type used for storing traces we created.
type traceContextKey struct{}

// contextWithSpan returns a new context with the span attached.
func contextWithSpan(ctx context.Context, span llmops.Span) context.Context {
	return context.WithValue(ctx, contextKey{}, span)
}

// spanFromContext retrieves the span from the context.
// Returns nil if no span is attached.
func spanFromContext(ctx context.Context) llmops.Span {
	span, _ := ctx.Value(contextKey{}).(llmops.Span)
	return span
}

// contextWithTrace returns a new context with the trace attached.
// This is used to track traces we create so we can end them in AfterResponse.
func contextWithTrace(ctx context.Context, trace llmops.Trace) context.Context {
	return context.WithValue(ctx, traceContextKey{}, trace)
}

// traceFromContext retrieves the trace from the context.
// Returns nil if no trace is attached.
func traceFromContext(ctx context.Context) llmops.Trace {
	trace, _ := ctx.Value(traceContextKey{}).(llmops.Trace)
	return trace
}
