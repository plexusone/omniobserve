package sloghandler

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

// FanoutHandler sends log records to multiple handlers.
type FanoutHandler struct {
	handlers []slog.Handler
	async    bool
	groups   []string
	attrs    []slog.Attr
}

// FanoutOption configures a FanoutHandler.
type FanoutOption func(*FanoutHandler)

// NewFanout creates a handler that fans out to multiple handlers.
func NewFanout(handlers []slog.Handler, opts ...FanoutOption) *FanoutHandler {
	h := &FanoutHandler{
		handlers: handlers,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// WithAsync enables asynchronous handling.
// When enabled, handlers are invoked concurrently and errors are not returned.
func WithAsync() FanoutOption {
	return func(h *FanoutHandler) {
		h.async = true
	}
}

// Enabled implements slog.Handler.
// Returns true if any underlying handler is enabled.
func (h *FanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle implements slog.Handler.
func (h *FanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.async {
		h.handleAsync(ctx, r)
		return nil
	}
	return h.handleSync(ctx, r)
}

func (h *FanoutHandler) handleSync(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (h *FanoutHandler) handleAsync(ctx context.Context, r slog.Record) {
	var wg sync.WaitGroup
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			wg.Add(1)
			go func(handler slog.Handler) {
				defer wg.Done()
				_ = handler.Handle(ctx, r) // Errors are silently ignored in async mode
			}(handler)
		}
	}
	wg.Wait()
}

// WithAttrs implements slog.Handler.
func (h *FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}

	return &FanoutHandler{
		handlers: newHandlers,
		async:    h.async,
		groups:   slices.Clone(h.groups),
		attrs:    append(slices.Clone(h.attrs), attrs...),
	}
}

// WithGroup implements slog.Handler.
func (h *FanoutHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}

	return &FanoutHandler{
		handlers: newHandlers,
		async:    h.async,
		groups:   append(slices.Clone(h.groups), name),
		attrs:    slices.Clone(h.attrs),
	}
}

// Logger returns an *slog.Logger using this handler.
func (h *FanoutHandler) Logger() *slog.Logger {
	return slog.New(h)
}

// Handlers returns the underlying handlers.
func (h *FanoutHandler) Handlers() []slog.Handler {
	return slices.Clone(h.handlers)
}

// Tee creates a simple two-way fanout handler.
func Tee(h1, h2 slog.Handler) *FanoutHandler {
	return NewFanout([]slog.Handler{h1, h2})
}

// TeeAsync creates a simple two-way async fanout handler.
func TeeAsync(h1, h2 slog.Handler) *FanoutHandler {
	return NewFanout([]slog.Handler{h1, h2}, WithAsync())
}
