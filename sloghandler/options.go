package sloghandler

import (
	"log/slog"
)

// Config holds configuration for the Handler.
type Config struct {
	// LocalHandler is the handler for local output (console, file, etc.).
	// If nil, no local output is produced.
	LocalHandler slog.Handler

	// RemoteHandler is the handler for remote output (observability backend).
	// If nil, no remote output is produced.
	RemoteHandler slog.Handler

	// RemoteLevel is the minimum level for remote export.
	// Logs below this level are only sent to the local handler.
	// Default: slog.LevelInfo
	RemoteLevel slog.Level

	// TraceContextExtractor extracts trace context from context.Context.
	// Default: DefaultTraceContextExtractor (OTel)
	TraceContextExtractor TraceContextExtractor

	// IncludeTraceContext controls whether trace_id and span_id are added to logs.
	// Default: true
	IncludeTraceContext bool

	// TraceIDKey is the attribute key for trace ID.
	// Default: "trace_id"
	TraceIDKey string

	// SpanIDKey is the attribute key for span ID.
	// Default: "span_id"
	SpanIDKey string

	// Processors are applied to attributes before logging.
	Processors []AttributeProcessor

	// AddSource adds source file information to log records.
	// Default: false
	AddSource bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		RemoteLevel:           slog.LevelInfo,
		TraceContextExtractor: DefaultTraceContextExtractor,
		IncludeTraceContext:   true,
		TraceIDKey:            "trace_id",
		SpanIDKey:             "span_id",
	}
}

// Option configures a Handler.
type Option func(*Config)

// ApplyOptions applies options to a config.
func ApplyOptions(cfg *Config, opts ...Option) {
	for _, opt := range opts {
		opt(cfg)
	}
}

// WithLocalHandler sets the local output handler.
func WithLocalHandler(h slog.Handler) Option {
	return func(c *Config) {
		c.LocalHandler = h
	}
}

// WithRemoteHandler sets the remote output handler.
func WithRemoteHandler(h slog.Handler) Option {
	return func(c *Config) {
		c.RemoteHandler = h
	}
}

// WithRemoteLevel sets the minimum level for remote export.
func WithRemoteLevel(level slog.Level) Option {
	return func(c *Config) {
		c.RemoteLevel = level
	}
}

// WithTraceContextExtractor sets the trace context extractor.
func WithTraceContextExtractor(ext TraceContextExtractor) Option {
	return func(c *Config) {
		c.TraceContextExtractor = ext
	}
}

// WithoutTraceContext disables automatic trace context inclusion.
func WithoutTraceContext() Option {
	return func(c *Config) {
		c.IncludeTraceContext = false
	}
}

// WithTraceIDKey sets the attribute key for trace ID.
func WithTraceIDKey(key string) Option {
	return func(c *Config) {
		c.TraceIDKey = key
	}
}

// WithSpanIDKey sets the attribute key for span ID.
func WithSpanIDKey(key string) Option {
	return func(c *Config) {
		c.SpanIDKey = key
	}
}

// WithProcessor adds an attribute processor.
func WithProcessor(p AttributeProcessor) Option {
	return func(c *Config) {
		c.Processors = append(c.Processors, p)
	}
}

// WithAddSource enables source file information in log records.
func WithAddSource() Option {
	return func(c *Config) {
		c.AddSource = true
	}
}

// AttributeProcessor processes log attributes before output.
type AttributeProcessor interface {
	// Process transforms attributes. Return nil to drop the record.
	Process(attrs []slog.Attr) []slog.Attr
}

// AttributeProcessorFunc is a function that implements AttributeProcessor.
type AttributeProcessorFunc func([]slog.Attr) []slog.Attr

// Process implements AttributeProcessor.
func (f AttributeProcessorFunc) Process(attrs []slog.Attr) []slog.Attr {
	return f(attrs)
}

// FilterProcessor creates a processor that filters attributes by key.
func FilterProcessor(allowKeys ...string) AttributeProcessor {
	allowed := make(map[string]bool, len(allowKeys))
	for _, k := range allowKeys {
		allowed[k] = true
	}
	return AttributeProcessorFunc(func(attrs []slog.Attr) []slog.Attr {
		result := make([]slog.Attr, 0, len(attrs))
		for _, a := range attrs {
			if allowed[a.Key] {
				result = append(result, a)
			}
		}
		return result
	})
}

// RedactProcessor creates a processor that redacts sensitive attribute values.
func RedactProcessor(redactKeys ...string) AttributeProcessor {
	redact := make(map[string]bool, len(redactKeys))
	for _, k := range redactKeys {
		redact[k] = true
	}
	return AttributeProcessorFunc(func(attrs []slog.Attr) []slog.Attr {
		result := make([]slog.Attr, len(attrs))
		for i, a := range attrs {
			if redact[a.Key] {
				result[i] = slog.String(a.Key, "[REDACTED]")
			} else {
				result[i] = a
			}
		}
		return result
	})
}

// RenameProcessor creates a processor that renames attribute keys.
func RenameProcessor(renames map[string]string) AttributeProcessor {
	return AttributeProcessorFunc(func(attrs []slog.Attr) []slog.Attr {
		result := make([]slog.Attr, len(attrs))
		for i, a := range attrs {
			if newKey, ok := renames[a.Key]; ok {
				result[i] = slog.Attr{Key: newKey, Value: a.Value}
			} else {
				result[i] = a
			}
		}
		return result
	})
}
