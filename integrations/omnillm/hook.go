// Package omnillm provides an ObservabilityHook implementation for OmniLLM
// that integrates with OmniObserve's llmops providers (Opik, Langfuse, Phoenix).
package omnillm

import (
	"context"

	"github.com/plexusone/omnillm"
	"github.com/plexusone/omnillm/provider"

	"github.com/plexusone/omniobserve/llmops"
)

// Hook implements omnillm.ObservabilityHook using an llmops.Provider.
// It automatically creates spans for each LLM call with model, provider,
// input/output, and token usage information.
type Hook struct {
	provider llmops.Provider
}

// NewHook creates a new OmniLLM observability hook.
// The provider should be initialized before passing to this function.
func NewHook(provider llmops.Provider) *Hook {
	return &Hook{provider: provider}
}

// Ensure Hook implements the interface at compile time
var _ omnillm.ObservabilityHook = (*Hook)(nil)

// BeforeRequest is called before each LLM call.
// It starts a new trace and span, returning a context with both attached.
func (h *Hook) BeforeRequest(ctx context.Context, info omnillm.LLMCallInfo, req *provider.ChatCompletionRequest) context.Context {
	// Check if there's already a trace in context
	_, hasTrace := h.provider.TraceFromContext(ctx)

	var createdTrace llmops.Trace
	if !hasTrace {
		// Create a trace for this LLM call
		traceName := "llm-call"
		if req.Model != "" {
			traceName = "llm-call-" + req.Model
		}
		var err error
		ctx, createdTrace, err = h.provider.StartTrace(ctx, traceName,
			llmops.WithTraceInput(req.Messages),
		)
		if err != nil {
			// Don't fail the request if observability fails
			return ctx
		}
	}

	// Start a span for this LLM call
	ctx, span, err := h.provider.StartSpan(ctx, "llm-completion",
		llmops.WithSpanType(llmops.SpanTypeLLM),
		llmops.WithModel(req.Model),
		llmops.WithProvider(info.ProviderName),
		llmops.WithSpanInput(req.Messages),
	)
	if err != nil {
		// If span creation failed but we created a trace, end it
		if createdTrace != nil {
			_ = createdTrace.End()
		}
		return ctx
	}

	// Store span and trace (if we created one) in context for AfterResponse
	ctx = contextWithSpan(ctx, span)
	if createdTrace != nil {
		ctx = contextWithTrace(ctx, createdTrace)
	}
	return ctx
}

// AfterResponse is called after each LLM call completes.
// It records the response output, token usage, and ends the span and trace.
func (h *Hook) AfterResponse(ctx context.Context, info omnillm.LLMCallInfo, req *provider.ChatCompletionRequest, resp *provider.ChatCompletionResponse, err error) {
	span := spanFromContext(ctx)
	trace := traceFromContext(ctx)

	// End span first
	if span != nil {
		if err != nil {
			_ = span.End(llmops.WithEndError(err))
		} else {
			if resp != nil {
				// Set output
				if len(resp.Choices) > 0 {
					output := resp.Choices[0].Message.Content
					_ = span.SetOutput(output)
				}

				// Set token usage
				_ = span.SetUsage(llmops.TokenUsage{
					PromptTokens:     resp.Usage.PromptTokens,
					CompletionTokens: resp.Usage.CompletionTokens,
					TotalTokens:      resp.Usage.TotalTokens,
				})
			}
			_ = span.End()
		}
	}

	// End trace if we created one
	if trace != nil {
		if resp != nil && len(resp.Choices) > 0 {
			output := resp.Choices[0].Message.Content
			_ = trace.SetOutput(output)
		}
		_ = trace.End()
	}
}

// WrapStream wraps a stream for observability.
// The wrapped stream will buffer content and record it when the stream ends.
func (h *Hook) WrapStream(ctx context.Context, info omnillm.LLMCallInfo, req *provider.ChatCompletionRequest, stream provider.ChatCompletionStream) provider.ChatCompletionStream {
	span := spanFromContext(ctx)
	if span == nil {
		return stream
	}
	trace := traceFromContext(ctx)
	return &observedStream{
		stream: stream,
		span:   span,
		trace:  trace,
		info:   info,
	}
}
