package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/plexusone/omniobserve/agentops"
)

// AgentHandlerConfig configures the agent HTTP handler middleware.
type AgentHandlerConfig struct {
	// AgentID is the unique identifier of the agent.
	AgentID string

	// AgentType categorizes the agent's role (e.g., "synthesis", "research").
	AgentType string

	// AgentName is the human-readable name of the agent.
	AgentName string

	// DefaultTaskType is the default task type if not specified in the request.
	DefaultTaskType string

	// TaskNameFromPath uses the URL path as the task name.
	TaskNameFromPath bool

	// Store is the agentops store. If nil, attempts to get from context.
	Store agentops.Store
}

// AgentHandler returns HTTP middleware that instruments handler as an agent task.
// It automatically:
//   - Creates a task when a request arrives
//   - Extracts workflow ID from headers if present
//   - Records duration, status, and errors
//   - Completes or fails the task based on response status
func AgentHandler(cfg AgentHandlerConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			store := cfg.Store
			if store == nil {
				store = StoreFromContext(r.Context())
			}

			// If no store available, just pass through
			if store == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Extract workflow ID from headers if present
			workflowID := r.Header.Get(HeaderWorkflowID)

			// Determine task name
			taskName := r.URL.Path
			if !cfg.TaskNameFromPath {
				taskName = r.Method + " " + r.URL.Path
			}

			// Build task options
			taskOpts := []agentops.TaskOption{}
			if cfg.DefaultTaskType != "" {
				taskOpts = append(taskOpts, agentops.WithTaskType(cfg.DefaultTaskType))
			}
			if cfg.AgentType != "" {
				taskOpts = append(taskOpts, agentops.WithAgentType(cfg.AgentType))
			}

			// Extract trace ID from headers
			if traceID := r.Header.Get(HeaderTraceID); traceID != "" {
				taskOpts = append(taskOpts, agentops.WithTaskTraceID(traceID))
			}

			// Start the task
			task, err := store.StartTask(ctx, workflowID, cfg.AgentID, taskName, taskOpts...)
			if err != nil {
				// Log error but don't fail the request
				next.ServeHTTP(w, r)
				return
			}

			startTime := time.Now()

			// Add task and agent info to context
			ctx = WithTask(ctx, task)
			ctx = WithAgent(ctx, AgentInfo{
				ID:   cfg.AgentID,
				Type: cfg.AgentType,
				Name: cfg.AgentName,
			})
			ctx = WithStore(ctx, store)

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Serve the request
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Calculate duration
			duration := time.Since(startTime).Milliseconds()

			// Complete or fail the task based on status code
			if rw.statusCode >= 400 {
				_ = store.FailTask(ctx, task.ID, &httpError{statusCode: rw.statusCode},
					agentops.WithTaskFailDuration(duration),
				)
			} else {
				_ = store.CompleteTask(ctx, task.ID,
					agentops.WithTaskCompleteDuration(duration),
				)
			}
		})
	}
}

// AgentHandlerFunc is a convenience wrapper for AgentHandler with http.HandlerFunc.
func AgentHandlerFunc(cfg AgentHandlerConfig, handler http.HandlerFunc) http.Handler {
	return AgentHandler(cfg)(handler)
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type httpError struct {
	statusCode int
}

func (e *httpError) Error() string {
	return http.StatusText(e.statusCode)
}

// AgentClientConfig configures the agent HTTP client for tracking handoffs.
type AgentClientConfig struct {
	// FromAgentID is the ID of the agent making the call.
	FromAgentID string

	// FromAgentType is the type of the agent making the call.
	FromAgentType string

	// Store is the agentops store. If nil, attempts to get from context.
	Store agentops.Store
}

// AgentClient wraps an http.Client to track inter-agent calls as handoffs.
type AgentClient struct {
	client *http.Client
	cfg    AgentClientConfig
}

// NewAgentClient creates a new AgentClient that tracks handoffs.
func NewAgentClient(client *http.Client, cfg AgentClientConfig) *AgentClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &AgentClient{
		client: client,
		cfg:    cfg,
	}
}

// Do executes an HTTP request and records it as a handoff to another agent.
func (c *AgentClient) Do(ctx context.Context, req *http.Request, toAgentID string) (*http.Response, error) {
	store := c.cfg.Store
	if store == nil {
		store = StoreFromContext(ctx)
	}

	// Propagate context headers
	if workflowID := WorkflowIDFromContext(ctx); workflowID != "" {
		req.Header.Set(HeaderWorkflowID, workflowID)
	}
	if taskID := TaskIDFromContext(ctx); taskID != "" {
		req.Header.Set(HeaderTaskID, taskID)
	}
	req.Header.Set(HeaderAgentID, c.cfg.FromAgentID)

	// If no store, just execute the request
	if store == nil {
		return c.client.Do(req) //nolint:gosec // G704: URL is from inter-agent communication, not user input
	}

	// Calculate payload size
	var payloadSize int
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			payloadSize = len(body)
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
	}

	// Record the handoff
	handoffOpts := []agentops.HandoffOption{
		agentops.WithHandoffType(agentops.HandoffTypeRequest),
		agentops.WithHandoffPayloadSize(payloadSize),
	}
	if c.cfg.FromAgentType != "" {
		handoffOpts = append(handoffOpts, agentops.WithFromAgentType(c.cfg.FromAgentType))
	}
	if workflowID := WorkflowIDFromContext(ctx); workflowID != "" {
		handoffOpts = append(handoffOpts, agentops.WithHandoffWorkflowID(workflowID))
	}
	if taskID := TaskIDFromContext(ctx); taskID != "" {
		handoffOpts = append(handoffOpts, agentops.WithFromTaskID(taskID))
	}

	handoff, err := store.RecordHandoff(ctx, c.cfg.FromAgentID, toAgentID, handoffOpts...)
	if err != nil {
		// Log error but continue with request
		return c.client.Do(req) //nolint:gosec // G704: URL is from inter-agent communication, not user input
	}

	startTime := time.Now()

	// Execute the request
	resp, err := c.client.Do(req) //nolint:gosec // G704: URL is from inter-agent communication, not user input

	latency := time.Since(startTime).Milliseconds()

	// Update handoff status
	if err != nil {
		_ = store.UpdateHandoff(ctx, handoff.ID,
			agentops.WithHandoffUpdateStatus(agentops.StatusFailed),
			agentops.WithHandoffUpdateLatency(latency),
			agentops.WithHandoffUpdateError(err.Error()),
		)
		return resp, err
	}

	status := agentops.StatusCompleted
	if resp.StatusCode >= 400 {
		status = agentops.StatusFailed
	}

	_ = store.UpdateHandoff(ctx, handoff.ID,
		agentops.WithHandoffUpdateStatus(status),
		agentops.WithHandoffUpdateLatency(latency),
	)

	return resp, nil
}

// Get performs a GET request to another agent.
func (c *AgentClient) Get(ctx context.Context, url string, toAgentID string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req, toAgentID)
}

// Post performs a POST request to another agent.
func (c *AgentClient) Post(ctx context.Context, url string, contentType string, body io.Reader, toAgentID string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req, toAgentID)
}

// PostJSON performs a POST request with JSON content type to another agent.
func (c *AgentClient) PostJSON(ctx context.Context, url string, body io.Reader, toAgentID string) (*http.Response, error) {
	return c.Post(ctx, url, "application/json", body, toAgentID)
}
