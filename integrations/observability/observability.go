// Package observability provides integration helpers for using observops with
// multi-agent systems and general Go applications.
//
// This package provides utilities for:
//   - HTTP client middleware for distributed tracing
//   - HTTP server middleware for request instrumentation
//   - Agent task instrumentation for multi-agent workflows
//   - Context helpers for trace propagation
//
// # HTTP Client Middleware
//
// Wrap your HTTP client to automatically trace outgoing requests:
//
//	client := observability.WrapHTTPClient(http.DefaultClient, provider)
//	resp, err := client.Get("https://api.example.com/data")
//
// # HTTP Server Middleware
//
// Use the handler middleware for automatic request tracing:
//
//	handler := observability.HTTPMiddleware(provider)(yourHandler)
//	http.ListenAndServe(":8080", handler)
//
// # Agent Task Instrumentation
//
// Instrument agent tasks for multi-agent workflows:
//
//	err := observability.ObserveTask(ctx, provider, "ProcessData",
//		observability.WithAgentID("agent-1"),
//		observability.WithTaskType("synthesis"),
//		func(ctx context.Context) error {
//			// Your task logic here
//			return nil
//		},
//	)
package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/plexusone/omniobserve/observops"
)

// HTTPMiddleware returns a middleware that instruments HTTP handlers with tracing.
func HTTPMiddleware(provider observops.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &tracingHandler{
			next:     next,
			provider: provider,
		}
	}
}

type tracingHandler struct {
	next     http.Handler
	provider observops.Provider
}

func (h *tracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tracer := h.provider.Tracer()

	// Extract span name from request
	spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

	// Start span
	ctx, span := tracer.Start(r.Context(), spanName,
		observops.WithSpanKind(observops.SpanKindServer),
		observops.WithSpanAttributes(
			observops.Attribute("http.method", r.Method),
			observops.Attribute("http.url", r.URL.String()),
			observops.Attribute("http.host", r.Host),
			observops.Attribute("http.user_agent", r.UserAgent()),
		),
	)
	defer span.End()

	// Wrap response writer to capture status code
	wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Serve request
	h.next.ServeHTTP(wrapped, r.WithContext(ctx))

	// Set status attributes
	span.SetAttributes(observops.Attribute("http.status_code", wrapped.statusCode))

	if wrapped.statusCode >= 400 {
		span.SetStatus(observops.StatusCodeError, http.StatusText(wrapped.statusCode))
	} else {
		span.SetStatus(observops.StatusCodeOK, "")
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// WrapHTTPClient returns a new HTTP client that instruments outgoing requests.
func WrapHTTPClient(client *http.Client, provider observops.Provider) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Transport:     &tracingTransport{transport: transport, provider: provider},
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
		Timeout:       client.Timeout,
	}
}

type tracingTransport struct {
	transport http.RoundTripper
	provider  observops.Provider
}

func (t *tracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tracer := t.provider.Tracer()

	// Start span
	spanName := fmt.Sprintf("%s %s", req.Method, req.URL.Host)
	ctx, span := tracer.Start(req.Context(), spanName,
		observops.WithSpanKind(observops.SpanKindClient),
		observops.WithSpanAttributes(
			observops.Attribute("http.method", req.Method),
			observops.Attribute("http.url", req.URL.String()),
			observops.Attribute("http.host", req.URL.Host),
		),
	)
	defer span.End()

	// Use context with span
	req = req.WithContext(ctx)

	// Inject trace context into headers (W3C Trace Context format)
	spanCtx := span.SpanContext()
	if spanCtx.TraceID != "" {
		req.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-%02x",
			spanCtx.TraceID, spanCtx.SpanID, spanCtx.TraceFlags))
	}

	// Execute request
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(observops.StatusCodeError, err.Error())
		return nil, err
	}

	// Set response attributes
	span.SetAttributes(observops.Attribute("http.status_code", resp.StatusCode))

	if resp.StatusCode >= 400 {
		span.SetStatus(observops.StatusCodeError, http.StatusText(resp.StatusCode))
	} else {
		span.SetStatus(observops.StatusCodeOK, "")
	}

	return resp, nil
}

// TaskOption configures task observation.
type TaskOption func(*taskConfig)

type taskConfig struct {
	agentID    string
	taskType   string
	metadata   map[string]string
	attributes []observops.KeyValue
}

// WithAgentID sets the agent ID for the task.
func WithAgentID(id string) TaskOption {
	return func(c *taskConfig) {
		c.agentID = id
	}
}

// WithTaskType sets the task type.
func WithTaskType(taskType string) TaskOption {
	return func(c *taskConfig) {
		c.taskType = taskType
	}
}

// WithTaskMetadata sets task metadata.
func WithTaskMetadata(metadata map[string]string) TaskOption {
	return func(c *taskConfig) {
		c.metadata = metadata
	}
}

// WithTaskAttributes sets additional span attributes.
func WithTaskAttributes(attrs ...observops.KeyValue) TaskOption {
	return func(c *taskConfig) {
		c.attributes = append(c.attributes, attrs...)
	}
}

// ObserveTask instruments a task with tracing and metrics.
// It automatically creates a span for the task, records duration,
// and logs errors if the task fails.
func ObserveTask(ctx context.Context, provider observops.Provider, name string, opts []TaskOption, fn func(context.Context) error) error {
	cfg := &taskConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	tracer := provider.Tracer()
	logger := provider.Logger()
	meter := provider.Meter()

	// Build span attributes
	attrs := []observops.KeyValue{}
	if cfg.agentID != "" {
		attrs = append(attrs, observops.Attribute("agent.id", cfg.agentID))
	}
	if cfg.taskType != "" {
		attrs = append(attrs, observops.Attribute("task.type", cfg.taskType))
	}
	for k, v := range cfg.metadata {
		attrs = append(attrs, observops.Attribute("task.metadata."+k, v))
	}
	attrs = append(attrs, cfg.attributes...)

	// Start span
	ctx, span := tracer.Start(ctx, name,
		observops.WithSpanKind(observops.SpanKindInternal),
		observops.WithSpanAttributes(attrs...),
	)

	startTime := time.Now()

	// Execute task
	err := fn(ctx)

	duration := time.Since(startTime)

	// Record metrics
	if durationHist, histErr := meter.Histogram("task.duration",
		observops.WithDescription("Task execution duration"),
		observops.WithUnit("ms"),
	); histErr == nil {
		recordAttrs := []observops.KeyValue{}
		if cfg.agentID != "" {
			recordAttrs = append(recordAttrs, observops.Attribute("agent.id", cfg.agentID))
		}
		if cfg.taskType != "" {
			recordAttrs = append(recordAttrs, observops.Attribute("task.type", cfg.taskType))
		}
		recordAttrs = append(recordAttrs, observops.Attribute("task.name", name))
		recordAttrs = append(recordAttrs, observops.Attribute("task.success", err == nil))
		durationHist.Record(ctx, float64(duration.Milliseconds()), observops.WithAttributes(recordAttrs...))
	}

	// Record task completion/failure counter
	if taskCounter, counterErr := meter.Counter("task.completed",
		observops.WithDescription("Number of completed tasks"),
	); counterErr == nil {
		recordAttrs := []observops.KeyValue{}
		if cfg.agentID != "" {
			recordAttrs = append(recordAttrs, observops.Attribute("agent.id", cfg.agentID))
		}
		if cfg.taskType != "" {
			recordAttrs = append(recordAttrs, observops.Attribute("task.type", cfg.taskType))
		}
		recordAttrs = append(recordAttrs, observops.Attribute("task.name", name))
		recordAttrs = append(recordAttrs, observops.Attribute("task.success", err == nil))
		taskCounter.Add(ctx, 1, observops.WithAttributes(recordAttrs...))
	}

	// Handle result
	if err != nil {
		span.RecordError(err)
		span.SetStatus(observops.StatusCodeError, err.Error())
		logger.Error(ctx, "Task failed",
			observops.LogAttr("task.name", name),
			observops.LogAttr("error", err.Error()),
			observops.LogAttr("duration_ms", duration.Milliseconds()),
		)
	} else {
		span.SetStatus(observops.StatusCodeOK, "")
		logger.Info(ctx, "Task completed",
			observops.LogAttr("task.name", name),
			observops.LogAttr("duration_ms", duration.Milliseconds()),
		)
	}

	span.End()
	return err
}

// StartAgentSpan creates a new span for an agent operation.
// This is useful for instrumenting agent workflows without using ObserveTask.
func StartAgentSpan(ctx context.Context, provider observops.Provider, name string, agentID string, opts ...observops.SpanOption) (context.Context, observops.Span) {
	attrs := []observops.KeyValue{
		observops.Attribute("agent.id", agentID),
	}

	// Merge with provided options
	allOpts := append([]observops.SpanOption{
		observops.WithSpanKind(observops.SpanKindInternal),
		observops.WithSpanAttributes(attrs...),
	}, opts...)

	return provider.Tracer().Start(ctx, name, allOpts...)
}

// RecordAgentMetric records a metric with agent context.
func RecordAgentMetric(ctx context.Context, provider observops.Provider, name string, value float64, agentID string, additionalAttrs ...observops.KeyValue) error {
	gauge, err := provider.Meter().Gauge(name)
	if err != nil {
		return err
	}

	attrs := []observops.KeyValue{
		observops.Attribute("agent.id", agentID),
	}
	attrs = append(attrs, additionalAttrs...)

	gauge.Record(ctx, value, observops.WithAttributes(attrs...))
	return nil
}

// IncrementAgentCounter increments a counter with agent context.
func IncrementAgentCounter(ctx context.Context, provider observops.Provider, name string, agentID string, additionalAttrs ...observops.KeyValue) error {
	counter, err := provider.Meter().Counter(name)
	if err != nil {
		return err
	}

	attrs := []observops.KeyValue{
		observops.Attribute("agent.id", agentID),
	}
	attrs = append(attrs, additionalAttrs...)

	counter.Add(ctx, 1, observops.WithAttributes(attrs...))
	return nil
}
