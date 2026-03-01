package middleware

import (
	"context"
	"time"

	"github.com/plexusone/omniobserve/agentops"
)

// WorkflowConfig configures workflow creation.
type WorkflowConfig struct {
	// Initiator identifies what started the workflow (e.g., "user:123", "api_key:abc").
	Initiator string

	// ParentWorkflowID links to a parent workflow for nested workflows.
	ParentWorkflowID string

	// Input is the initial input data for the workflow.
	Input map[string]any

	// Metadata is additional metadata for the workflow.
	Metadata map[string]any

	// TraceID is an optional trace ID for distributed tracing correlation.
	TraceID string
}

// WorkflowOption configures workflow creation.
type WorkflowOption func(*WorkflowConfig)

// WithInitiator sets the workflow initiator.
func WithInitiator(initiator string) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.Initiator = initiator
	}
}

// WithParentWorkflow sets the parent workflow ID.
func WithParentWorkflow(parentID string) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.ParentWorkflowID = parentID
	}
}

// WithWorkflowInput sets the workflow input data.
func WithWorkflowInput(input map[string]any) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.Input = input
	}
}

// WithWorkflowMetadata sets the workflow metadata.
func WithWorkflowMetadata(metadata map[string]any) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.Metadata = metadata
	}
}

// WithTraceID sets the trace ID for distributed tracing.
func WithTraceID(traceID string) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.TraceID = traceID
	}
}

// StartWorkflow creates a new workflow and returns a context with the workflow attached.
// The returned context should be used for all subsequent operations within the workflow.
func StartWorkflow(ctx context.Context, store agentops.Store, name string, opts ...WorkflowOption) (context.Context, *agentops.Workflow, error) {
	cfg := &WorkflowConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build agentops options
	var agentOpts []agentops.WorkflowOption
	if cfg.Initiator != "" {
		agentOpts = append(agentOpts, agentops.WithWorkflowInitiator(cfg.Initiator))
	}
	if cfg.ParentWorkflowID != "" {
		agentOpts = append(agentOpts, agentops.WithParentWorkflowID(cfg.ParentWorkflowID))
	}
	if cfg.Input != nil {
		agentOpts = append(agentOpts, agentops.WithWorkflowInput(cfg.Input))
	}
	if cfg.Metadata != nil {
		agentOpts = append(agentOpts, agentops.WithWorkflowMetadata(cfg.Metadata))
	}
	if cfg.TraceID != "" {
		agentOpts = append(agentOpts, agentops.WithWorkflowTraceID(cfg.TraceID))
	}

	workflow, err := store.StartWorkflow(ctx, name, agentOpts...)
	if err != nil {
		return ctx, nil, err
	}

	// Attach workflow and store to context
	ctx = WithWorkflow(ctx, workflow)
	ctx = WithStore(ctx, store)

	return ctx, workflow, nil
}

// CompleteWorkflowConfig configures workflow completion.
type CompleteWorkflowConfig struct {
	Output   map[string]any
	Metadata map[string]any
}

// CompleteWorkflowOption configures workflow completion.
type CompleteWorkflowOption func(*CompleteWorkflowConfig)

// WithWorkflowOutput sets the workflow output data.
func WithWorkflowOutput(output map[string]any) CompleteWorkflowOption {
	return func(c *CompleteWorkflowConfig) {
		c.Output = output
	}
}

// CompleteWorkflow marks the workflow in context as completed.
func CompleteWorkflow(ctx context.Context, opts ...CompleteWorkflowOption) error {
	store := StoreFromContext(ctx)
	workflow := WorkflowFromContext(ctx)

	if store == nil || workflow == nil {
		return nil // No-op if not in a workflow context
	}

	cfg := &CompleteWorkflowConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var agentOpts []agentops.WorkflowCompleteOption
	if cfg.Output != nil {
		agentOpts = append(agentOpts, agentops.WithWorkflowCompleteOutput(cfg.Output))
	}

	return store.CompleteWorkflow(ctx, workflow.ID, agentOpts...)
}

// FailWorkflow marks the workflow in context as failed.
func FailWorkflow(ctx context.Context, err error) error {
	store := StoreFromContext(ctx)
	workflow := WorkflowFromContext(ctx)

	if store == nil || workflow == nil {
		return nil // No-op if not in a workflow context
	}

	return store.FailWorkflow(ctx, workflow.ID, err)
}

// WorkflowScope provides a convenient way to manage workflow lifecycle.
// It automatically completes or fails the workflow based on the returned error.
//
// Usage:
//
//	err := WorkflowScope(ctx, store, "my-workflow", func(ctx context.Context, wf *agentops.Workflow) error {
//	    // Do work...
//	    return nil
//	})
func WorkflowScope(ctx context.Context, store agentops.Store, name string, fn func(context.Context, *agentops.Workflow) error, opts ...WorkflowOption) error {
	ctx, workflow, err := StartWorkflow(ctx, store, name, opts...)
	if err != nil {
		return err
	}

	startTime := time.Now()

	if fnErr := fn(ctx, workflow); fnErr != nil {
		_ = FailWorkflow(ctx, fnErr)
		return fnErr
	}

	// Update duration on completion
	duration := time.Since(startTime).Milliseconds()
	_ = store.UpdateWorkflow(ctx, workflow.ID, agentops.WithWorkflowUpdateDuration(duration))

	return CompleteWorkflow(ctx)
}
