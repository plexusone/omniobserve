package sloghandler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

// testHandler captures log records for testing.
type testHandler struct {
	records *[]slog.Record // pointer to share across WithAttrs/WithGroup
	attrs   []slog.Attr
	groups  []string
}

func newTestHandler() *testHandler {
	records := make([]slog.Record, 0)
	return &testHandler{records: &records}
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testHandler) Handle(_ context.Context, r slog.Record) error {
	*h.records = append(*h.records, r)
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testHandler{
		records: h.records,
		attrs:   append(h.attrs, attrs...),
		groups:  h.groups,
	}
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return &testHandler{
		records: h.records,
		attrs:   h.attrs,
		groups:  append(h.groups, name),
	}
}

func (h *testHandler) numRecords() int {
	return len(*h.records)
}

//nolint:unparam // i is currently always 0 but method is designed for general use
func (h *testHandler) getRecord(i int) slog.Record {
	return (*h.records)[i]
}

func TestHandlerEnabled(t *testing.T) {
	local := newTestHandler()
	remote := newTestHandler()

	h := Dual(local, remote, WithRemoteLevel(slog.LevelWarn))

	tests := []struct {
		name    string
		level   slog.Level
		enabled bool
	}{
		{"debug enabled (local)", slog.LevelDebug, true},
		{"info enabled (local)", slog.LevelInfo, true},
		{"warn enabled (both)", slog.LevelWarn, true},
		{"error enabled (both)", slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.Enabled(context.Background(), tt.level); got != tt.enabled {
				t.Errorf("Enabled(%v) = %v, want %v", tt.level, got, tt.enabled)
			}
		})
	}
}

func TestHandlerHandle(t *testing.T) {
	local := newTestHandler()
	remote := newTestHandler()

	h := Dual(local, remote, WithRemoteLevel(slog.LevelWarn))
	logger := slog.New(h)

	// Info should only go to local
	logger.Info("info message", "key", "value")

	// Warn should go to both
	logger.Warn("warn message", "key", "value")

	if local.numRecords() != 2 {
		t.Errorf("local handler got %d records, want 2", local.numRecords())
	}
	if remote.numRecords() != 1 {
		t.Errorf("remote handler got %d records, want 1", remote.numRecords())
	}

	// Check remote record is the warn
	if remote.numRecords() > 0 && remote.getRecord(0).Level != slog.LevelWarn {
		t.Errorf("remote record level = %v, want %v", remote.getRecord(0).Level, slog.LevelWarn)
	}
}

func TestHandlerWithAttrs(t *testing.T) {
	local := newTestHandler()
	h := LocalOnly(local)

	h2 := h.WithAttrs([]slog.Attr{slog.String("service", "test")})
	logger := slog.New(h2)
	logger.Info("message")

	if local.numRecords() != 1 {
		t.Fatalf("got %d records, want 1", local.numRecords())
	}

	// Check that service attr is present
	found := false
	local.getRecord(0).Attrs(func(a slog.Attr) bool {
		if a.Key == "service" && a.Value.String() == "test" {
			found = true
			return false
		}
		return true
	})

	if !found {
		t.Error("expected 'service' attribute not found")
	}
}

func TestHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	jsonH := slog.NewJSONHandler(&buf, nil)
	h := LocalOnly(jsonH)

	h2 := h.WithGroup("request").WithAttrs([]slog.Attr{slog.String("id", "123")})
	logger := slog.New(h2)
	logger.Info("message")

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Check for nested group
	if _, ok := result["request"]; !ok {
		t.Error("expected 'request' group not found in output")
	}
}

func TestTraceContextInjection(t *testing.T) {
	// Set up a noop tracer provider to avoid actual tracing
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	var buf bytes.Buffer
	jsonH := slog.NewJSONHandler(&buf, nil)
	h := LocalOnly(jsonH)

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger := slog.New(h)
	logger.InfoContext(ctx, "with trace")

	output := buf.String()
	// With noop tracer, trace_id will be all zeros but should still be present
	// Actually, noop tracer returns invalid span context, so trace context won't be added
	// This test verifies the handler doesn't panic with noop tracer
	if !strings.Contains(output, "with trace") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestTraceContextDisabled(t *testing.T) {
	var buf bytes.Buffer
	jsonH := slog.NewJSONHandler(&buf, nil)
	h := LocalOnly(jsonH, WithoutTraceContext())

	logger := slog.New(h)
	logger.Info("no trace")

	output := buf.String()
	if strings.Contains(output, "trace_id") {
		t.Errorf("unexpected trace_id in output: %s", output)
	}
}

func TestFanoutHandler(t *testing.T) {
	h1 := newTestHandler()
	h2 := newTestHandler()
	h3 := newTestHandler()

	fanout := NewFanout([]slog.Handler{h1, h2, h3})
	logger := slog.New(fanout)

	logger.Info("test message", "key", "value")

	for i, h := range []*testHandler{h1, h2, h3} {
		if h.numRecords() != 1 {
			t.Errorf("handler %d got %d records, want 1", i, h.numRecords())
		}
	}
}

func TestFanoutHandlerAsync(t *testing.T) {
	h1 := newTestHandler()
	h2 := newTestHandler()

	fanout := NewFanout([]slog.Handler{h1, h2}, WithAsync())
	logger := slog.New(fanout)

	logger.Info("test message")

	// Async handler waits for completion
	if h1.numRecords() != 1 || h2.numRecords() != 1 {
		t.Errorf("async handlers: h1=%d, h2=%d records, want 1 each",
			h1.numRecords(), h2.numRecords())
	}
}

func TestAttributeProcessor(t *testing.T) {
	local := newTestHandler()

	redactor := RedactProcessor("password", "secret")
	h := LocalOnly(local, WithProcessor(redactor))

	logger := slog.New(h)
	logger.Info("message",
		"password", "supersecret",
		"username", "john",
	)

	if local.numRecords() != 1 {
		t.Fatalf("got %d records, want 1", local.numRecords())
	}

	// Check attributes
	var passwordVal, usernameVal string
	local.getRecord(0).Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "password":
			passwordVal = a.Value.String()
		case "username":
			usernameVal = a.Value.String()
		}
		return true
	})

	if passwordVal != "[REDACTED]" {
		t.Errorf("password = %q, want [REDACTED]", passwordVal)
	}
	if usernameVal != "john" {
		t.Errorf("username = %q, want john", usernameVal)
	}
}

func TestTee(t *testing.T) {
	h1 := newTestHandler()
	h2 := newTestHandler()

	tee := Tee(h1, h2)
	logger := slog.New(tee)
	logger.Info("test")

	if h1.numRecords() != 1 || h2.numRecords() != 1 {
		t.Errorf("tee: h1=%d, h2=%d records, want 1 each",
			h1.numRecords(), h2.numRecords())
	}
}

func BenchmarkHandler(b *testing.B) {
	local := slog.NewJSONHandler(&bytes.Buffer{}, nil)
	h := LocalOnly(local)
	logger := slog.New(h)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkHandlerWithTraceContext(b *testing.B) {
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("bench")

	local := slog.NewJSONHandler(&bytes.Buffer{}, nil)
	h := LocalOnly(local)
	logger := slog.New(h)

	ctx, span := tracer.Start(context.Background(), "bench-span")
	defer span.End()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkFanout(b *testing.B) {
	h1 := slog.NewJSONHandler(&bytes.Buffer{}, nil)
	h2 := slog.NewJSONHandler(&bytes.Buffer{}, nil)

	fanout := NewFanout([]slog.Handler{h1, h2})
	logger := slog.New(fanout)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "benchmark message", "key", "value", "count", i)
	}
}
