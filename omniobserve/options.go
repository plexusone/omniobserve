package omniobserve

import (
	"log/slog"

	"github.com/plexusone/omniobserve/observops"
)

// Config holds the configuration for the Observability instance.
type Config struct {
	// ServiceName is the name of the service (required).
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// Endpoint is the observability backend endpoint.
	Endpoint string

	// APIKey is the API key for authentication.
	APIKey string

	// Headers are additional headers to send with requests.
	Headers map[string]string

	// Insecure disables TLS verification.
	Insecure bool

	// Disabled disables all observability (returns noop provider).
	Disabled bool

	// Debug enables debug logging for the provider.
	Debug bool

	// LocalHandler is a local slog.Handler for dual output.
	LocalHandler slog.Handler

	// SlogLevel is the minimum level for remote slog output.
	SlogLevel slog.Level

	// ProviderOptions are additional options passed to the provider.
	ProviderOptions []observops.ClientOption

	// Middleware configuration
	MiddlewareConfig MiddlewareConfig
}

// MiddlewareConfig holds middleware-specific configuration.
type MiddlewareConfig struct {
	// SkipPaths are paths to skip tracing (e.g., health checks).
	SkipPaths []string

	// SkipFunc is a custom function to determine if a path should be skipped.
	SkipFunc func(path string) bool

	// SpanNameFormatter formats the span name from method and path.
	SpanNameFormatter func(method, path string) string

	// RecordRequestBody records request body as span attribute.
	RecordRequestBody bool

	// RecordResponseBody records response body as span attribute.
	RecordResponseBody bool

	// MaxBodySize is the maximum body size to record (default 1KB).
	MaxBodySize int

	// PropagateTraceID adds trace_id to response headers.
	PropagateTraceID bool

	// TraceIDHeader is the header name for trace ID (default "X-Trace-ID").
	TraceIDHeader string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Headers: make(map[string]string),
		MiddlewareConfig: MiddlewareConfig{
			SkipPaths:        []string{"/health", "/healthz", "/ready", "/readyz", "/live", "/livez", "/metrics"},
			MaxBodySize:      1024,
			PropagateTraceID: true,
			TraceIDHeader:    "X-Trace-ID",
			SpanNameFormatter: func(method, path string) string {
				return method + " " + path
			},
		},
	}
}

// Option is a functional option for configuring Observability.
type Option func(*Config)

// WithServiceName sets the service name.
func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithServiceVersion sets the service version.
func WithServiceVersion(version string) Option {
	return func(c *Config) {
		c.ServiceVersion = version
	}
}

// WithEndpoint sets the observability backend endpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.Endpoint = endpoint
	}
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *Config) {
		c.APIKey = key
	}
}

// WithHeaders sets additional headers.
func WithHeaders(headers map[string]string) Option {
	return func(c *Config) {
		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithInsecure disables TLS verification.
func WithInsecure() Option {
	return func(c *Config) {
		c.Insecure = true
	}
}

// WithDisabled disables all observability.
func WithDisabled() Option {
	return func(c *Config) {
		c.Disabled = true
	}
}

// WithDebug enables debug logging.
func WithDebug() Option {
	return func(c *Config) {
		c.Debug = true
	}
}

// WithLocalHandler sets a local slog.Handler for dual output.
func WithLocalHandler(h slog.Handler) Option {
	return func(c *Config) {
		c.LocalHandler = h
	}
}

// WithSlogLevel sets the minimum level for remote slog output.
func WithSlogLevel(level slog.Level) Option {
	return func(c *Config) {
		c.SlogLevel = level
	}
}

// WithProviderOptions adds additional options for the underlying provider.
func WithProviderOptions(opts ...observops.ClientOption) Option {
	return func(c *Config) {
		c.ProviderOptions = append(c.ProviderOptions, opts...)
	}
}

// WithSkipPaths sets paths to skip in middleware.
func WithSkipPaths(paths ...string) Option {
	return func(c *Config) {
		c.MiddlewareConfig.SkipPaths = paths
	}
}

// WithSkipFunc sets a custom skip function for middleware.
func WithSkipFunc(fn func(path string) bool) Option {
	return func(c *Config) {
		c.MiddlewareConfig.SkipFunc = fn
	}
}

// WithSpanNameFormatter sets the span name formatter.
func WithSpanNameFormatter(fn func(method, path string) string) Option {
	return func(c *Config) {
		c.MiddlewareConfig.SpanNameFormatter = fn
	}
}

// WithRecordRequestBody enables recording request body.
func WithRecordRequestBody() Option {
	return func(c *Config) {
		c.MiddlewareConfig.RecordRequestBody = true
	}
}

// WithRecordResponseBody enables recording response body.
func WithRecordResponseBody() Option {
	return func(c *Config) {
		c.MiddlewareConfig.RecordResponseBody = true
	}
}

// WithMaxBodySize sets the maximum body size to record.
func WithMaxBodySize(size int) Option {
	return func(c *Config) {
		c.MiddlewareConfig.MaxBodySize = size
	}
}

// WithPropagateTraceID enables/disables trace ID propagation in response headers.
func WithPropagateTraceID(enabled bool) Option {
	return func(c *Config) {
		c.MiddlewareConfig.PropagateTraceID = enabled
	}
}

// WithTraceIDHeader sets the header name for trace ID.
func WithTraceIDHeader(header string) Option {
	return func(c *Config) {
		c.MiddlewareConfig.TraceIDHeader = header
	}
}
