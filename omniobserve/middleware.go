package omniobserve

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/plexusone/omniobserve/observops"
)

// Middleware returns an http.Handler middleware that instruments HTTP requests
// with tracing, metrics, and logging.
func (o *Observability) Middleware() func(http.Handler) http.Handler {
	cfg := o.config.MiddlewareConfig

	// Pre-create metrics
	requestCounter, _ := o.Counter("http.server.request.total",
		observops.WithDescription("Total number of HTTP requests"),
	)
	requestDuration, _ := o.Histogram("http.server.request.duration",
		observops.WithDescription("HTTP request duration in milliseconds"),
		observops.WithUnit("ms"),
	)
	requestSize, _ := o.Histogram("http.server.request.size",
		observops.WithDescription("HTTP request body size in bytes"),
		observops.WithUnit("By"),
	)
	responseSize, _ := o.Histogram("http.server.response.size",
		observops.WithDescription("HTTP response body size in bytes"),
		observops.WithUnit("By"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			if o.shouldSkip(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Start span
			spanName := cfg.SpanNameFormatter(r.Method, r.URL.Path)
			ctx, span := o.StartSpan(r.Context(), spanName,
				observops.WithSpanKind(observops.SpanKindServer),
				observops.WithSpanAttributes(
					String("http.method", r.Method),
					String("http.url", r.URL.String()),
					String("http.target", r.URL.Path),
					String("http.host", r.Host),
					String("http.scheme", scheme(r)),
					String("http.user_agent", r.UserAgent()),
					String("http.client_ip", clientIP(r)),
				),
			)
			defer span.End()

			// Add trace context to response header
			if cfg.PropagateTraceID {
				sc := span.SpanContext()
				if sc.TraceID != "" {
					w.Header().Set(cfg.TraceIDHeader, sc.TraceID)
				}
			}

			// Store observability and logger in context
			ctx = ContextWithObservability(ctx, o)
			ctx = ContextWithLogger(ctx, o.LoggerFromContext(ctx))

			// Record request body if configured
			if cfg.RecordRequestBody && r.Body != nil && r.ContentLength > 0 && r.ContentLength <= int64(cfg.MaxBodySize) {
				body, err := io.ReadAll(io.LimitReader(r.Body, int64(cfg.MaxBodySize)))
				if err == nil {
					span.SetAttributes(String("http.request.body", string(body)))
					r.Body = io.NopCloser(bytes.NewReader(body))
				}
			}

			// Wrap response writer to capture status and size
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Serve request
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record metrics and span attributes
			duration := time.Since(start)
			durationMs := float64(duration.Milliseconds())

			attrs := []observops.KeyValue{
				String("http.method", r.Method),
				String("http.route", r.URL.Path),
				Int("http.status_code", rw.statusCode),
			}

			if requestCounter != nil {
				requestCounter.Add(ctx, 1, observops.WithAttributes(attrs...))
			}
			if requestDuration != nil {
				requestDuration.Record(ctx, durationMs, observops.WithAttributes(attrs...))
			}
			if requestSize != nil && r.ContentLength > 0 {
				requestSize.Record(ctx, float64(r.ContentLength), observops.WithAttributes(attrs...))
			}
			if responseSize != nil && rw.bytesWritten > 0 {
				responseSize.Record(ctx, float64(rw.bytesWritten), observops.WithAttributes(attrs...))
			}

			// Update span with response info
			span.SetAttributes(
				Int("http.status_code", rw.statusCode),
				Int("http.response.size", rw.bytesWritten),
				Float64("http.duration_ms", durationMs),
			)

			// Set span status based on HTTP status
			if rw.statusCode >= 400 {
				span.SetStatus(observops.StatusCodeError, http.StatusText(rw.statusCode))
			} else {
				span.SetStatus(observops.StatusCodeOK, "")
			}

			// Log the request
			logger := o.LoggerFromContext(ctx)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", durationMs,
				"bytes", rw.bytesWritten,
				"client_ip", clientIP(r),
			)
		})
	}
}

// shouldSkip returns true if the path should be skipped.
func (o *Observability) shouldSkip(path string) bool {
	cfg := o.config.MiddlewareConfig

	// Check custom skip function first
	if cfg.SkipFunc != nil && cfg.SkipFunc(path) {
		return true
	}

	// Check skip paths
	for _, p := range cfg.SkipPaths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}

	return false
}

// responseWriter wraps http.ResponseWriter to capture response info.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// scheme returns the request scheme (http or https).
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if s := r.Header.Get("X-Forwarded-Proto"); s != "" {
		return s
	}
	return "http"
}

// clientIP extracts the client IP from the request.
func clientIP(r *http.Request) string {
	// Check X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
