package recorder

import (
	"net/http"
	"time"

	"github.com/plexusone/omniobserve/specs/red"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware returns HTTP middleware that records RED metrics.
func (r *Recorder) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(rw, req)

		// Record RED metrics. Error is intentionally ignored here because we're in
		// HTTP middleware where we can't propagate errors to the caller. Metric
		// recording failures are non-fatal and should not affect request handling.
		_ = r.RecordRED(req.Context(), "http.server.request", red.Observation{
			Duration:   time.Since(start),
			StatusCode: rw.statusCode,
			Attributes: map[string]string{
				"service.name":     r.serviceName,
				"http.method":      req.Method,
				"http.route":       req.URL.Path,
				"http.status_code": http.StatusText(rw.statusCode),
			},
		})
	})
}

// HTTPMiddlewareFunc returns HTTP middleware as a function compatible with Chi router.
func (r *Recorder) HTTPMiddlewareFunc() func(http.Handler) http.Handler {
	return r.HTTPMiddleware
}
