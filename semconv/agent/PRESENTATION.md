---
marp: true
theme: agentplexus
paginate: true
---

# OpenTelemetry Semantic Conventions for Agentic AI

**Observability for Multi-Agent Systems**

![bg right:40% 80%](https://opentelemetry.io/img/logos/opentelemetry-horizontal-color.svg)

---

# The Rise of Agentic AI

Modern AI systems are evolving from single-model interactions to **multi-agent architectures**

```
User Request
     │
     ▼
┌─────────────┐
│ Orchestrator│
└─────────────┘
     │
     ├──────────────┬──────────────┐
     ▼              ▼              ▼
┌─────────┐   ┌──────────┐   ┌─────────┐
│Research │   │Synthesis │   │ Quality │
│  Agent  │   │  Agent   │   │  Agent  │
└─────────┘   └──────────┘   └─────────┘
     │              │              │
     ▼              ▼              ▼
  [Tools]       [LLM Calls]    [Validation]
```

---

# A Typical Multi-Agent Workflow

1. **Orchestrator** receives user request
2. **Research Agent** searches web, extracts information
3. **Verification Agent** cross-references facts
4. **Synthesis Agent** compiles findings
5. **Quality Agent** reviews before delivery

Each agent makes **multiple LLM calls**, invokes **external tools**, and **passes context** to other agents.

> When something goes wrong—where do you look?

---

# The Observability Gap

## What OpenTelemetry GenAI Covers

- Model identification (`gen_ai.system`, `gen_ai.request.model`)
- Token usage (`gen_ai.usage.*`)
- Tool definitions (`gen_ai.tool.*`)
- Basic agent identity (`gen_ai.agent.id`, `gen_ai.agent.name`)

## What's Missing

- **Workflows** - End-to-end sessions spanning multiple agents
- **Tasks** - Units of work performed by individual agents
- **Handoffs** - Communication and delegation between agents
- **Tool Calls** - Actual tool invocations (not just definitions)

---

# Bridging the Gap

We extend OpenTelemetry's `gen_ai.agent.*` namespace:

```
gen_ai.agent.*              # Agent identity (aligned with OTel)
gen_ai.agent.workflow.*     # Workflow/session tracking
gen_ai.agent.task.*         # Task execution
gen_ai.agent.handoff.*      # Agent-to-agent communication
gen_ai.agent.tool_call.*    # Tool invocations
```

### Design Principles

- **Extend, don't replace** - Works alongside existing GenAI conventions
- **Avoid collisions** - `tool_call.*` vs OTel's `tool.*`
- **Enable convergence** - Minimal migration if OTel adopts upstream

---

# Namespace Alignment

| OTel GenAI | Our Extension | Purpose |
|------------|---------------|---------|
| `gen_ai.system` | - | LLM provider |
| `gen_ai.request.model` | - | Model identification |
| `gen_ai.usage.*` | (reused) | Token/cost tracking |
| `gen_ai.agent.id` | (aligned) | Agent identification |
| `gen_ai.tool.*` | - | Tool definitions |
| - | `gen_ai.agent.workflow.*` | Workflow orchestration |
| - | `gen_ai.agent.task.*` | Task execution |
| - | `gen_ai.agent.handoff.*` | Agent communication |
| - | `gen_ai.agent.tool_call.*` | Tool execution |

---

# Workflow Attributes

Track end-to-end processing sessions:

```yaml
gen_ai.agent.workflow.id: "wf-550e8400-e29b-41d4-a716-446655440000"
gen_ai.agent.workflow.name: "statistics-extraction"
gen_ai.agent.workflow.status: "completed"
gen_ai.agent.workflow.task.count: 5
gen_ai.agent.workflow.task.completed_count: 5
gen_ai.agent.workflow.duration: 45000
gen_ai.usage.total_tokens: 15420
gen_ai.usage.cost: 0.0847
```

At a glance: workflow health, task success rate, duration, and cost.

---

# Task Attributes

Track individual units of work:

```yaml
gen_ai.agent.task.id: "task-research-001"
gen_ai.agent.task.name: "extract_gdp_statistics"
gen_ai.agent.task.type: "extraction"
gen_ai.agent.id: "research-agent-1"
gen_ai.agent.task.llm.call_count: 3
gen_ai.agent.task.tool_call.count: 7
gen_ai.agent.task.duration: 12500
gen_ai.agent.task.error.type: "rate_limit"
gen_ai.agent.task.error.message: "OpenAI rate limit exceeded"
```

Immediate visibility: which agent failed, what it was doing, and why.

---

# Handoff Attributes

Track agent-to-agent communication:

```yaml
gen_ai.agent.handoff.id: "ho-789"
gen_ai.agent.handoff.type: "delegate"
gen_ai.agent.handoff.from.agent.id: "orchestrator"
gen_ai.agent.handoff.to.agent.id: "synthesis-agent"
gen_ai.agent.handoff.payload.size: 4096
gen_ai.agent.handoff.latency: 23
gen_ai.agent.handoff.status: "completed"
```

Understand communication patterns, measure latency, identify bottlenecks.

---

# Tool Call Attributes

Track actual tool invocations:

```yaml
gen_ai.agent.tool_call.id: "tc-search-042"
gen_ai.agent.tool_call.name: "web_search"
gen_ai.agent.tool_call.type: "search"
gen_ai.agent.tool_call.duration: 850
gen_ai.agent.tool_call.http.status_code: 200
gen_ai.agent.tool_call.response.size: 15360
gen_ai.agent.tool_call.retry_count: 1
```

**Note:** This is distinct from OTel's `gen_ai.tool.*` which describes tool *definitions*.

---

# Span Hierarchy

The conventions create a natural trace structure:

```
Workflow Span (gen_ai.agent.workflow)
├── Task Span (gen_ai.agent.task) - Agent A
│   ├── LLM Span (gen_ai inference) - via OTel GenAI
│   ├── Tool Call Span (gen_ai.agent.tool_call)
│   └── LLM Span (gen_ai inference)
├── Handoff Span (gen_ai.agent.handoff) - A → B
├── Task Span (gen_ai.agent.task) - Agent B
│   ├── LLM Span (gen_ai inference)
│   └── Tool Call Span (gen_ai.agent.tool_call)
└── Task Span (gen_ai.agent.task) - Agent C
```

---

# Status Values

Consistent status tracking across all entity types:

| Value | Description |
|-------|-------------|
| `pending` | Not yet started |
| `running` | Currently executing |
| `completed` | Successfully finished |
| `failed` | Finished with error |
| `cancelled` | Cancelled before completion |

**Handoff-specific:**
| Value | Description |
|-------|-------------|
| `accepted` | Accepted by target agent |
| `rejected` | Rejected by target agent |

---

# Handoff Types

Different patterns of agent communication:

| Type | Description | Use Case |
|------|-------------|----------|
| `request` | Request for action/info | "Please search for X" |
| `response` | Reply to a request | "Here are the results" |
| `delegate` | Transfer responsibility | "You handle this task" |
| `broadcast` | Send to multiple agents | "All agents: new context" |

---

# Error Types

Standardized error categorization:

| Type | Description |
|------|-------------|
| `timeout` | Operation timed out |
| `rate_limit` | Rate limit exceeded |
| `validation` | Input validation failed |
| `internal` | Internal error |
| `network` | Network error |
| `auth` | Authentication/authorization error |

Enables consistent error analysis across workflows.

---

# Implementation: Middleware Approach

Minimal code changes with maximum observability:

```go
import (
    "github.com/plexusone/omniobserve/agentops"
    "github.com/plexusone/omniobserve/agentops/middleware"
)

// 1. Create a store
store, _ := agentops.Open("postgres", agentops.WithDSN(dsn))

// 2. Start a workflow
ctx, workflow, _ := middleware.StartWorkflow(ctx, store,
    "statistics-extraction",
    middleware.WithInitiator("user:123"),
)
defer middleware.CompleteWorkflow(ctx)
```

---

# Instrumenting Agents

Wrap HTTP handlers for automatic task tracking:

```go
// Configure agent handler
handler := middleware.AgentHandler(middleware.AgentHandlerConfig{
    AgentID:   "synthesis-agent",
    AgentType: "synthesis",
    Store:     store,
})(yourHandler)

// Each request automatically creates a task that captures:
// - Start/end time and duration
// - HTTP status code
// - Success/failure status
// - Link to parent workflow
```

---

# Tracking Handoffs

Instrumented client for inter-agent communication:

```go
// Create instrumented client
client := middleware.NewAgentClient(http.DefaultClient,
    middleware.AgentClientConfig{
        FromAgentID:   "orchestrator",
        FromAgentType: "orchestration",
        Store:         store,
    },
)

// Calls are automatically recorded as handoffs
resp, _ := client.PostJSON(ctx,
    "http://synthesis:8004/extract",
    body,
    "synthesis-agent",
)
```

---

# Instrumenting Tool Calls

Generic wrapper for any function:

```go
results, err := middleware.ToolCall(ctx, "web_search",
    func() ([]Result, error) {
        return searchService.Search(query)
    },
    middleware.WithToolType("search"),
)
```

Convenience wrappers for common patterns:

```go
// Search tools
results, _ := middleware.SearchToolCall(ctx, "web_search", query, searchFn)

// Database tools
rows, _ := middleware.DatabaseToolCall(ctx, "user_query", sql, queryFn)

// API tools
data, _ := middleware.APIToolCall(ctx, "weather_api", "GET", url, apiFn)
```

---

# Automatic Context Propagation

Context flows automatically across boundaries:

**Within a process:**
```go
ctx = middleware.WithWorkflow(ctx, workflow)
// Workflow, task, agent info available via context
```

**Across services (HTTP headers):**
```
X-AgentOps-Workflow-ID: wf-550e8400-...
X-AgentOps-Task-ID: task-123
X-AgentOps-Agent-ID: orchestrator
```

No manual ID passing required.

---

# What the Middleware Tracks

| Component | What It Tracks | Code |
|-----------|----------------|------|
| `StartWorkflow()` | Lifecycle, duration, task counts | ~3 lines |
| `AgentHandler()` | Task timing, HTTP status, errors | ~5 lines/agent |
| `NewAgentClient()` | Handoff latency, payload size | ~5 lines shared |
| `ToolCall()` | Execution time, request/response size | ~3 lines/call |

---

# Use Case: Debugging Failed Workflows

Query by `gen_ai.agent.workflow.id` to see:

- Which task failed → `gen_ai.agent.task.error.type`
- Which agent was responsible → `gen_ai.agent.id`
- What tool call caused it → `gen_ai.agent.tool_call.error.message`
- How many retries occurred → `gen_ai.agent.tool_call.retry_count`

```sql
SELECT task_name, agent_id, error_type, error_message
FROM tasks
WHERE workflow_id = 'wf-550e8400-...'
  AND status = 'failed';
```

---

# Use Case: Cost Attribution

Aggregate costs by workflow, agent, or task type:

```sql
-- Cost by workflow type
SELECT workflow_name, SUM(cost) as total_cost
FROM workflows
GROUP BY workflow_name
ORDER BY total_cost DESC;

-- Cost by agent type
SELECT agent_type, SUM(total_tokens) as tokens
FROM tasks
GROUP BY agent_type;
```

Identify which agents and tasks consume the most resources.

---

# Use Case: Performance Optimization

Find bottlenecks in your agent system:

```sql
-- Slow tasks
SELECT task_name, AVG(duration) as avg_duration
FROM tasks
GROUP BY task_name
ORDER BY avg_duration DESC
LIMIT 10;

-- Handoff latency between agents
SELECT from_agent_id, to_agent_id, AVG(latency) as avg_latency
FROM handoffs
GROUP BY from_agent_id, to_agent_id;

-- Slow tools
SELECT tool_name, AVG(duration), COUNT(*) as calls
FROM tool_invocations
GROUP BY tool_name;
```

---

# YAML Model Definitions

Following OTel's approach, conventions are defined in YAML:

```
model/
├── registry.yaml   # Attribute definitions
├── spans.yaml      # Span type definitions
└── events.yaml     # Event type definitions
```

These serve as the **source of truth** for:
- Code generation in multiple languages
- Documentation generation
- Validation tooling

---

# Registry Example

```yaml
- id: gen_ai.agent.task.status
  type:
    members:
      - id: pending
        value: "pending"
        brief: Task not yet started.
      - id: running
        value: "running"
        brief: Task currently executing.
      - id: completed
        value: "completed"
        brief: Task finished successfully.
      - id: failed
        value: "failed"
        brief: Task finished with error.
  brief: Current status of the task.
```

---

# Span Definition Example

```yaml
- span_name: gen_ai.agent.task
  brief: A span representing a task executed by an agent.
  attributes:
    - ref: gen_ai.agent.task.id
      requirement_level: required
    - ref: gen_ai.agent.task.name
      requirement_level: required
    - ref: gen_ai.agent.id
      requirement_level: required
    - ref: gen_ai.agent.task.status
      requirement_level: recommended
    - ref: gen_ai.agent.task.duration
      requirement_level: recommended
```

---

# Direct Attribute Usage

For manual instrumentation with OpenTelemetry:

```go
import (
    "github.com/plexusone/omniobserve/semconv/agent"
    "go.opentelemetry.io/otel/attribute"
)

span.SetAttributes(
    attribute.String(agent.AgentID, "synthesis-agent-1"),
    attribute.String(agent.AgentType, "synthesis"),
    attribute.String(agent.WorkflowID, "wf-123"),
    attribute.String(agent.TaskName, "extract_statistics"),
    attribute.String(agent.TaskStatus, agent.StatusRunning),
)
```

---

# Event Emission

Emit events at key lifecycle points:

```go
span.AddEvent(agent.EventNameTaskStarted,
    trace.WithAttributes(
        attribute.String(agent.TaskID, taskID),
        attribute.String(agent.TaskName, "verify_sources"),
        attribute.String(agent.AgentID, "verification-agent"),
    ),
)
```

Standard event names:
- `gen_ai.agent.workflow.started` / `.completed` / `.failed`
- `gen_ai.agent.task.started` / `.completed` / `.failed`
- `gen_ai.agent.handoff.initiated` / `.completed` / `.failed`
- `gen_ai.agent.tool_call.started` / `.completed` / `.failed`

---

# Compatibility

Designed to work with:

- **OpenTelemetry GenAI Semantic Conventions**
  - Extends `gen_ai.agent.*` namespace
  - Reuses `gen_ai.usage.*` for token tracking

- **OpenTelemetry Trace Context**
  - Standard W3C trace propagation

- **OpenInference (Arize Phoenix)**
  - Compatible attribute naming

---

# Summary

## What We Provide

| Concept | Namespace | Purpose |
|---------|-----------|---------|
| Workflows | `gen_ai.agent.workflow.*` | End-to-end session tracking |
| Tasks | `gen_ai.agent.task.*` | Individual work units |
| Handoffs | `gen_ai.agent.handoff.*` | Agent communication |
| Tool Calls | `gen_ai.agent.tool_call.*` | Tool invocations |

## Implementation Options

- **Middleware** - Minimal code, maximum coverage
- **Direct** - Full control with OTel integration

---

# Get Started

```go
import "github.com/plexusone/omniobserve/semconv/agent"
```

Or use the YAML models for code generation in other languages.

## Resources

- [OpenTelemetry GenAI Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
- [OpenTelemetry GenAI Agent Spans](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/)
- [OmniObserve Repository](https://github.com/plexusone/omniobserve)

---

# Thank You

**OpenTelemetry Semantic Conventions for Agentic AI**

*Multi-agent AI systems deserve first-class observability.*

---

<!--
Appendix slides for reference
-->

# Appendix: Full Attribute Reference

## Agent Attributes

| Attribute | Type | Example |
|-----------|------|---------|
| `gen_ai.agent.id` | string | `"synthesis-agent-1"` |
| `gen_ai.agent.name` | string | `"Synthesis Agent"` |
| `gen_ai.agent.type` | string | `"synthesis"` |
| `gen_ai.agent.version` | string | `"1.0.0"` |

---

# Appendix: Workflow Attributes

| Attribute | Type | Example |
|-----------|------|---------|
| `gen_ai.agent.workflow.id` | string | `"wf-550e8400-..."` |
| `gen_ai.agent.workflow.name` | string | `"statistics-extraction"` |
| `gen_ai.agent.workflow.status` | string | `"completed"` |
| `gen_ai.agent.workflow.parent_id` | string | `"wf-parent-123"` |
| `gen_ai.agent.workflow.initiator` | string | `"user:123"` |
| `gen_ai.agent.workflow.task.count` | int | `5` |
| `gen_ai.agent.workflow.task.completed_count` | int | `4` |
| `gen_ai.agent.workflow.task.failed_count` | int | `1` |
| `gen_ai.agent.workflow.duration` | int64 | `45000` |

---

# Appendix: Task Attributes

| Attribute | Type | Example |
|-----------|------|---------|
| `gen_ai.agent.task.id` | string | `"task-123"` |
| `gen_ai.agent.task.name` | string | `"extract_stats"` |
| `gen_ai.agent.task.type` | string | `"extraction"` |
| `gen_ai.agent.task.status` | string | `"running"` |
| `gen_ai.agent.task.parent_id` | string | `"task-parent"` |
| `gen_ai.agent.task.retry_count` | int | `2` |
| `gen_ai.agent.task.duration` | int64 | `1500` |
| `gen_ai.agent.task.llm.call_count` | int | `3` |
| `gen_ai.agent.task.tool_call.count` | int | `5` |
| `gen_ai.agent.task.error.type` | string | `"timeout"` |
| `gen_ai.agent.task.error.message` | string | `"API timeout"` |

---

# Appendix: Handoff Attributes

| Attribute | Type | Example |
|-----------|------|---------|
| `gen_ai.agent.handoff.id` | string | `"ho-789"` |
| `gen_ai.agent.handoff.type` | string | `"delegate"` |
| `gen_ai.agent.handoff.status` | string | `"completed"` |
| `gen_ai.agent.handoff.from.agent.id` | string | `"research-agent"` |
| `gen_ai.agent.handoff.from.agent.type` | string | `"research"` |
| `gen_ai.agent.handoff.to.agent.id` | string | `"synthesis-agent"` |
| `gen_ai.agent.handoff.to.agent.type` | string | `"synthesis"` |
| `gen_ai.agent.handoff.payload.size` | int | `2048` |
| `gen_ai.agent.handoff.latency` | int64 | `50` |

---

# Appendix: Tool Call Attributes

| Attribute | Type | Example |
|-----------|------|---------|
| `gen_ai.agent.tool_call.id` | string | `"tc-abc123"` |
| `gen_ai.agent.tool_call.name` | string | `"web_search"` |
| `gen_ai.agent.tool_call.type` | string | `"search"` |
| `gen_ai.agent.tool_call.status` | string | `"completed"` |
| `gen_ai.agent.tool_call.duration` | int64 | `250` |
| `gen_ai.agent.tool_call.request.size` | int | `512` |
| `gen_ai.agent.tool_call.response.size` | int | `4096` |
| `gen_ai.agent.tool_call.retry_count` | int | `1` |
| `gen_ai.agent.tool_call.http.method` | string | `"POST"` |
| `gen_ai.agent.tool_call.http.url` | string | `"https://..."` |
| `gen_ai.agent.tool_call.http.status_code` | int | `200` |
