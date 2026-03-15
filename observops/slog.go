package observops

import (
	"context"
	"log/slog"
	"slices"

	"go.opentelemetry.io/otel/trace"
)

// LoggerSlogHandler wraps an observops.Logger as an slog.Handler.
// This allows using observops.Logger as the remote backend for sloghandler.
type LoggerSlogHandler struct {
	logger              Logger
	localHandler        slog.Handler
	remoteLevel         slog.Level
	includeTraceContext bool
	traceIDKey          string
	spanIDKey           string
	groups              []string
	attrs               []slog.Attr
}

// NewLoggerSlogHandler creates an slog.Handler that wraps an observops.Logger.
func NewLoggerSlogHandler(logger Logger, cfg *SlogConfig) *LoggerSlogHandler {
	if cfg == nil {
		cfg = DefaultSlogConfig()
	}

	var localHandler slog.Handler
	if cfg.LocalHandler != nil {
		if h, ok := cfg.LocalHandler.(slog.Handler); ok {
			localHandler = h
		}
	}

	return &LoggerSlogHandler{
		logger:              logger,
		localHandler:        localHandler,
		remoteLevel:         slog.Level(cfg.RemoteLevel),
		includeTraceContext: cfg.IncludeTraceContext,
		traceIDKey:          cfg.TraceIDKey,
		spanIDKey:           cfg.SpanIDKey,
	}
}

// Enabled implements slog.Handler.
func (h *LoggerSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	localEnabled := h.localHandler != nil && h.localHandler.Enabled(ctx, level)
	remoteEnabled := h.logger != nil && level >= h.remoteLevel
	return localEnabled || remoteEnabled
}

// Handle implements slog.Handler.
func (h *LoggerSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Collect attributes
	attrs := h.collectAttrs(r)

	// Extract trace context
	var traceAttrs []slog.Attr
	if h.includeTraceContext {
		if tc := extractTraceContext(ctx); tc.TraceID != "" {
			traceAttrs = []slog.Attr{
				slog.String(h.traceIDKey, tc.TraceID),
				slog.String(h.spanIDKey, tc.SpanID),
			}
		}
	}

	var firstErr error

	// Handle locally
	if h.localHandler != nil && h.localHandler.Enabled(ctx, r.Level) {
		localRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		for _, a := range h.attrs {
			localRecord.AddAttrs(a)
		}
		for _, a := range attrs {
			localRecord.AddAttrs(a)
		}
		for _, a := range traceAttrs {
			localRecord.AddAttrs(a)
		}
		if err := h.localHandler.Handle(ctx, localRecord); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Handle remotely
	if h.logger != nil && r.Level >= h.remoteLevel {
		logAttrs := h.toLogAttrs(attrs, traceAttrs)

		switch r.Level {
		case slog.LevelDebug:
			h.logger.Debug(ctx, r.Message, logAttrs...)
		case slog.LevelInfo:
			h.logger.Info(ctx, r.Message, logAttrs...)
		case slog.LevelWarn:
			h.logger.Warn(ctx, r.Message, logAttrs...)
		case slog.LevelError:
			h.logger.Error(ctx, r.Message, logAttrs...)
		default:
			// For levels between standard levels, use closest
			if r.Level < slog.LevelInfo {
				h.logger.Debug(ctx, r.Message, logAttrs...)
			} else if r.Level < slog.LevelWarn {
				h.logger.Info(ctx, r.Message, logAttrs...)
			} else if r.Level < slog.LevelError {
				h.logger.Warn(ctx, r.Message, logAttrs...)
			} else {
				h.logger.Error(ctx, r.Message, logAttrs...)
			}
		}
	}

	return firstErr
}

// WithAttrs implements slog.Handler.
func (h *LoggerSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)
	if h2.localHandler != nil {
		h2.localHandler = h2.localHandler.WithAttrs(attrs)
	}
	return h2
}

// WithGroup implements slog.Handler.
func (h *LoggerSlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	if h2.localHandler != nil {
		h2.localHandler = h2.localHandler.WithGroup(name)
	}
	return h2
}

func (h *LoggerSlogHandler) clone() *LoggerSlogHandler {
	return &LoggerSlogHandler{
		logger:              h.logger,
		localHandler:        h.localHandler,
		remoteLevel:         h.remoteLevel,
		includeTraceContext: h.includeTraceContext,
		traceIDKey:          h.traceIDKey,
		spanIDKey:           h.spanIDKey,
		groups:              slices.Clone(h.groups),
		attrs:               slices.Clone(h.attrs),
	}
}

func (h *LoggerSlogHandler) collectAttrs(r slog.Record) []slog.Attr {
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	return attrs
}

func (h *LoggerSlogHandler) toLogAttrs(slogAttrs []slog.Attr, traceAttrs []slog.Attr) []LogAttribute {
	all := append(h.attrs, slogAttrs...)
	all = append(all, traceAttrs...)

	result := make([]LogAttribute, 0, len(all))
	for _, a := range all {
		result = append(result, LogAttribute{
			Key:   h.attrKey(a.Key),
			Value: a.Value.Any(),
		})
	}
	return result
}

func (h *LoggerSlogHandler) attrKey(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	// Flatten groups: group1.group2.key
	prefix := ""
	for _, g := range h.groups {
		prefix += g + "."
	}
	return prefix + key
}

// traceContext holds extracted trace information.
type traceContext struct {
	TraceID string
	SpanID  string
}

// extractTraceContext extracts trace context from OTel span in context.
func extractTraceContext(ctx context.Context) traceContext {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return traceContext{}
	}

	sc := span.SpanContext()
	if !sc.IsValid() {
		return traceContext{}
	}

	return traceContext{
		TraceID: sc.TraceID().String(),
		SpanID:  sc.SpanID().String(),
	}
}

// NoopSlogHandler returns an slog.Handler that discards all logs.
func NoopSlogHandler() slog.Handler {
	return &noopSlogHandler{}
}

type noopSlogHandler struct{}

func (h *noopSlogHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (h *noopSlogHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h *noopSlogHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return h }
func (h *noopSlogHandler) WithGroup(_ string) slog.Handler               { return h }
