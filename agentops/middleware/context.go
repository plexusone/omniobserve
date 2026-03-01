package middleware

import (
	"context"

	"github.com/plexusone/omniobserve/agentops"
)

// Context keys for propagating observability data through request context.
type contextKey string

const (
	workflowKey contextKey = "agentops.workflow"
	taskKey     contextKey = "agentops.task"
	agentKey    contextKey = "agentops.agent"
	storeKey    contextKey = "agentops.store"
)

// AgentInfo holds agent identification for context propagation.
type AgentInfo struct {
	ID   string
	Type string
	Name string
}

// WithWorkflow adds a workflow to the context.
func WithWorkflow(ctx context.Context, workflow *agentops.Workflow) context.Context {
	return context.WithValue(ctx, workflowKey, workflow)
}

// WorkflowFromContext retrieves the workflow from context.
func WorkflowFromContext(ctx context.Context) *agentops.Workflow {
	if v := ctx.Value(workflowKey); v != nil {
		return v.(*agentops.Workflow)
	}
	return nil
}

// WorkflowIDFromContext retrieves just the workflow ID from context.
func WorkflowIDFromContext(ctx context.Context) string {
	if wf := WorkflowFromContext(ctx); wf != nil {
		return wf.ID
	}
	return ""
}

// WithTask adds a task to the context.
func WithTask(ctx context.Context, task *agentops.Task) context.Context {
	return context.WithValue(ctx, taskKey, task)
}

// TaskFromContext retrieves the task from context.
func TaskFromContext(ctx context.Context) *agentops.Task {
	if v := ctx.Value(taskKey); v != nil {
		return v.(*agentops.Task)
	}
	return nil
}

// TaskIDFromContext retrieves just the task ID from context.
func TaskIDFromContext(ctx context.Context) string {
	if task := TaskFromContext(ctx); task != nil {
		return task.ID
	}
	return ""
}

// WithAgent adds agent info to the context.
func WithAgent(ctx context.Context, agent AgentInfo) context.Context {
	return context.WithValue(ctx, agentKey, agent)
}

// AgentFromContext retrieves agent info from context.
func AgentFromContext(ctx context.Context) AgentInfo {
	if v := ctx.Value(agentKey); v != nil {
		return v.(AgentInfo)
	}
	return AgentInfo{}
}

// WithStore adds the store to the context.
func WithStore(ctx context.Context, store agentops.Store) context.Context {
	return context.WithValue(ctx, storeKey, store)
}

// StoreFromContext retrieves the store from context.
func StoreFromContext(ctx context.Context) agentops.Store {
	if v := ctx.Value(storeKey); v != nil {
		return v.(agentops.Store)
	}
	return nil
}

// PropagationHeaders are HTTP headers used to propagate context across services.
const (
	HeaderWorkflowID = "X-AgentOps-Workflow-ID"
	HeaderTaskID     = "X-AgentOps-Task-ID"
	HeaderAgentID    = "X-AgentOps-Agent-ID"
	HeaderTraceID    = "X-AgentOps-Trace-ID"
)
