package middleware

import (
	"context"
	"encoding/json"
	"time"

	"github.com/plexusone/omniobserve/agentops"
)

// ToolCallConfig configures a tool call invocation.
type ToolCallConfig struct {
	// ToolType categorizes the tool (e.g., "search", "database", "api").
	ToolType string

	// Input is the input data for the tool call.
	Input map[string]any

	// HTTPMethod is the HTTP method if this is an HTTP-based tool.
	HTTPMethod string

	// HTTPURL is the URL if this is an HTTP-based tool.
	HTTPURL string

	// Store is the agentops store. If nil, attempts to get from context.
	Store agentops.Store
}

// ToolCallOption configures a tool call.
type ToolCallOption func(*ToolCallConfig)

// WithToolType sets the tool type.
func WithToolType(toolType string) ToolCallOption {
	return func(c *ToolCallConfig) {
		c.ToolType = toolType
	}
}

// WithToolInput sets the tool input data.
func WithToolInput(input map[string]any) ToolCallOption {
	return func(c *ToolCallConfig) {
		c.Input = input
	}
}

// WithToolHTTP sets HTTP method and URL for HTTP-based tools.
func WithToolHTTP(method, url string) ToolCallOption {
	return func(c *ToolCallConfig) {
		c.HTTPMethod = method
		c.HTTPURL = url
	}
}

// WithToolStore sets the store for the tool call.
func WithToolStore(store agentops.Store) ToolCallOption {
	return func(c *ToolCallConfig) {
		c.Store = store
	}
}

// ToolCallResult holds the result of a tool call for recording.
type ToolCallResult struct {
	Output         map[string]any
	HTTPStatusCode int
	ResponseSize   int
	Error          error
}

// ToolCall wraps a function as an instrumented tool call.
// It automatically records the tool invocation with timing, status, and errors.
//
// Usage:
//
//	result, err := ToolCall(ctx, "web_search", func() (any, error) {
//	    return searchService.Search(query)
//	}, WithToolType("search"), WithToolInput(map[string]any{"query": query}))
func ToolCall[T any](ctx context.Context, toolName string, fn func() (T, error), opts ...ToolCallOption) (T, error) {
	cfg := &ToolCallConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	store := cfg.Store
	if store == nil {
		store = StoreFromContext(ctx)
	}

	// Get agent and task info from context
	agent := AgentFromContext(ctx)
	agentID := agent.ID
	if agentID == "" {
		agentID = "unknown"
	}
	taskID := TaskIDFromContext(ctx)

	var zero T

	// If no store, just execute the function
	if store == nil {
		return fn()
	}

	// Build tool invocation options
	toolOpts := []agentops.ToolInvocationOption{}
	if cfg.ToolType != "" {
		toolOpts = append(toolOpts, agentops.WithToolType(cfg.ToolType))
	}
	if cfg.Input != nil {
		toolOpts = append(toolOpts, agentops.WithToolInput(cfg.Input))
		// Calculate request size
		if inputBytes, err := json.Marshal(cfg.Input); err == nil {
			toolOpts = append(toolOpts, agentops.WithToolRequestSize(len(inputBytes)))
		}
	}
	if cfg.HTTPMethod != "" {
		toolOpts = append(toolOpts, agentops.WithToolHTTPMethod(cfg.HTTPMethod))
	}
	if cfg.HTTPURL != "" {
		toolOpts = append(toolOpts, agentops.WithToolHTTPURL(cfg.HTTPURL))
	}

	// Record the tool invocation
	invocation, err := store.RecordToolInvocation(ctx, taskID, agentID, toolName, toolOpts...)
	if err != nil {
		// Log error but continue with function execution
		return fn()
	}

	startTime := time.Now()

	// Execute the function
	result, fnErr := fn()

	duration := time.Since(startTime).Milliseconds()

	// Build completion options
	completeOpts := []agentops.ToolInvocationCompleteOption{
		agentops.WithToolCompleteDuration(duration),
	}

	// Try to calculate response size
	if fnErr == nil {
		if outputBytes, err := json.Marshal(result); err == nil {
			completeOpts = append(completeOpts, agentops.WithToolCompleteResponseSize(len(outputBytes)))
		}
	}

	// Complete or fail the tool invocation
	if fnErr != nil {
		_ = store.UpdateToolInvocation(ctx, invocation.ID,
			agentops.WithToolUpdateStatus(agentops.StatusFailed),
			agentops.WithToolUpdateDuration(duration),
			agentops.WithToolUpdateError("error", fnErr.Error()),
		)
		return zero, fnErr
	}

	_ = store.CompleteToolInvocation(ctx, invocation.ID, completeOpts...)

	return result, nil
}

// ToolCallVoid wraps a void function as an instrumented tool call.
// Use this when the tool doesn't return a value.
//
// Usage:
//
//	err := ToolCallVoid(ctx, "send_notification", func() error {
//	    return notificationService.Send(message)
//	}, WithToolType("notification"))
func ToolCallVoid(ctx context.Context, toolName string, fn func() error, opts ...ToolCallOption) error {
	_, err := ToolCall(ctx, toolName, func() (struct{}, error) {
		return struct{}{}, fn()
	}, opts...)
	return err
}

// HTTPToolCall wraps an HTTP-based tool call with automatic instrumentation.
// It records HTTP method, URL, status code, and response size.
//
// Usage:
//
//	resp, err := HTTPToolCall(ctx, "external_api", func() (*http.Response, error) {
//	    return http.Get("https://api.example.com/data")
//	})
func HTTPToolCall(ctx context.Context, toolName string, method string, url string, fn func() (HTTPToolResponse, error), opts ...ToolCallOption) (HTTPToolResponse, error) {
	opts = append(opts, WithToolHTTP(method, url))
	return ToolCall(ctx, toolName, func() (HTTPToolResponse, error) {
		resp, err := fn()
		return resp, err
	}, opts...)
}

// HTTPToolResponse represents the response from an HTTP tool call.
type HTTPToolResponse struct {
	StatusCode   int
	Body         []byte
	Headers      map[string][]string
	ResponseSize int
}

// SearchToolCall is a convenience wrapper for search tool calls.
//
// Usage:
//
//	results, err := SearchToolCall(ctx, "web_search", query, func() ([]SearchResult, error) {
//	    return searchService.Search(query)
//	})
func SearchToolCall[T any](ctx context.Context, toolName string, query string, fn func() (T, error), opts ...ToolCallOption) (T, error) {
	opts = append(opts,
		WithToolType("search"),
		WithToolInput(map[string]any{"query": query}),
	)
	return ToolCall(ctx, toolName, fn, opts...)
}

// DatabaseToolCall is a convenience wrapper for database tool calls.
//
// Usage:
//
//	rows, err := DatabaseToolCall(ctx, "user_lookup", "SELECT * FROM users WHERE id = ?", func() ([]User, error) {
//	    return db.Query(query, userID)
//	})
func DatabaseToolCall[T any](ctx context.Context, toolName string, query string, fn func() (T, error), opts ...ToolCallOption) (T, error) {
	opts = append(opts,
		WithToolType("database"),
		WithToolInput(map[string]any{"query": query}),
	)
	return ToolCall(ctx, toolName, fn, opts...)
}

// APIToolCall is a convenience wrapper for external API tool calls.
//
// Usage:
//
//	data, err := APIToolCall(ctx, "weather_api", "GET", "https://api.weather.com/current", func() (WeatherData, error) {
//	    return weatherClient.GetCurrent(location)
//	})
func APIToolCall[T any](ctx context.Context, toolName string, method string, url string, fn func() (T, error), opts ...ToolCallOption) (T, error) {
	opts = append(opts,
		WithToolType("api"),
		WithToolHTTP(method, url),
	)
	return ToolCall(ctx, toolName, fn, opts...)
}

// RetryToolCall wraps a tool call with automatic retry tracking.
// It increments the retry count on each attempt.
//
// Usage:
//
//	result, err := RetryToolCall(ctx, "flaky_api", 3, func(attempt int) (Data, error) {
//	    return flakyAPI.Call()
//	})
func RetryToolCall[T any](ctx context.Context, toolName string, maxRetries int, fn func(attempt int) (T, error), opts ...ToolCallOption) (T, error) {
	var lastErr error
	var zero T

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := ToolCall(ctx, toolName, func() (T, error) {
			return fn(attempt)
		}, opts...)

		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt == maxRetries {
			break
		}
	}

	return zero, lastErr
}
