// Package agentops provides observability for multi-agent systems.
// It tracks agent lifecycle, workflows, handoffs, and tool invocations,
// complementing llmops (LLM-specific) and observops (infrastructure).
//
// # Architecture
//
// AgentOps operates at a higher level than LLMOps:
//
//	┌─────────────────────────────────────────────────────┐
//	│                     AgentOps                        │
//	│  (Workflows, Tasks, Handoffs, Tool Invocations)     │
//	│                                                     │
//	│  ┌─────────────────────────────────────────────┐    │
//	│  │                  LLMOps                     │    │
//	│  │  (LLM calls, prompts, tokens, completions)  │    │
//	│  └─────────────────────────────────────────────┘    │
//	└─────────────────────────────────────────────────────┘
//
// # Quick Start
//
//	import (
//		"github.com/plexusone/omniobserve/agentops"
//		_ "github.com/plexusone/omniobserve/agentops/postgres"
//	)
//
//	store, err := agentops.Open("postgres",
//		agentops.WithDSN("postgres://user:pass@localhost/db"),
//	)
//	defer store.Close()
//
//	// Start a workflow
//	workflow, err := store.StartWorkflow(ctx, "statistics-extraction",
//		agentops.WithWorkflowInput(map[string]any{"topic": "GDP"}),
//	)
//
//	// Record a task
//	task, err := store.StartTask(ctx, workflow.ID, "synthesis-agent", "extract",
//		agentops.WithTaskType("extraction"),
//	)
//
//	// Complete the task
//	err = store.CompleteTask(ctx, task.ID,
//		agentops.WithTaskOutput(map[string]any{"stats": results}),
//	)
//
// # Event Types
//
// AgentOps tracks several types of events:
//
//   - Workflows: End-to-end processing sessions
//   - Tasks: Individual agent tasks within a workflow
//   - Handoffs: Agent-to-agent communication
//   - Tool Invocations: External tool/API calls
//   - Events: Generic extensible events
package agentops

import (
	"context"
	"time"
)

// Store is the main interface for AgentOps storage backends.
type Store interface {
	WorkflowStore
	TaskStore
	HandoffStore
	ToolInvocationStore
	EventStore

	// Close closes the store connection.
	Close() error

	// Ping checks the connection to the store.
	Ping(ctx context.Context) error
}

// WorkflowStore handles workflow operations.
type WorkflowStore interface {
	// StartWorkflow creates a new workflow.
	StartWorkflow(ctx context.Context, name string, opts ...WorkflowOption) (*Workflow, error)

	// GetWorkflow retrieves a workflow by ID.
	GetWorkflow(ctx context.Context, id string) (*Workflow, error)

	// UpdateWorkflow updates workflow fields.
	UpdateWorkflow(ctx context.Context, id string, opts ...WorkflowUpdateOption) error

	// CompleteWorkflow marks a workflow as completed.
	CompleteWorkflow(ctx context.Context, id string, opts ...WorkflowCompleteOption) error

	// FailWorkflow marks a workflow as failed.
	FailWorkflow(ctx context.Context, id string, err error) error

	// ListWorkflows lists workflows with optional filters.
	ListWorkflows(ctx context.Context, opts ...ListOption) ([]*Workflow, error)
}

// TaskStore handles agent task operations.
type TaskStore interface {
	// StartTask creates a new task.
	StartTask(ctx context.Context, workflowID, agentID, name string, opts ...TaskOption) (*Task, error)

	// GetTask retrieves a task by ID.
	GetTask(ctx context.Context, id string) (*Task, error)

	// UpdateTask updates task fields.
	UpdateTask(ctx context.Context, id string, opts ...TaskUpdateOption) error

	// CompleteTask marks a task as completed.
	CompleteTask(ctx context.Context, id string, opts ...TaskCompleteOption) error

	// FailTask marks a task as failed.
	FailTask(ctx context.Context, id string, err error, opts ...TaskFailOption) error

	// ListTasks lists tasks with optional filters.
	ListTasks(ctx context.Context, opts ...ListOption) ([]*Task, error)
}

// HandoffStore handles agent handoff operations.
type HandoffStore interface {
	// RecordHandoff records a handoff between agents.
	RecordHandoff(ctx context.Context, fromAgentID, toAgentID string, opts ...HandoffOption) (*Handoff, error)

	// GetHandoff retrieves a handoff by ID.
	GetHandoff(ctx context.Context, id string) (*Handoff, error)

	// UpdateHandoff updates handoff status.
	UpdateHandoff(ctx context.Context, id string, opts ...HandoffUpdateOption) error

	// ListHandoffs lists handoffs with optional filters.
	ListHandoffs(ctx context.Context, opts ...ListOption) ([]*Handoff, error)
}

// ToolInvocationStore handles tool invocation operations.
type ToolInvocationStore interface {
	// RecordToolInvocation records a tool invocation.
	RecordToolInvocation(ctx context.Context, taskID, agentID, toolName string, opts ...ToolInvocationOption) (*ToolInvocation, error)

	// GetToolInvocation retrieves a tool invocation by ID.
	GetToolInvocation(ctx context.Context, id string) (*ToolInvocation, error)

	// UpdateToolInvocation updates a tool invocation.
	UpdateToolInvocation(ctx context.Context, id string, opts ...ToolInvocationUpdateOption) error

	// CompleteToolInvocation marks a tool invocation as completed.
	CompleteToolInvocation(ctx context.Context, id string, opts ...ToolInvocationCompleteOption) error

	// ListToolInvocations lists tool invocations with optional filters.
	ListToolInvocations(ctx context.Context, opts ...ListOption) ([]*ToolInvocation, error)
}

// EventStore handles generic event operations.
type EventStore interface {
	// EmitEvent emits a generic event.
	EmitEvent(ctx context.Context, eventType string, opts ...EventOption) (*Event, error)

	// GetEvent retrieves an event by ID.
	GetEvent(ctx context.Context, id string) (*Event, error)

	// ListEvents lists events with optional filters.
	ListEvents(ctx context.Context, opts ...ListOption) ([]*Event, error)
}

// Workflow represents an end-to-end workflow/session.
// JSON field names follow OpenTelemetry Semantic Conventions for Agentic AI.
type Workflow struct {
	ID                 string         `json:"gen_ai.agent.workflow.id"`
	Name               string         `json:"gen_ai.agent.workflow.name"`
	Status             string         `json:"gen_ai.agent.workflow.status"`
	TraceID            string         `json:"trace_id,omitempty"`
	ParentWorkflowID   string         `json:"gen_ai.agent.workflow.parent_id,omitempty"`
	Initiator          string         `json:"gen_ai.agent.workflow.initiator,omitempty"`
	Input              map[string]any `json:"input,omitempty"`
	Output             map[string]any `json:"output,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	TaskCount          int            `json:"gen_ai.agent.workflow.task.count"`
	CompletedTaskCount int            `json:"gen_ai.agent.workflow.task.completed_count"`
	FailedTaskCount    int            `json:"gen_ai.agent.workflow.task.failed_count"`
	TotalCostUSD       float64        `json:"gen_ai.usage.cost"`
	TotalTokens        int            `json:"gen_ai.usage.total_tokens"`
	DurationMs         int64          `json:"gen_ai.agent.workflow.duration,omitempty"`
	ErrorMessage       string         `json:"error.message,omitempty"`
	StartedAt          time.Time      `json:"started_at"`
	EndedAt            *time.Time     `json:"ended_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// Task represents an agent task.
// JSON field names follow OpenTelemetry Semantic Conventions for Agentic AI.
type Task struct {
	ID               string         `json:"gen_ai.agent.task.id"`
	WorkflowID       string         `json:"gen_ai.agent.workflow.id,omitempty"`
	AgentID          string         `json:"gen_ai.agent.id"`
	AgentType        string         `json:"gen_ai.agent.type,omitempty"`
	TaskType         string         `json:"gen_ai.agent.task.type"`
	Name             string         `json:"gen_ai.agent.task.name"`
	Status           string         `json:"gen_ai.agent.task.status"`
	TraceID          string         `json:"trace_id,omitempty"`
	SpanID           string         `json:"span_id,omitempty"`
	ParentSpanID     string         `json:"gen_ai.agent.task.parent_id,omitempty"`
	Input            map[string]any `json:"input,omitempty"`
	Output           map[string]any `json:"output,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	LLMCallCount     int            `json:"gen_ai.agent.task.llm.call_count"`
	ToolCallCount    int            `json:"gen_ai.agent.task.tool_call.count"`
	RetryCount       int            `json:"gen_ai.agent.task.retry_count"`
	TokensPrompt     int            `json:"gen_ai.usage.input_tokens"`
	TokensCompletion int            `json:"gen_ai.usage.output_tokens"`
	TokensTotal      int            `json:"gen_ai.usage.total_tokens"`
	CostUSD          float64        `json:"gen_ai.usage.cost"`
	DurationMs       int64          `json:"gen_ai.agent.task.duration,omitempty"`
	ErrorType        string         `json:"gen_ai.agent.task.error.type,omitempty"`
	ErrorMessage     string         `json:"gen_ai.agent.task.error.message,omitempty"`
	StartedAt        time.Time      `json:"started_at"`
	EndedAt          *time.Time     `json:"ended_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// Handoff represents an agent-to-agent handoff.
// JSON field names follow OpenTelemetry Semantic Conventions for Agentic AI.
type Handoff struct {
	ID               string         `json:"gen_ai.agent.handoff.id"`
	WorkflowID       string         `json:"gen_ai.agent.workflow.id,omitempty"`
	FromAgentID      string         `json:"gen_ai.agent.handoff.from.agent.id"`
	FromAgentType    string         `json:"gen_ai.agent.handoff.from.agent.type,omitempty"`
	ToAgentID        string         `json:"gen_ai.agent.handoff.to.agent.id"`
	ToAgentType      string         `json:"gen_ai.agent.handoff.to.agent.type,omitempty"`
	HandoffType      string         `json:"gen_ai.agent.handoff.type"`
	Status           string         `json:"gen_ai.agent.handoff.status"`
	TraceID          string         `json:"trace_id,omitempty"`
	FromTaskID       string         `json:"gen_ai.agent.handoff.from.task.id,omitempty"`
	ToTaskID         string         `json:"gen_ai.agent.handoff.to.task.id,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	PayloadSizeBytes int            `json:"gen_ai.agent.handoff.payload.size"`
	LatencyMs        int64          `json:"gen_ai.agent.handoff.latency,omitempty"`
	ErrorMessage     string         `json:"gen_ai.agent.handoff.error.message,omitempty"`
	InitiatedAt      time.Time      `json:"initiated_at"`
	AcceptedAt       *time.Time     `json:"accepted_at,omitempty"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// ToolInvocation represents a tool/function call.
// JSON field names follow OpenTelemetry Semantic Conventions for Agentic AI.
// Uses gen_ai.agent.tool_call.* to avoid collision with OTel's gen_ai.tool.* (definitions).
type ToolInvocation struct {
	ID                string         `json:"gen_ai.agent.tool_call.id"`
	TaskID            string         `json:"gen_ai.agent.task.id,omitempty"`
	AgentID           string         `json:"gen_ai.agent.id"`
	ToolName          string         `json:"gen_ai.agent.tool_call.name"`
	ToolType          string         `json:"gen_ai.agent.tool_call.type,omitempty"`
	Status            string         `json:"gen_ai.agent.tool_call.status"`
	TraceID           string         `json:"trace_id,omitempty"`
	SpanID            string         `json:"span_id,omitempty"`
	Input             map[string]any `json:"input,omitempty"`
	Output            map[string]any `json:"output,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	HTTPMethod        string         `json:"gen_ai.agent.tool_call.http.method,omitempty"`
	HTTPURL           string         `json:"gen_ai.agent.tool_call.http.url,omitempty"`
	HTTPStatusCode    int            `json:"gen_ai.agent.tool_call.http.status_code,omitempty"`
	DurationMs        int64          `json:"gen_ai.agent.tool_call.duration,omitempty"`
	RequestSizeBytes  int            `json:"gen_ai.agent.tool_call.request.size"`
	ResponseSizeBytes int            `json:"gen_ai.agent.tool_call.response.size"`
	RetryCount        int            `json:"gen_ai.agent.tool_call.retry_count"`
	ErrorType         string         `json:"gen_ai.agent.tool_call.error.type,omitempty"`
	ErrorMessage      string         `json:"gen_ai.agent.tool_call.error.message,omitempty"`
	StartedAt         time.Time      `json:"started_at"`
	EndedAt           *time.Time     `json:"ended_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// Event represents a generic event.
// JSON field names follow OpenTelemetry Semantic Conventions for Agentic AI.
type Event struct {
	ID            string         `json:"gen_ai.agent.event.id"`
	EventType     string         `json:"gen_ai.agent.event.name"`
	EventCategory string         `json:"gen_ai.agent.event.category"`
	WorkflowID    string         `json:"gen_ai.agent.workflow.id,omitempty"`
	TaskID        string         `json:"gen_ai.agent.task.id,omitempty"`
	AgentID       string         `json:"gen_ai.agent.id,omitempty"`
	TraceID       string         `json:"trace_id,omitempty"`
	SpanID        string         `json:"span_id,omitempty"`
	Severity      string         `json:"gen_ai.agent.event.severity"`
	Data          map[string]any `json:"data,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	Source        string         `json:"gen_ai.agent.event.source,omitempty"`
	Version       string         `json:"event.version"`
	Timestamp     time.Time      `json:"timestamp"`
	CreatedAt     time.Time      `json:"created_at"`
}

// Status constants
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Handoff type constants
const (
	HandoffTypeRequest   = "request"
	HandoffTypeResponse  = "response"
	HandoffTypeBroadcast = "broadcast"
	HandoffTypeDelegate  = "delegate"
)

// Event category constants
const (
	EventCategoryAgent    = "agent"
	EventCategoryWorkflow = "workflow"
	EventCategoryTool     = "tool"
	EventCategoryDomain   = "domain"
	EventCategorySystem   = "system"
)

// Severity constants
const (
	SeverityDebug = "debug"
	SeverityInfo  = "info"
	SeverityWarn  = "warn"
	SeverityError = "error"
)

// Common event types
const (
	EventTypeTaskStarted       = "gen_ai.agent.task.started"
	EventTypeTaskCompleted     = "gen_ai.agent.task.completed"
	EventTypeTaskFailed        = "gen_ai.agent.task.failed"
	EventTypeHandoffInitiated  = "gen_ai.agent.handoff.initiated"
	EventTypeHandoffCompleted  = "gen_ai.agent.handoff.completed"
	EventTypeToolCallInvoked   = "gen_ai.agent.tool_call.invoked"
	EventTypeToolCallCompleted = "gen_ai.agent.tool_call.completed"
	EventTypeWorkflowStarted   = "gen_ai.agent.workflow.started"
	EventTypeWorkflowCompleted = "gen_ai.agent.workflow.completed"
	EventTypeRetryAttempted    = "gen_ai.agent.retry.attempted"
)
