# OpenTelemetry Semantic Conventions for Agentic AI

This package provides semantic conventions for observability in multi-agent AI systems. It extends the [OpenTelemetry Semantic Conventions for Generative AI](https://opentelemetry.io/docs/specs/semconv/gen-ai/) with agent-specific concepts like workflows, tasks, handoffs, and tool calls.

## Status

**Development** - These conventions are under active development and may change.

## Model Files

The semantic conventions are defined in YAML format following the same structure as
[OpenTelemetry Semantic Conventions](https://github.com/open-telemetry/semantic-conventions):

```
model/
├── registry.yaml   # Attribute definitions
├── spans.yaml      # Span type definitions
└── events.yaml     # Event type definitions
```

These YAML files serve as the source of truth and can be used to generate code for
multiple languages or validate instrumentation.

## Overview

Modern AI applications increasingly use multi-agent architectures where specialized agents collaborate to accomplish complex tasks. This package defines semantic conventions for observing these systems, complementing OpenTelemetry's existing `gen_ai.*` namespace.

### Namespace Structure

```
gen_ai.agent.*              # Agent identity (aligns with OTel)
gen_ai.agent.workflow.*     # Workflow/session tracking
gen_ai.agent.task.*         # Task execution
gen_ai.agent.handoff.*      # Agent-to-agent communication
gen_ai.agent.tool_call.*    # Tool/function invocations
```

### Relationship to OpenTelemetry GenAI Conventions

These conventions are designed to work alongside OpenTelemetry's GenAI semantic conventions:

| OTel GenAI | This Package | Purpose |
|------------|--------------|---------|
| `gen_ai.system` | - | LLM provider identification |
| `gen_ai.request.model` | - | Model used for requests |
| `gen_ai.usage.*` | `gen_ai.usage.*` (reused) | Token/cost tracking |
| `gen_ai.agent.id` | `gen_ai.agent.id` (aligned) | Agent identification |
| `gen_ai.agent.name` | `gen_ai.agent.name` (aligned) | Agent naming |
| `gen_ai.tool.*` | - | Tool definitions/schemas |
| - | `gen_ai.agent.workflow.*` | Workflow orchestration |
| - | `gen_ai.agent.task.*` | Task execution tracking |
| - | `gen_ai.agent.handoff.*` | Agent communication |
| - | `gen_ai.agent.tool_call.*` | Tool execution (not definitions) |

## Conventions

### Agent Attributes

Core agent identification attributes, aligned with OTel's `gen_ai.agent.*`:

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `gen_ai.agent.id` | string | Unique agent instance identifier | `"synthesis-agent-1"` |
| `gen_ai.agent.name` | string | Human-readable agent name | `"Synthesis Agent"` |
| `gen_ai.agent.type` | string | Agent role/function category | `"synthesis"`, `"research"` |
| `gen_ai.agent.version` | string | Agent implementation version | `"1.0.0"` |

### Workflow Attributes

Workflows represent end-to-end processing sessions that may involve multiple agents and tasks:

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `gen_ai.agent.workflow.id` | string | Unique workflow identifier | `"wf-550e8400-..."` |
| `gen_ai.agent.workflow.name` | string | Workflow type/name | `"statistics-extraction"` |
| `gen_ai.agent.workflow.status` | string | Current status | `"running"`, `"completed"` |
| `gen_ai.agent.workflow.parent_id` | string | Parent workflow for nesting | `"wf-parent-123"` |
| `gen_ai.agent.workflow.initiator` | string | What started the workflow | `"user:123"`, `"api_key:abc"` |
| `gen_ai.agent.workflow.task.count` | int | Total tasks in workflow | `5` |
| `gen_ai.agent.workflow.task.completed_count` | int | Completed task count | `3` |
| `gen_ai.agent.workflow.task.failed_count` | int | Failed task count | `1` |
| `gen_ai.agent.workflow.duration` | int64 | Duration in milliseconds | `45000` |

### Task Attributes

Tasks represent individual units of work performed by an agent:

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `gen_ai.agent.task.id` | string | Unique task identifier | `"task-123"` |
| `gen_ai.agent.task.name` | string | Task name | `"extract_gdp_stats"` |
| `gen_ai.agent.task.type` | string | Task category | `"extraction"`, `"verification"` |
| `gen_ai.agent.task.status` | string | Current status | `"running"`, `"completed"` |
| `gen_ai.agent.task.parent_id` | string | Parent task for nesting | `"task-parent-456"` |
| `gen_ai.agent.task.retry_count` | int | Number of retry attempts | `2` |
| `gen_ai.agent.task.duration` | int64 | Duration in milliseconds | `1500` |
| `gen_ai.agent.task.llm.call_count` | int | LLM calls made | `3` |
| `gen_ai.agent.task.tool_call.count` | int | Tool calls made | `5` |
| `gen_ai.agent.task.error.type` | string | Error category if failed | `"timeout"`, `"rate_limit"` |
| `gen_ai.agent.task.error.message` | string | Error message if failed | `"API timeout after 30s"` |

### Handoff Attributes

Handoffs represent communication between agents:

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `gen_ai.agent.handoff.id` | string | Unique handoff identifier | `"ho-789"` |
| `gen_ai.agent.handoff.type` | string | Handoff type | `"request"`, `"delegate"` |
| `gen_ai.agent.handoff.status` | string | Current status | `"pending"`, `"accepted"` |
| `gen_ai.agent.handoff.from.agent.id` | string | Source agent ID | `"research-agent-1"` |
| `gen_ai.agent.handoff.from.agent.type` | string | Source agent type | `"research"` |
| `gen_ai.agent.handoff.to.agent.id` | string | Target agent ID | `"synthesis-agent-1"` |
| `gen_ai.agent.handoff.to.agent.type` | string | Target agent type | `"synthesis"` |
| `gen_ai.agent.handoff.from.task.id` | string | Source task ID | `"task-123"` |
| `gen_ai.agent.handoff.to.task.id` | string | Target task ID | `"task-456"` |
| `gen_ai.agent.handoff.payload.size` | int | Payload size in bytes | `2048` |
| `gen_ai.agent.handoff.latency` | int64 | Latency in milliseconds | `50` |
| `gen_ai.agent.handoff.error.message` | string | Error message if failed | `"Agent unavailable"` |

### Tool Call Attributes

Tool calls represent invocations of external tools/functions by agents. This is distinct from OTel's `gen_ai.tool.*` which describes tool definitions/schemas.

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `gen_ai.agent.tool_call.id` | string | Unique invocation identifier | `"tc-abc123"` |
| `gen_ai.agent.tool_call.name` | string | Tool name | `"web_search"` |
| `gen_ai.agent.tool_call.type` | string | Tool category | `"search"`, `"database"` |
| `gen_ai.agent.tool_call.status` | string | Execution status | `"running"`, `"completed"` |
| `gen_ai.agent.tool_call.duration` | int64 | Duration in milliseconds | `250` |
| `gen_ai.agent.tool_call.request.size` | int | Request payload size | `512` |
| `gen_ai.agent.tool_call.response.size` | int | Response payload size | `4096` |
| `gen_ai.agent.tool_call.retry_count` | int | Retry attempts | `1` |
| `gen_ai.agent.tool_call.error.type` | string | Error category | `"network"`, `"timeout"` |
| `gen_ai.agent.tool_call.error.message` | string | Error message | `"Connection refused"` |
| `gen_ai.agent.tool_call.http.method` | string | HTTP method if applicable | `"POST"` |
| `gen_ai.agent.tool_call.http.url` | string | HTTP URL if applicable | `"https://api.example.com"` |
| `gen_ai.agent.tool_call.http.status_code` | int | HTTP status code | `200` |

### Event Attributes

Events provide a generic mechanism for domain-specific observability:

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `gen_ai.agent.event.id` | string | Unique event identifier | `"evt-xyz"` |
| `gen_ai.agent.event.name` | string | Event name/type | `"statistic.extracted"` |
| `gen_ai.agent.event.category` | string | Event category | `"agent"`, `"domain"` |
| `gen_ai.agent.event.source` | string | Event source | `"synthesis-agent"` |
| `gen_ai.agent.event.severity` | string | Severity level | `"info"`, `"error"` |

## Enumerated Values

### Status Values

Used by workflow, task, handoff, and tool_call status attributes:

| Value | Description |
|-------|-------------|
| `pending` | Not yet started |
| `running` | Currently executing |
| `completed` | Successfully finished |
| `failed` | Finished with error |
| `cancelled` | Cancelled before completion |

Additional handoff-specific status values:

| Value | Description |
|-------|-------------|
| `accepted` | Handoff accepted by target agent |
| `rejected` | Handoff rejected by target agent |

### Handoff Types

| Value | Description |
|-------|-------------|
| `request` | Request for action/information |
| `response` | Response to a previous request |
| `broadcast` | Broadcast to multiple agents |
| `delegate` | Delegation of responsibility |

### Error Types

| Value | Description |
|-------|-------------|
| `timeout` | Operation timed out |
| `rate_limit` | Rate limit exceeded |
| `validation` | Input validation failed |
| `internal` | Internal error |
| `network` | Network error |
| `auth` | Authentication/authorization error |

### Event Categories

| Value | Description |
|-------|-------------|
| `agent` | Agent lifecycle events |
| `workflow` | Workflow-level events |
| `tool` | Tool-related events |
| `domain` | Domain-specific events |
| `system` | System-level events |

### Severity Levels

| Value | Description |
|-------|-------------|
| `debug` | Debug information |
| `info` | Informational |
| `warn` | Warning |
| `error` | Error |

## Usage

### Quick Start with Middleware

The easiest way to instrument your agent system is using the middleware package, which provides
reusable helpers that minimize code changes:

```go
import (
    "github.com/plexusone/omniobserve/agentops"
    "github.com/plexusone/omniobserve/agentops/middleware"
    _ "github.com/plexusone/omniobserve/agentops/postgres"
)

// 1. Create a store
store, _ := agentops.Open("postgres", agentops.WithDSN(dsn))
defer store.Close()

// 2. Start a workflow (in orchestrator)
ctx, workflow, _ := middleware.StartWorkflow(ctx, store, "my-workflow",
    middleware.WithInitiator("user:123"),
)
defer middleware.CompleteWorkflow(ctx)

// 3. Instrument agent HTTP handlers (automatic task creation)
handler := middleware.AgentHandler(middleware.AgentHandlerConfig{
    AgentID:   "synthesis-agent",
    AgentType: "synthesis",
    Store:     store,
})(yourHandler)

// 4. Use instrumented client for inter-agent calls (automatic handoff tracking)
client := middleware.NewAgentClient(http.DefaultClient, middleware.AgentClientConfig{
    FromAgentID: "orchestrator",
    Store:       store,
})
resp, _ := client.PostJSON(ctx, "http://synthesis:8004/extract", body, "synthesis-agent")

// 5. Wrap tool calls (automatic timing and error tracking)
results, _ := middleware.ToolCall(ctx, "web_search", func() ([]Result, error) {
    return searchService.Search(query)
}, middleware.WithToolType("search"))
```

### Middleware Components

| Component | Purpose | Code Changes |
|-----------|---------|--------------|
| `middleware.StartWorkflow()` | Create workflow, attach to context | ~3 lines in orchestrator |
| `middleware.AgentHandler()` | Wrap HTTP handlers as tasks | ~5 lines per agent |
| `middleware.NewAgentClient()` | Track inter-agent calls as handoffs | ~5 lines shared |
| `middleware.ToolCall()` | Instrument tool/function calls | ~3 lines per call site |

### Convenience Wrappers

```go
// Search tools
results, _ := middleware.SearchToolCall(ctx, "web_search", query, searchFn)

// Database tools
rows, _ := middleware.DatabaseToolCall(ctx, "user_query", sql, queryFn)

// API tools
data, _ := middleware.APIToolCall(ctx, "weather_api", "GET", url, apiFn)

// With automatic retry tracking
result, _ := middleware.RetryToolCall(ctx, "flaky_api", 3, retryableFn)
```

### Context Propagation

The middleware automatically propagates observability context:

- **Go context**: Workflow, task, agent, and store attached to `context.Context`
- **HTTP headers**: `X-AgentOps-Workflow-ID`, `X-AgentOps-Task-ID`, `X-AgentOps-Agent-ID`

This enables distributed tracing across agent boundaries without manual header management.

### Direct Attribute Usage

For manual instrumentation or OpenTelemetry integration:

```go
import (
    "github.com/plexusone/omniobserve/semconv/agent"
    "go.opentelemetry.io/otel/attribute"
)

// Set span attributes
span.SetAttributes(
    attribute.String(agent.AgentID, "synthesis-agent-1"),
    attribute.String(agent.AgentType, "synthesis"),
    attribute.String(agent.WorkflowID, "wf-123"),
    attribute.String(agent.TaskName, "extract_statistics"),
    attribute.String(agent.TaskStatus, agent.StatusRunning),
)

// Record tool call
span.SetAttributes(
    attribute.String(agent.ToolCallID, "tc-456"),
    attribute.String(agent.ToolCallName, "web_search"),
    attribute.Int64(agent.ToolCallDuration, 250),
)
```

## Compatibility

These conventions are designed to be compatible with:

- [OpenTelemetry Semantic Conventions for GenAI](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
- [OpenTelemetry Trace Context](https://www.w3.org/TR/trace-context/)
- [OpenInference](https://github.com/Arize-ai/openinference) (Arize Phoenix)

## References

- [OpenTelemetry GenAI Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
- [OpenTelemetry GenAI Agent Spans](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/)
- [OmniObserve](https://github.com/plexusone/omniobserve)

## License

Apache 2.0 - See [LICENSE](../../LICENSE) for details.
