// Package middleware provides reusable instrumentation helpers for multi-agent systems.
//
// This package offers middleware and wrapper functions that automatically instrument
// agent operations following OpenTelemetry semantic conventions for Agentic AI.
// It is designed to minimize code changes required to add observability to existing
// agent systems.
//
// # Components
//
// The package provides four main components:
//
//   - Context Propagation: Helpers for passing workflow, task, and agent information
//     through request context and HTTP headers.
//
//   - Workflow Management: Functions for starting, completing, and failing workflows
//     with automatic context attachment.
//
//   - HTTP Middleware: Handler middleware for instrumenting agents as tasks, and
//     client middleware for tracking inter-agent calls as handoffs.
//
//   - Tool Wrappers: Generic wrappers for instrumenting tool/function invocations
//     with timing, status, and error tracking.
//
// # Quick Start
//
// Basic usage in an agent system:
//
//	// 1. Create a store
//	store, _ := agentops.Open("postgres", agentops.WithDSN(dsn))
//	defer store.Close()
//
//	// 2. Start a workflow (in orchestrator)
//	ctx, workflow, _ := middleware.StartWorkflow(ctx, store, "my-workflow",
//	    middleware.WithInitiator("user:123"),
//	)
//	defer middleware.CompleteWorkflow(ctx)
//
//	// 3. Instrument agent HTTP handlers
//	handler := middleware.AgentHandler(middleware.AgentHandlerConfig{
//	    AgentID:   "synthesis-agent",
//	    AgentType: "synthesis",
//	    Store:     store,
//	})(yourHandler)
//
//	// 4. Use instrumented client for inter-agent calls
//	client := middleware.NewAgentClient(http.DefaultClient, middleware.AgentClientConfig{
//	    FromAgentID: "orchestrator",
//	    Store:       store,
//	})
//	resp, _ := client.PostJSON(ctx, "http://synthesis:8004/extract", body, "synthesis-agent")
//
//	// 5. Wrap tool calls
//	results, _ := middleware.ToolCall(ctx, "web_search", func() ([]Result, error) {
//	    return searchService.Search(query)
//	}, middleware.WithToolType("search"))
//
// # Context Propagation
//
// The package automatically propagates observability context through:
//
//   - Go context.Context: Workflow, task, agent, and store are attached to context
//   - HTTP Headers: X-AgentOps-Workflow-ID, X-AgentOps-Task-ID, etc.
//
// This enables distributed tracing across agent boundaries:
//
//	// In agent A
//	ctx = middleware.WithWorkflow(ctx, workflow)
//
//	// Call agent B - headers automatically set
//	client.PostJSON(ctx, agentBURL, body, "agent-b")
//
//	// In agent B - workflow ID extracted from headers
//	workflowID := r.Header.Get(middleware.HeaderWorkflowID)
//
// # HTTP Middleware
//
// AgentHandler wraps HTTP handlers to automatically create tasks:
//
//	mux := http.NewServeMux()
//	mux.Handle("/extract", middleware.AgentHandlerFunc(cfg, extractHandler))
//	mux.Handle("/analyze", middleware.AgentHandlerFunc(cfg, analyzeHandler))
//
// Each request creates a task that captures:
//   - Start/end time and duration
//   - HTTP status code
//   - Success/failure status
//   - Automatic linking to parent workflow
//
// AgentClient wraps http.Client to track inter-agent calls as handoffs:
//
//	client := middleware.NewAgentClient(nil, middleware.AgentClientConfig{
//	    FromAgentID:   "orchestrator",
//	    FromAgentType: "orchestration",
//	    Store:         store,
//	})
//
//	// This call is recorded as a handoff
//	resp, err := client.PostJSON(ctx, synthesisURL, body, "synthesis-agent")
//
// # Tool Wrappers
//
// Generic tool wrapper for any function:
//
//	result, err := middleware.ToolCall(ctx, "tool_name", func() (ResultType, error) {
//	    return myTool.Execute(args)
//	}, middleware.WithToolType("api"))
//
// Convenience wrappers for common tool types:
//
//	// Search tools
//	results, _ := middleware.SearchToolCall(ctx, "web_search", query, searchFn)
//
//	// Database tools
//	rows, _ := middleware.DatabaseToolCall(ctx, "user_query", sql, queryFn)
//
//	// API tools
//	data, _ := middleware.APIToolCall(ctx, "weather_api", "GET", url, apiFn)
//
// # Workflow Scopes
//
// For simpler workflow management, use WorkflowScope:
//
//	err := middleware.WorkflowScope(ctx, store, "my-workflow",
//	    func(ctx context.Context, wf *agentops.Workflow) error {
//	        // Do work...
//	        // Workflow automatically completed on success, failed on error
//	        return nil
//	    },
//	    middleware.WithInitiator("user:123"),
//	)
//
// # Integration with OpenTelemetry
//
// The middleware uses semantic conventions from github.com/plexusone/omniobserve/semconv/agent,
// which align with OpenTelemetry's gen_ai.agent.* namespace. This enables integration with
// OpenTelemetry-compatible observability platforms.
package middleware
