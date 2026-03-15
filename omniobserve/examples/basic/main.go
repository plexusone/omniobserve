// Package main demonstrates basic usage of the omniobserve unified entry point.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/plexusone/omniobserve/observops/otlp" // Register OTLP provider
	"github.com/plexusone/omniobserve/omniobserve"
)

func main() {
	// Create observability instance
	obs, err := omniobserve.New("otlp",
		omniobserve.WithServiceName("example-service"),
		omniobserve.WithServiceVersion("1.0.0"),
		omniobserve.WithEndpoint("localhost:4317"),
		omniobserve.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to create observability: %v", err)
	}
	defer func() {
		if err := obs.Shutdown(context.Background()); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	// Create HTTP mux with middleware
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/", handleRoot(obs))
	mux.HandleFunc("/api/users", handleUsers(obs))
	mux.HandleFunc("/health", handleHealth)

	// Wrap with observability middleware
	handler := obs.Middleware()(mux)

	// Start server
	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Println("Starting server on :8080")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func handleRoot(obs *omniobserve.Observability) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := omniobserve.L(ctx)

		// Start a child span
		_, span := obs.StartSpan(ctx, "process-root")
		defer span.End()

		logger.Info("Processing root request")

		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		_, _ = w.Write([]byte("Hello, World!"))
	}
}

func handleUsers(obs *omniobserve.Observability) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := omniobserve.L(ctx)

		// Use Trace helper for automatic span management
		err := omniobserve.Trace(ctx, "fetch-users", func(ctx context.Context) error {
			logger.Info("Fetching users from database")
			time.Sleep(50 * time.Millisecond)
			return nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]`))
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
