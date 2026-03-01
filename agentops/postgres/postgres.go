// Package postgres provides a PostgreSQL backend for agentops using Ent.
package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/plexusone/omniobserve/agentops"
	"github.com/plexusone/omniobserve/agentops/ent"
	"github.com/plexusone/omniobserve/agentops/ent/agentevent"
	"github.com/plexusone/omniobserve/agentops/ent/agenthandoff"
	"github.com/plexusone/omniobserve/agentops/ent/agenttask"
	"github.com/plexusone/omniobserve/agentops/ent/toolinvocation"
	"github.com/plexusone/omniobserve/agentops/ent/workflow"
)

const providerName = "postgres"

func init() {
	agentops.Register(providerName, New)
	agentops.RegisterInfo(agentops.ProviderInfo{
		Name:        providerName,
		Description: "PostgreSQL store using Ent ORM",
		Features:    []string{"transactions", "migrations", "relations"},
	})
}

// Store implements agentops.Store using PostgreSQL with Ent.
type Store struct {
	client *ent.Client
	cfg    *agentops.ClientConfig
}

// New creates a new PostgreSQL store.
func New(opts ...agentops.ClientOption) (agentops.Store, error) {
	cfg := agentops.ApplyClientOptions(opts...)

	if cfg.DSN == "" {
		return nil, agentops.ErrMissingDSN
	}

	var entOpts []ent.Option
	if cfg.Debug {
		entOpts = append(entOpts, ent.Debug())
	}

	client, err := ent.Open("postgres", cfg.DSN, entOpts...)
	if err != nil {
		return nil, agentops.WrapError(providerName, "open", err)
	}

	s := &Store{
		client: client,
		cfg:    cfg,
	}

	if cfg.AutoMigrate {
		if err := client.Schema.Create(context.Background()); err != nil {
			client.Close()
			return nil, agentops.WrapError(providerName, "migrate", err)
		}
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.client.Close()
}

// Ping checks the database connection.
func (s *Store) Ping(ctx context.Context) error {
	// Ent doesn't have a direct Ping, so we do a simple query
	_, err := s.client.Workflow.Query().Limit(1).All(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return agentops.WrapError(providerName, "ping", err)
	}
	return nil
}

// =============================================================================
// Workflow Operations
// =============================================================================

func (s *Store) StartWorkflow(ctx context.Context, name string, opts ...agentops.WorkflowOption) (*agentops.Workflow, error) {
	cfg := agentops.ApplyWorkflowOptions(opts...)

	id := uuid.New().String()
	now := time.Now()

	create := s.client.Workflow.Create().
		SetID(id).
		SetName(name).
		SetStatus(agentops.StatusRunning).
		SetStartedAt(now).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if cfg.TraceID != "" {
		create.SetTraceID(cfg.TraceID)
	}
	if cfg.ParentWorkflowID != "" {
		create.SetParentWorkflowID(cfg.ParentWorkflowID)
	}
	if cfg.Initiator != "" {
		create.SetInitiator(cfg.Initiator)
	}
	if cfg.Input != nil {
		create.SetInput(cfg.Input)
	}
	if cfg.Metadata != nil {
		create.SetMetadata(cfg.Metadata)
	}

	w, err := create.Save(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "start_workflow", err)
	}

	return entWorkflowToAgentops(w), nil
}

func (s *Store) GetWorkflow(ctx context.Context, id string) (*agentops.Workflow, error) {
	w, err := s.client.Workflow.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, agentops.ErrNotFound
		}
		return nil, agentops.WrapError(providerName, "get_workflow", err)
	}
	return entWorkflowToAgentops(w), nil
}

func (s *Store) UpdateWorkflow(ctx context.Context, id string, opts ...agentops.WorkflowUpdateOption) error {
	cfg := agentops.ApplyWorkflowUpdateOptions(opts...)

	update := s.client.Workflow.UpdateOneID(id).SetUpdatedAt(time.Now())

	if cfg.Output != nil {
		update.SetOutput(cfg.Output)
	}
	if cfg.Metadata != nil {
		update.SetMetadata(cfg.Metadata)
	}
	if cfg.AddCost > 0 {
		update.AddTotalCostUsd(cfg.AddCost)
	}
	if cfg.AddTokens > 0 {
		update.AddTotalTokens(cfg.AddTokens)
	}

	_, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "update_workflow", err)
	}
	return nil
}

func (s *Store) CompleteWorkflow(ctx context.Context, id string, opts ...agentops.WorkflowCompleteOption) error {
	cfg := agentops.ApplyWorkflowCompleteOptions(opts...)
	now := time.Now()

	// Get current workflow to calculate duration
	w, err := s.client.Workflow.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "complete_workflow", err)
	}

	if w.Status == agentops.StatusCompleted || w.Status == agentops.StatusFailed {
		return agentops.ErrAlreadyCompleted
	}

	durationMs := now.Sub(w.StartedAt).Milliseconds()

	update := s.client.Workflow.UpdateOneID(id).
		SetStatus(agentops.StatusCompleted).
		SetEndedAt(now).
		SetDurationMs(durationMs).
		SetUpdatedAt(now)

	if cfg.Output != nil {
		update.SetOutput(cfg.Output)
	}
	if cfg.Metadata != nil {
		update.SetMetadata(cfg.Metadata)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return agentops.WrapError(providerName, "complete_workflow", err)
	}
	return nil
}

func (s *Store) FailWorkflow(ctx context.Context, id string, failErr error) error {
	now := time.Now()

	w, err := s.client.Workflow.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "fail_workflow", err)
	}

	if w.Status == agentops.StatusCompleted || w.Status == agentops.StatusFailed {
		return agentops.ErrAlreadyCompleted
	}

	durationMs := now.Sub(w.StartedAt).Milliseconds()

	_, err = s.client.Workflow.UpdateOneID(id).
		SetStatus(agentops.StatusFailed).
		SetEndedAt(now).
		SetDurationMs(durationMs).
		SetErrorMessage(failErr.Error()).
		SetUpdatedAt(now).
		Save(ctx)

	if err != nil {
		return agentops.WrapError(providerName, "fail_workflow", err)
	}
	return nil
}

func (s *Store) ListWorkflows(ctx context.Context, opts ...agentops.ListOption) ([]*agentops.Workflow, error) {
	cfg := agentops.ApplyListOptions(opts...)

	query := s.client.Workflow.Query()

	if cfg.Status != "" {
		query.Where(workflow.StatusEQ(cfg.Status))
	}
	if cfg.StartTime != nil {
		query.Where(workflow.StartedAtGTE(*cfg.StartTime))
	}
	if cfg.EndTime != nil {
		query.Where(workflow.StartedAtLTE(*cfg.EndTime))
	}

	if cfg.OrderBy != "" {
		if cfg.OrderDesc {
			query.Order(ent.Desc(cfg.OrderBy))
		} else {
			query.Order(ent.Asc(cfg.OrderBy))
		}
	} else {
		query.Order(ent.Desc(workflow.FieldCreatedAt))
	}

	if cfg.Limit > 0 {
		query.Limit(cfg.Limit)
	}
	if cfg.Offset > 0 {
		query.Offset(cfg.Offset)
	}

	workflows, err := query.All(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "list_workflows", err)
	}

	result := make([]*agentops.Workflow, len(workflows))
	for i, w := range workflows {
		result[i] = entWorkflowToAgentops(w)
	}
	return result, nil
}

// =============================================================================
// Task Operations
// =============================================================================

func (s *Store) StartTask(ctx context.Context, workflowID, agentID, name string, opts ...agentops.TaskOption) (*agentops.Task, error) {
	cfg := agentops.ApplyTaskOptions(opts...)

	id := uuid.New().String()
	now := time.Now()

	create := s.client.AgentTask.Create().
		SetID(id).
		SetAgentID(agentID).
		SetName(name).
		SetStatus(agentops.StatusRunning).
		SetStartedAt(now).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if workflowID != "" {
		create.SetWorkflowID(workflowID)
		// Increment workflow task count
		_, _ = s.client.Workflow.UpdateOneID(workflowID).AddTaskCount(1).Save(ctx)
	}

	if cfg.AgentType != "" {
		create.SetAgentType(cfg.AgentType)
	}
	if cfg.TaskType != "" {
		create.SetTaskType(cfg.TaskType)
	}
	if cfg.TraceID != "" {
		create.SetTraceID(cfg.TraceID)
	}
	if cfg.SpanID != "" {
		create.SetSpanID(cfg.SpanID)
	}
	if cfg.ParentSpanID != "" {
		create.SetParentSpanID(cfg.ParentSpanID)
	}
	if cfg.Input != nil {
		create.SetInput(cfg.Input)
	}
	if cfg.Metadata != nil {
		create.SetMetadata(cfg.Metadata)
	}

	t, err := create.Save(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "start_task", err)
	}

	return entTaskToAgentops(t), nil
}

func (s *Store) GetTask(ctx context.Context, id string) (*agentops.Task, error) {
	t, err := s.client.AgentTask.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, agentops.ErrNotFound
		}
		return nil, agentops.WrapError(providerName, "get_task", err)
	}
	return entTaskToAgentops(t), nil
}

func (s *Store) UpdateTask(ctx context.Context, id string, opts ...agentops.TaskUpdateOption) error {
	cfg := agentops.ApplyTaskUpdateOptions(opts...)

	update := s.client.AgentTask.UpdateOneID(id).SetUpdatedAt(time.Now())

	if cfg.AddLLMCalls > 0 {
		update.AddLlmCallCount(cfg.AddLLMCalls)
	}
	if cfg.AddToolCalls > 0 {
		update.AddToolCallCount(cfg.AddToolCalls)
	}
	if cfg.AddRetries > 0 {
		update.AddRetryCount(cfg.AddRetries)
	}
	if cfg.AddTokens.Prompt > 0 {
		update.AddTokensPrompt(cfg.AddTokens.Prompt)
	}
	if cfg.AddTokens.Completion > 0 {
		update.AddTokensCompletion(cfg.AddTokens.Completion)
	}
	if cfg.AddTokens.Prompt > 0 || cfg.AddTokens.Completion > 0 {
		update.AddTokensTotal(cfg.AddTokens.Prompt + cfg.AddTokens.Completion)
	}
	if cfg.AddCost > 0 {
		update.AddCostUsd(cfg.AddCost)
	}
	if cfg.Metadata != nil {
		update.SetMetadata(cfg.Metadata)
	}

	_, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "update_task", err)
	}
	return nil
}

func (s *Store) CompleteTask(ctx context.Context, id string, opts ...agentops.TaskCompleteOption) error {
	cfg := agentops.ApplyTaskCompleteOptions(opts...)
	now := time.Now()

	t, err := s.client.AgentTask.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "complete_task", err)
	}

	if t.Status == agentops.StatusCompleted || t.Status == agentops.StatusFailed {
		return agentops.ErrAlreadyCompleted
	}

	durationMs := now.Sub(t.StartedAt).Milliseconds()

	update := s.client.AgentTask.UpdateOneID(id).
		SetStatus(agentops.StatusCompleted).
		SetEndedAt(now).
		SetDurationMs(durationMs).
		SetUpdatedAt(now)

	if cfg.Output != nil {
		update.SetOutput(cfg.Output)
	}
	if cfg.Metadata != nil {
		update.SetMetadata(cfg.Metadata)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return agentops.WrapError(providerName, "complete_task", err)
	}

	// Update workflow completed task count
	if t.WorkflowID != "" {
		_, _ = s.client.Workflow.UpdateOneID(t.WorkflowID).
			AddCompletedTaskCount(1).
			AddTotalCostUsd(t.CostUsd).
			AddTotalTokens(t.TokensTotal).
			Save(ctx)
	}

	return nil
}

func (s *Store) FailTask(ctx context.Context, id string, failErr error, opts ...agentops.TaskFailOption) error {
	cfg := agentops.ApplyTaskFailOptions(opts...)
	now := time.Now()

	t, err := s.client.AgentTask.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "fail_task", err)
	}

	if t.Status == agentops.StatusCompleted || t.Status == agentops.StatusFailed {
		return agentops.ErrAlreadyCompleted
	}

	durationMs := now.Sub(t.StartedAt).Milliseconds()

	update := s.client.AgentTask.UpdateOneID(id).
		SetStatus(agentops.StatusFailed).
		SetEndedAt(now).
		SetDurationMs(durationMs).
		SetErrorMessage(failErr.Error()).
		SetUpdatedAt(now)

	if cfg.ErrorType != "" {
		update.SetErrorType(cfg.ErrorType)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return agentops.WrapError(providerName, "fail_task", err)
	}

	// Update workflow failed task count
	if t.WorkflowID != "" {
		_, _ = s.client.Workflow.UpdateOneID(t.WorkflowID).
			AddFailedTaskCount(1).
			Save(ctx)
	}

	return nil
}

//nolint:dupl // Intentionally similar to ListToolInvocations, different entity types
func (s *Store) ListTasks(ctx context.Context, opts ...agentops.ListOption) ([]*agentops.Task, error) {
	cfg := agentops.ApplyListOptions(opts...)

	query := s.client.AgentTask.Query()

	if cfg.WorkflowID != "" {
		query.Where(agenttask.WorkflowIDEQ(cfg.WorkflowID))
	}
	if cfg.AgentID != "" {
		query.Where(agenttask.AgentIDEQ(cfg.AgentID))
	}
	if cfg.Status != "" {
		query.Where(agenttask.StatusEQ(cfg.Status))
	}
	if cfg.StartTime != nil {
		query.Where(agenttask.StartedAtGTE(*cfg.StartTime))
	}
	if cfg.EndTime != nil {
		query.Where(agenttask.StartedAtLTE(*cfg.EndTime))
	}

	if cfg.OrderBy != "" {
		if cfg.OrderDesc {
			query.Order(ent.Desc(cfg.OrderBy))
		} else {
			query.Order(ent.Asc(cfg.OrderBy))
		}
	} else {
		query.Order(ent.Desc(agenttask.FieldCreatedAt))
	}

	if cfg.Limit > 0 {
		query.Limit(cfg.Limit)
	}
	if cfg.Offset > 0 {
		query.Offset(cfg.Offset)
	}

	tasks, err := query.All(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "list_tasks", err)
	}

	result := make([]*agentops.Task, len(tasks))
	for i, t := range tasks {
		result[i] = entTaskToAgentops(t)
	}
	return result, nil
}

// =============================================================================
// Handoff Operations
// =============================================================================

func (s *Store) RecordHandoff(ctx context.Context, fromAgentID, toAgentID string, opts ...agentops.HandoffOption) (*agentops.Handoff, error) {
	cfg := agentops.ApplyHandoffOptions(opts...)

	id := uuid.New().String()
	now := time.Now()

	create := s.client.AgentHandoff.Create().
		SetID(id).
		SetFromAgentID(fromAgentID).
		SetToAgentID(toAgentID).
		SetStatus(agentops.StatusPending).
		SetInitiatedAt(now).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if cfg.WorkflowID != "" {
		create.SetWorkflowID(cfg.WorkflowID)
	}
	if cfg.FromAgentType != "" {
		create.SetFromAgentType(cfg.FromAgentType)
	}
	if cfg.ToAgentType != "" {
		create.SetToAgentType(cfg.ToAgentType)
	}
	if cfg.HandoffType != "" {
		create.SetHandoffType(cfg.HandoffType)
	} else {
		create.SetHandoffType(agentops.HandoffTypeRequest)
	}
	if cfg.TraceID != "" {
		create.SetTraceID(cfg.TraceID)
	}
	if cfg.FromTaskID != "" {
		create.SetFromTaskID(cfg.FromTaskID)
	}
	if cfg.ToTaskID != "" {
		create.SetToTaskID(cfg.ToTaskID)
	}
	if cfg.Payload != nil {
		create.SetPayload(cfg.Payload)
		if data, err := json.Marshal(cfg.Payload); err == nil {
			create.SetPayloadSizeBytes(len(data))
		}
	}
	if cfg.Metadata != nil {
		create.SetMetadata(cfg.Metadata)
	}

	h, err := create.Save(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "record_handoff", err)
	}

	return entHandoffToAgentops(h), nil
}

func (s *Store) GetHandoff(ctx context.Context, id string) (*agentops.Handoff, error) {
	h, err := s.client.AgentHandoff.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, agentops.ErrNotFound
		}
		return nil, agentops.WrapError(providerName, "get_handoff", err)
	}
	return entHandoffToAgentops(h), nil
}

func (s *Store) UpdateHandoff(ctx context.Context, id string, opts ...agentops.HandoffUpdateOption) error {
	cfg := agentops.ApplyHandoffUpdateOptions(opts...)
	now := time.Now()

	h, err := s.client.AgentHandoff.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "update_handoff", err)
	}

	update := s.client.AgentHandoff.UpdateOneID(id).SetUpdatedAt(now)

	if cfg.Status != "" {
		update.SetStatus(cfg.Status)
		if cfg.Status == agentops.StatusRunning {
			update.SetAcceptedAt(now)
		} else if cfg.Status == agentops.StatusCompleted || cfg.Status == agentops.StatusFailed {
			update.SetCompletedAt(now)
			latencyMs := now.Sub(h.InitiatedAt).Milliseconds()
			update.SetLatencyMs(latencyMs)
		}
	}
	if cfg.ToTaskID != "" {
		update.SetToTaskID(cfg.ToTaskID)
	}
	if cfg.ErrorMessage != "" {
		update.SetErrorMessage(cfg.ErrorMessage)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return agentops.WrapError(providerName, "update_handoff", err)
	}
	return nil
}

func (s *Store) ListHandoffs(ctx context.Context, opts ...agentops.ListOption) ([]*agentops.Handoff, error) {
	cfg := agentops.ApplyListOptions(opts...)

	query := s.client.AgentHandoff.Query()

	if cfg.WorkflowID != "" {
		query.Where(agenthandoff.WorkflowIDEQ(cfg.WorkflowID))
	}
	if cfg.AgentID != "" {
		query.Where(
			agenthandoff.Or(
				agenthandoff.FromAgentIDEQ(cfg.AgentID),
				agenthandoff.ToAgentIDEQ(cfg.AgentID),
			),
		)
	}
	if cfg.Status != "" {
		query.Where(agenthandoff.StatusEQ(cfg.Status))
	}
	if cfg.StartTime != nil {
		query.Where(agenthandoff.InitiatedAtGTE(*cfg.StartTime))
	}
	if cfg.EndTime != nil {
		query.Where(agenthandoff.InitiatedAtLTE(*cfg.EndTime))
	}

	if cfg.OrderBy != "" {
		if cfg.OrderDesc {
			query.Order(ent.Desc(cfg.OrderBy))
		} else {
			query.Order(ent.Asc(cfg.OrderBy))
		}
	} else {
		query.Order(ent.Desc(agenthandoff.FieldCreatedAt))
	}

	if cfg.Limit > 0 {
		query.Limit(cfg.Limit)
	}
	if cfg.Offset > 0 {
		query.Offset(cfg.Offset)
	}

	handoffs, err := query.All(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "list_handoffs", err)
	}

	result := make([]*agentops.Handoff, len(handoffs))
	for i, h := range handoffs {
		result[i] = entHandoffToAgentops(h)
	}
	return result, nil
}

// =============================================================================
// Tool Invocation Operations
// =============================================================================

func (s *Store) RecordToolInvocation(ctx context.Context, taskID, agentID, toolName string, opts ...agentops.ToolInvocationOption) (*agentops.ToolInvocation, error) {
	cfg := agentops.ApplyToolInvocationOptions(opts...)

	id := uuid.New().String()
	now := time.Now()

	create := s.client.ToolInvocation.Create().
		SetID(id).
		SetAgentID(agentID).
		SetToolName(toolName).
		SetStatus(agentops.StatusRunning).
		SetStartedAt(now).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if taskID != "" {
		create.SetTaskID(taskID)
	}
	if cfg.ToolType != "" {
		create.SetToolType(cfg.ToolType)
	}
	if cfg.TraceID != "" {
		create.SetTraceID(cfg.TraceID)
	}
	if cfg.SpanID != "" {
		create.SetSpanID(cfg.SpanID)
	}
	if cfg.Input != nil {
		create.SetInput(cfg.Input)
		if data, err := json.Marshal(cfg.Input); err == nil {
			create.SetRequestSizeBytes(len(data))
		}
	}
	if cfg.Metadata != nil {
		create.SetMetadata(cfg.Metadata)
	}
	if cfg.HTTPMethod != "" {
		create.SetHTTPMethod(cfg.HTTPMethod)
	}
	if cfg.HTTPURL != "" {
		create.SetHTTPURL(cfg.HTTPURL)
	}

	ti, err := create.Save(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "record_tool_invocation", err)
	}

	// Update task tool call count
	if taskID != "" {
		_, _ = s.client.AgentTask.UpdateOneID(taskID).AddToolCallCount(1).Save(ctx)
	}

	return entToolInvocationToAgentops(ti), nil
}

func (s *Store) GetToolInvocation(ctx context.Context, id string) (*agentops.ToolInvocation, error) {
	ti, err := s.client.ToolInvocation.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, agentops.ErrNotFound
		}
		return nil, agentops.WrapError(providerName, "get_tool_invocation", err)
	}
	return entToolInvocationToAgentops(ti), nil
}

func (s *Store) UpdateToolInvocation(ctx context.Context, id string, opts ...agentops.ToolInvocationUpdateOption) error {
	cfg := agentops.ApplyToolInvocationUpdateOptions(opts...)

	update := s.client.ToolInvocation.UpdateOneID(id).SetUpdatedAt(time.Now())

	if cfg.RetryCount > 0 {
		update.AddRetryCount(cfg.RetryCount)
	}

	_, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "update_tool_invocation", err)
	}
	return nil
}

func (s *Store) CompleteToolInvocation(ctx context.Context, id string, opts ...agentops.ToolInvocationCompleteOption) error {
	cfg := agentops.ApplyToolInvocationCompleteOptions(opts...)
	now := time.Now()

	ti, err := s.client.ToolInvocation.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return agentops.ErrNotFound
		}
		return agentops.WrapError(providerName, "complete_tool_invocation", err)
	}

	durationMs := now.Sub(ti.StartedAt).Milliseconds()

	update := s.client.ToolInvocation.UpdateOneID(id).
		SetStatus(agentops.StatusCompleted).
		SetEndedAt(now).
		SetDurationMs(durationMs).
		SetUpdatedAt(now)

	if cfg.Output != nil {
		update.SetOutput(cfg.Output)
	}
	if cfg.HTTPStatusCode > 0 {
		update.SetHTTPStatusCode(cfg.HTTPStatusCode)
	}
	if cfg.ResponseSizeBytes > 0 {
		update.SetResponseSizeBytes(cfg.ResponseSizeBytes)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return agentops.WrapError(providerName, "complete_tool_invocation", err)
	}
	return nil
}

//nolint:dupl // Intentionally similar to ListTasks, different entity types
func (s *Store) ListToolInvocations(ctx context.Context, opts ...agentops.ListOption) ([]*agentops.ToolInvocation, error) {
	cfg := agentops.ApplyListOptions(opts...)

	query := s.client.ToolInvocation.Query()

	if cfg.TaskID != "" {
		query.Where(toolinvocation.TaskIDEQ(cfg.TaskID))
	}
	if cfg.AgentID != "" {
		query.Where(toolinvocation.AgentIDEQ(cfg.AgentID))
	}
	if cfg.Status != "" {
		query.Where(toolinvocation.StatusEQ(cfg.Status))
	}
	if cfg.StartTime != nil {
		query.Where(toolinvocation.StartedAtGTE(*cfg.StartTime))
	}
	if cfg.EndTime != nil {
		query.Where(toolinvocation.StartedAtLTE(*cfg.EndTime))
	}

	if cfg.OrderBy != "" {
		if cfg.OrderDesc {
			query.Order(ent.Desc(cfg.OrderBy))
		} else {
			query.Order(ent.Asc(cfg.OrderBy))
		}
	} else {
		query.Order(ent.Desc(toolinvocation.FieldCreatedAt))
	}

	if cfg.Limit > 0 {
		query.Limit(cfg.Limit)
	}
	if cfg.Offset > 0 {
		query.Offset(cfg.Offset)
	}

	invocations, err := query.All(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "list_tool_invocations", err)
	}

	result := make([]*agentops.ToolInvocation, len(invocations))
	for i, ti := range invocations {
		result[i] = entToolInvocationToAgentops(ti)
	}
	return result, nil
}

// =============================================================================
// Event Operations
// =============================================================================

func (s *Store) EmitEvent(ctx context.Context, eventType string, opts ...agentops.EventOption) (*agentops.Event, error) {
	cfg := agentops.ApplyEventOptions(opts...)

	id := uuid.New().String()
	now := time.Now()

	create := s.client.AgentEvent.Create().
		SetID(id).
		SetEventType(eventType).
		SetTimestamp(now).
		SetCreatedAt(now).
		SetVersion("1.0")

	if cfg.Category != "" {
		create.SetEventCategory(cfg.Category)
	} else {
		create.SetEventCategory(agentops.EventCategoryAgent)
	}
	if cfg.WorkflowID != "" {
		create.SetWorkflowID(cfg.WorkflowID)
	}
	if cfg.TaskID != "" {
		create.SetTaskID(cfg.TaskID)
	}
	if cfg.AgentID != "" {
		create.SetAgentID(cfg.AgentID)
	}
	if cfg.TraceID != "" {
		create.SetTraceID(cfg.TraceID)
	}
	if cfg.SpanID != "" {
		create.SetSpanID(cfg.SpanID)
	}
	if cfg.Severity != "" {
		create.SetSeverity(cfg.Severity)
	} else {
		create.SetSeverity(agentops.SeverityInfo)
	}
	if cfg.Data != nil {
		create.SetData(cfg.Data)
	}
	if cfg.Metadata != nil {
		create.SetMetadata(cfg.Metadata)
	}
	if len(cfg.Tags) > 0 {
		create.SetTags(cfg.Tags)
	}
	if cfg.Source != "" {
		create.SetSource(cfg.Source)
	}

	e, err := create.Save(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "emit_event", err)
	}

	return entEventToAgentops(e), nil
}

func (s *Store) GetEvent(ctx context.Context, id string) (*agentops.Event, error) {
	e, err := s.client.AgentEvent.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, agentops.ErrNotFound
		}
		return nil, agentops.WrapError(providerName, "get_event", err)
	}
	return entEventToAgentops(e), nil
}

func (s *Store) ListEvents(ctx context.Context, opts ...agentops.ListOption) ([]*agentops.Event, error) {
	cfg := agentops.ApplyListOptions(opts...)

	query := s.client.AgentEvent.Query()

	if cfg.WorkflowID != "" {
		query.Where(agentevent.WorkflowIDEQ(cfg.WorkflowID))
	}
	if cfg.TaskID != "" {
		query.Where(agentevent.TaskIDEQ(cfg.TaskID))
	}
	if cfg.AgentID != "" {
		query.Where(agentevent.AgentIDEQ(cfg.AgentID))
	}
	if cfg.EventType != "" {
		query.Where(agentevent.EventTypeEQ(cfg.EventType))
	}
	if cfg.StartTime != nil {
		query.Where(agentevent.TimestampGTE(*cfg.StartTime))
	}
	if cfg.EndTime != nil {
		query.Where(agentevent.TimestampLTE(*cfg.EndTime))
	}

	if cfg.OrderBy != "" {
		if cfg.OrderDesc {
			query.Order(ent.Desc(cfg.OrderBy))
		} else {
			query.Order(ent.Asc(cfg.OrderBy))
		}
	} else {
		query.Order(ent.Desc(agentevent.FieldTimestamp))
	}

	if cfg.Limit > 0 {
		query.Limit(cfg.Limit)
	}
	if cfg.Offset > 0 {
		query.Offset(cfg.Offset)
	}

	events, err := query.All(ctx)
	if err != nil {
		return nil, agentops.WrapError(providerName, "list_events", err)
	}

	result := make([]*agentops.Event, len(events))
	for i, e := range events {
		result[i] = entEventToAgentops(e)
	}
	return result, nil
}

// =============================================================================
// Conversion Functions
// =============================================================================

func entWorkflowToAgentops(w *ent.Workflow) *agentops.Workflow {
	return &agentops.Workflow{
		ID:                 w.ID,
		Name:               w.Name,
		Status:             w.Status,
		TraceID:            w.TraceID,
		ParentWorkflowID:   w.ParentWorkflowID,
		Initiator:          w.Initiator,
		Input:              w.Input,
		Output:             w.Output,
		Metadata:           w.Metadata,
		TaskCount:          w.TaskCount,
		CompletedTaskCount: w.CompletedTaskCount,
		FailedTaskCount:    w.FailedTaskCount,
		TotalCostUSD:       w.TotalCostUsd,
		TotalTokens:        w.TotalTokens,
		DurationMs:         w.DurationMs,
		ErrorMessage:       w.ErrorMessage,
		StartedAt:          w.StartedAt,
		EndedAt:            w.EndedAt,
		CreatedAt:          w.CreatedAt,
		UpdatedAt:          w.UpdatedAt,
	}
}

func entTaskToAgentops(t *ent.AgentTask) *agentops.Task {
	return &agentops.Task{
		ID:               t.ID,
		WorkflowID:       t.WorkflowID,
		AgentID:          t.AgentID,
		AgentType:        t.AgentType,
		TaskType:         t.TaskType,
		Name:             t.Name,
		Status:           t.Status,
		TraceID:          t.TraceID,
		SpanID:           t.SpanID,
		ParentSpanID:     t.ParentSpanID,
		Input:            t.Input,
		Output:           t.Output,
		Metadata:         t.Metadata,
		LLMCallCount:     t.LlmCallCount,
		ToolCallCount:    t.ToolCallCount,
		RetryCount:       t.RetryCount,
		TokensPrompt:     t.TokensPrompt,
		TokensCompletion: t.TokensCompletion,
		TokensTotal:      t.TokensTotal,
		CostUSD:          t.CostUsd,
		DurationMs:       t.DurationMs,
		ErrorType:        t.ErrorType,
		ErrorMessage:     t.ErrorMessage,
		StartedAt:        t.StartedAt,
		EndedAt:          t.EndedAt,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

func entHandoffToAgentops(h *ent.AgentHandoff) *agentops.Handoff {
	return &agentops.Handoff{
		ID:               h.ID,
		WorkflowID:       h.WorkflowID,
		FromAgentID:      h.FromAgentID,
		FromAgentType:    h.FromAgentType,
		ToAgentID:        h.ToAgentID,
		ToAgentType:      h.ToAgentType,
		HandoffType:      h.HandoffType,
		Status:           h.Status,
		TraceID:          h.TraceID,
		FromTaskID:       h.FromTaskID,
		ToTaskID:         h.ToTaskID,
		Payload:          h.Payload,
		Metadata:         h.Metadata,
		PayloadSizeBytes: h.PayloadSizeBytes,
		LatencyMs:        h.LatencyMs,
		ErrorMessage:     h.ErrorMessage,
		InitiatedAt:      h.InitiatedAt,
		AcceptedAt:       h.AcceptedAt,
		CompletedAt:      h.CompletedAt,
		CreatedAt:        h.CreatedAt,
		UpdatedAt:        h.UpdatedAt,
	}
}

func entToolInvocationToAgentops(ti *ent.ToolInvocation) *agentops.ToolInvocation {
	return &agentops.ToolInvocation{
		ID:                ti.ID,
		TaskID:            ti.TaskID,
		AgentID:           ti.AgentID,
		ToolName:          ti.ToolName,
		ToolType:          ti.ToolType,
		Status:            ti.Status,
		TraceID:           ti.TraceID,
		SpanID:            ti.SpanID,
		Input:             ti.Input,
		Output:            ti.Output,
		Metadata:          ti.Metadata,
		HTTPMethod:        ti.HTTPMethod,
		HTTPURL:           ti.HTTPURL,
		HTTPStatusCode:    ti.HTTPStatusCode,
		DurationMs:        ti.DurationMs,
		RequestSizeBytes:  ti.RequestSizeBytes,
		ResponseSizeBytes: ti.ResponseSizeBytes,
		RetryCount:        ti.RetryCount,
		ErrorType:         ti.ErrorType,
		ErrorMessage:      ti.ErrorMessage,
		StartedAt:         ti.StartedAt,
		EndedAt:           ti.EndedAt,
		CreatedAt:         ti.CreatedAt,
		UpdatedAt:         ti.UpdatedAt,
	}
}

func entEventToAgentops(e *ent.AgentEvent) *agentops.Event {
	return &agentops.Event{
		ID:            e.ID,
		EventType:     e.EventType,
		EventCategory: e.EventCategory,
		WorkflowID:    e.WorkflowID,
		TaskID:        e.TaskID,
		AgentID:       e.AgentID,
		TraceID:       e.TraceID,
		SpanID:        e.SpanID,
		Severity:      e.Severity,
		Data:          e.Data,
		Metadata:      e.Metadata,
		Tags:          e.Tags,
		Source:        e.Source,
		Version:       e.Version,
		Timestamp:     e.Timestamp,
		CreatedAt:     e.CreatedAt,
	}
}
