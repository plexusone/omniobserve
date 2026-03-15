package sloghandler

import (
	"context"
	"log/slog"
	"slices"
)

// Handler implements slog.Handler with observability integration.
// It supports dual output (local + remote), automatic trace context injection,
// and attribute processing.
type Handler struct {
	cfg    Config
	groups []string
	attrs  []slog.Attr
}

// New creates a new Handler with the given options.
func New(opts ...Option) *Handler {
	cfg := DefaultConfig()
	ApplyOptions(&cfg, opts...)
	return &Handler{cfg: cfg}
}

// NewWithConfig creates a new Handler with the given configuration.
func NewWithConfig(cfg Config) *Handler {
	// Apply defaults for zero values
	if cfg.TraceContextExtractor == nil {
		cfg.TraceContextExtractor = DefaultTraceContextExtractor
	}
	if cfg.TraceIDKey == "" {
		cfg.TraceIDKey = "trace_id"
	}
	if cfg.SpanIDKey == "" {
		cfg.SpanIDKey = "span_id"
	}
	return &Handler{cfg: cfg}
}

// Enabled implements slog.Handler.
// Returns true if either local or remote handler would handle the record.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	localEnabled := h.cfg.LocalHandler != nil && h.cfg.LocalHandler.Enabled(ctx, level)
	remoteEnabled := h.cfg.RemoteHandler != nil && level >= h.cfg.RemoteLevel
	return localEnabled || remoteEnabled
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Extract trace context if enabled
	var traceAttrs []slog.Attr
	if h.cfg.IncludeTraceContext && h.cfg.TraceContextExtractor != nil {
		if tc := h.cfg.TraceContextExtractor(ctx); tc.IsValid() {
			traceAttrs = []slog.Attr{
				slog.String(h.cfg.TraceIDKey, tc.TraceID),
				slog.String(h.cfg.SpanIDKey, tc.SpanID),
			}
		}
	}

	// Collect all attributes
	attrs := h.collectAttrs(r)

	// Apply processors
	for _, p := range h.cfg.Processors {
		attrs = p.Process(attrs)
		if attrs == nil {
			return nil // Record dropped by processor
		}
	}

	var firstErr error

	// Handle locally
	if h.cfg.LocalHandler != nil && h.cfg.LocalHandler.Enabled(ctx, r.Level) {
		localRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

		// Add handler-level attrs with groups
		h.addGroupedAttrs(&localRecord, h.attrs)

		// Add record attrs
		for _, a := range attrs {
			localRecord.AddAttrs(a)
		}

		// Add trace context
		for _, a := range traceAttrs {
			localRecord.AddAttrs(a)
		}

		if err := h.cfg.LocalHandler.Handle(ctx, localRecord); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Handle remotely
	if h.cfg.RemoteHandler != nil && r.Level >= h.cfg.RemoteLevel {
		remoteRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

		// Add handler-level attrs with groups
		h.addGroupedAttrs(&remoteRecord, h.attrs)

		// Add record attrs
		for _, a := range attrs {
			remoteRecord.AddAttrs(a)
		}

		// Add trace context
		for _, a := range traceAttrs {
			remoteRecord.AddAttrs(a)
		}

		if err := h.cfg.RemoteHandler.Handle(ctx, remoteRecord); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)

	// Propagate to underlying handlers
	if h2.cfg.LocalHandler != nil {
		h2.cfg.LocalHandler = h2.cfg.LocalHandler.WithAttrs(attrs)
	}
	if h2.cfg.RemoteHandler != nil {
		h2.cfg.RemoteHandler = h2.cfg.RemoteHandler.WithAttrs(attrs)
	}

	return h2
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h2 := h.clone()
	h2.groups = append(h2.groups, name)

	// Propagate to underlying handlers
	if h2.cfg.LocalHandler != nil {
		h2.cfg.LocalHandler = h2.cfg.LocalHandler.WithGroup(name)
	}
	if h2.cfg.RemoteHandler != nil {
		h2.cfg.RemoteHandler = h2.cfg.RemoteHandler.WithGroup(name)
	}

	return h2
}

// Logger returns an *slog.Logger using this handler.
func (h *Handler) Logger() *slog.Logger {
	return slog.New(h)
}

// clone creates a shallow copy of the handler.
func (h *Handler) clone() *Handler {
	return &Handler{
		cfg:    h.cfg,
		groups: slices.Clone(h.groups),
		attrs:  slices.Clone(h.attrs),
	}
}

// collectAttrs collects attributes from a record.
func (h *Handler) collectAttrs(r slog.Record) []slog.Attr {
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	return attrs
}

// addGroupedAttrs adds attributes with current group nesting.
func (h *Handler) addGroupedAttrs(r *slog.Record, attrs []slog.Attr) {
	if len(h.groups) == 0 {
		for _, a := range attrs {
			r.AddAttrs(a)
		}
		return
	}

	// Wrap attrs in nested groups
	wrapped := attrs
	for i := len(h.groups) - 1; i >= 0; i-- {
		wrapped = []slog.Attr{slog.Group(h.groups[i], attrsToAny(wrapped)...)}
	}
	for _, a := range wrapped {
		r.AddAttrs(a)
	}
}

// attrsToAny converts []slog.Attr to []any for slog.Group.
func attrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, a := range attrs {
		result[i] = a
	}
	return result
}

// LocalOnly creates a handler that only outputs locally (no remote).
func LocalOnly(h slog.Handler, opts ...Option) *Handler {
	cfg := DefaultConfig()
	cfg.LocalHandler = h
	cfg.RemoteHandler = nil
	ApplyOptions(&cfg, opts...)
	return NewWithConfig(cfg)
}

// RemoteOnly creates a handler that only outputs remotely (no local).
func RemoteOnly(h slog.Handler, opts ...Option) *Handler {
	cfg := DefaultConfig()
	cfg.LocalHandler = nil
	cfg.RemoteHandler = h
	ApplyOptions(&cfg, opts...)
	return NewWithConfig(cfg)
}

// Dual creates a handler that outputs to both local and remote handlers.
func Dual(local, remote slog.Handler, opts ...Option) *Handler {
	cfg := DefaultConfig()
	cfg.LocalHandler = local
	cfg.RemoteHandler = remote
	ApplyOptions(&cfg, opts...)
	return NewWithConfig(cfg)
}
