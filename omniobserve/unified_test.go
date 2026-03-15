package omniobserve

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/plexusone/omniobserve/observops/otlp"
)

func TestNewWithDisabled(t *testing.T) {
	obs, err := New("otlp",
		WithServiceName("test-service"),
		WithDisabled(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = obs.Shutdown(context.Background()) }()

	if obs.Provider() == nil {
		t.Error("Provider should not be nil")
	}
	if obs.Logger() == nil {
		t.Error("Logger should not be nil")
	}
}

func TestContextFunctions(t *testing.T) {
	obs, err := New("otlp",
		WithServiceName("test-service"),
		WithDisabled(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = obs.Shutdown(context.Background()) }()

	ctx := context.Background()

	// Test ContextWithObservability
	ctx = ContextWithObservability(ctx, obs)
	if got := ObservabilityFromContext(ctx); got != obs {
		t.Error("ObservabilityFromContext should return the stored observability")
	}

	// Test shorthand
	if got := O(ctx); got != obs {
		t.Error("O() should return the stored observability")
	}

	// Test ContextWithLogger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx = ContextWithLogger(ctx, logger)
	if got := LoggerFromContext(ctx); got != logger {
		t.Error("LoggerFromContext should return the stored logger")
	}

	// Test shorthand
	if got := L(ctx); got != logger {
		t.Error("L() should return the stored logger")
	}
}

func TestMiddleware(t *testing.T) {
	obs, err := New("otlp",
		WithServiceName("test-service"),
		WithDisabled(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = obs.Shutdown(context.Background()) }()

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify observability is in context
		if O(r.Context()) == nil {
			t.Error("Observability should be in request context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with middleware
	wrapped := obs.Middleware()(handler)

	// Test request
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Note: When disabled, trace ID may be empty (noop provider)
	// The header is still set, just with empty value when disabled
}

func TestMiddlewareSkipPaths(t *testing.T) {
	obs, err := New("otlp",
		WithServiceName("test-service"),
		WithDisabled(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = obs.Shutdown(context.Background()) }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := obs.Middleware()(handler)

	// Test skipped path (health check)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Health check should not have trace ID (skipped)
	if rec.Header().Get("X-Trace-ID") != "" {
		t.Error("X-Trace-ID header should not be set for skipped paths")
	}
}

func TestTrace(t *testing.T) {
	obs, err := New("otlp",
		WithServiceName("test-service"),
		WithDisabled(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = obs.Shutdown(context.Background()) }()

	ctx := ContextWithObservability(context.Background(), obs)

	called := false
	err = Trace(ctx, "test-span", func(ctx context.Context) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Trace() error = %v", err)
	}
	if !called {
		t.Error("Trace function was not called")
	}
}

func TestTraceFunc(t *testing.T) {
	obs, err := New("otlp",
		WithServiceName("test-service"),
		WithDisabled(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = obs.Shutdown(context.Background()) }()

	ctx := ContextWithObservability(context.Background(), obs)

	result, err := TraceFunc(ctx, "test-span", func(ctx context.Context) (string, error) {
		return "hello", nil
	})

	if err != nil {
		t.Errorf("TraceFunc() error = %v", err)
	}
	if result != "hello" {
		t.Errorf("TraceFunc() result = %v, want hello", result)
	}
}

func TestAttributeHelpers(t *testing.T) {
	kv := String("key", "value")
	if kv.Key != "key" || kv.Value != "value" {
		t.Error("String() did not create correct attribute")
	}

	kv = Int("count", 42)
	if kv.Key != "count" || kv.Value != 42 {
		t.Error("Int() did not create correct attribute")
	}

	kv = Float64("rate", 3.14)
	if kv.Key != "rate" || kv.Value != 3.14 {
		t.Error("Float64() did not create correct attribute")
	}

	kv = Bool("enabled", true)
	if kv.Key != "enabled" || kv.Value != true {
		t.Error("Bool() did not create correct attribute")
	}
}
