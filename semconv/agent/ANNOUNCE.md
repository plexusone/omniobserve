# Introducing OpenTelemetry Semantic Conventions for Agentic AI

**Observability for the Next Generation of AI Systems**

As AI systems evolve from single-model interactions to sophisticated multi-agent architectures, our observability tools must evolve with them. Today, we're introducing semantic conventions for Agentic AI—an extension to OpenTelemetry that brings structured observability to multi-agent systems.

## The Rise of Agentic AI

The AI landscape is shifting. What started as simple prompt-response interactions has evolved into complex systems where multiple specialized agents collaborate to accomplish tasks. Consider a modern AI workflow:

1. An **orchestrator agent** receives a user request to research a topic
2. A **research agent** searches the web and extracts relevant information
3. A **verification agent** cross-references facts against authoritative sources
4. A **synthesis agent** compiles findings into a coherent response
5. A **quality agent** reviews the output before delivery

Each agent makes multiple LLM calls, invokes external tools, and passes context to other agents. When something goes wrong—or when you need to optimize performance—where do you look?

## The Observability Gap

OpenTelemetry has made excellent progress with semantic conventions for Generative AI (`gen_ai.*`), covering LLM-specific concepts like:

- Model identification and provider tracking
- Token usage and cost attribution
- Prompt and completion content
- Tool definitions

But these conventions operate at the LLM call level. They don't capture the higher-level concepts that define agentic systems:

- **Workflows**: End-to-end sessions spanning multiple agents
- **Tasks**: Units of work performed by individual agents
- **Handoffs**: Communication and delegation between agents
- **Tool Calls**: External tool invocations (distinct from tool definitions)

Without conventions for these concepts, teams resort to ad-hoc logging, inconsistent tagging, and fragmented observability that makes debugging multi-agent systems a nightmare.

## Bridging the Gap

Our semantic conventions extend OpenTelemetry's `gen_ai.*` namespace with agent-specific attributes:

```
gen_ai.agent.*              # Agent identity (aligned with OTel)
gen_ai.agent.workflow.*     # Workflow/session tracking
gen_ai.agent.task.*         # Task execution
gen_ai.agent.handoff.*      # Agent-to-agent communication
gen_ai.agent.tool_call.*    # Tool invocations (execution, not definitions)
```

### Why Extend Rather Than Replace?

We deliberately chose to extend `gen_ai.agent.*` rather than create a separate namespace. This approach:

1. **Aligns with OTel's direction** - OpenTelemetry already defines `gen_ai.agent.id` and `gen_ai.agent.name` for agent spans
2. **Enables seamless integration** - Existing GenAI instrumentation continues to work
3. **Reduces cognitive load** - One namespace hierarchy to learn
4. **Supports future convergence** - If OTel adopts these conventions, migration is minimal

### Avoiding Collisions

We carefully designed the namespace to avoid conflicts:

- **`gen_ai.tool.*`** (OTel) = Tool definitions and schemas
- **`gen_ai.agent.tool_call.*`** (ours) = Tool execution and invocation tracking

This distinction is important. OTel's tool attributes describe what tools are available. Our tool_call attributes track what happened when a tool was actually used.

## What You Can Observe

### Workflow-Level Visibility

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

At a glance, you can see the entire workflow's health, how many tasks succeeded, total duration, and cost.

### Task-Level Debugging

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

When a task fails, you immediately know which agent failed, what it was doing, and why.

### Handoff Tracing

```yaml
gen_ai.agent.handoff.id: "ho-789"
gen_ai.agent.handoff.type: "delegate"
gen_ai.agent.handoff.from.agent.id: "orchestrator"
gen_ai.agent.handoff.to.agent.id: "synthesis-agent"
gen_ai.agent.handoff.payload.size: 4096
gen_ai.agent.handoff.latency: 23
gen_ai.agent.handoff.status: "completed"
```

Track how agents communicate, measure handoff latency, and identify bottlenecks in agent coordination.

### Tool Call Analysis

```yaml
gen_ai.agent.tool_call.id: "tc-search-042"
gen_ai.agent.tool_call.name: "web_search"
gen_ai.agent.tool_call.type: "search"
gen_ai.agent.tool_call.duration: 850
gen_ai.agent.tool_call.http.status_code: 200
gen_ai.agent.tool_call.response.size: 15360
gen_ai.agent.tool_call.retry_count: 1
```

Understand which tools agents use, how long they take, and where retries occur.

## Implementation Guide

### The Easy Way: Middleware

We provide a middleware package that makes instrumentation nearly effortless. Instead of manually
creating spans and setting attributes, use our reusable helpers:

```go
import (
    "github.com/plexusone/omniobserve/agentops"
    "github.com/plexusone/omniobserve/agentops/middleware"
    _ "github.com/plexusone/omniobserve/agentops/postgres"
)

// 1. Create a store (one-time setup)
store, _ := agentops.Open("postgres", agentops.WithDSN(dsn))
defer store.Close()

// 2. Start a workflow (in your orchestrator)
ctx, workflow, _ := middleware.StartWorkflow(ctx, store, "statistics-extraction",
    middleware.WithInitiator("user:123"),
)
defer middleware.CompleteWorkflow(ctx)

// 3. Wrap your agent HTTP handlers (automatic task tracking)
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

### What the Middleware Does For You

| Component | What It Tracks | Code Required |
|-----------|---------------|---------------|
| `StartWorkflow()` | Workflow lifecycle, duration, task counts | ~3 lines |
| `AgentHandler()` | Task start/end, duration, HTTP status, errors | ~5 lines per agent |
| `NewAgentClient()` | Handoff initiation, latency, payload size, status | ~5 lines shared |
| `ToolCall()` | Tool execution, duration, request/response size, errors | ~3 lines per call |

### Convenience Wrappers

For common tool types, we provide specialized wrappers:

```go
// Search tools - automatically sets tool type and captures query
results, _ := middleware.SearchToolCall(ctx, "web_search", query, searchFn)

// Database tools - captures SQL query
rows, _ := middleware.DatabaseToolCall(ctx, "user_query", sql, queryFn)

// API tools - captures HTTP method and URL
data, _ := middleware.APIToolCall(ctx, "weather_api", "GET", url, apiFn)

// With automatic retry tracking
result, _ := middleware.RetryToolCall(ctx, "flaky_api", 3, retryableFn)
```

### Automatic Context Propagation

The middleware handles context propagation automatically:

- **Within a process**: Workflow, task, and agent info flow through `context.Context`
- **Across services**: HTTP headers (`X-AgentOps-Workflow-ID`, `X-AgentOps-Task-ID`) propagate context

You don't need to manually pass IDs between agents—the middleware handles it.

### Span Hierarchy

The middleware creates this trace structure automatically:

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

### Direct Implementation (Manual Control)

For cases where you need full control, you can use the semantic conventions directly:

```go
import (
    "github.com/plexusone/omniobserve/semconv/agent"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func runWorkflow(ctx context.Context, tracer trace.Tracer) {
    ctx, span := tracer.Start(ctx, "workflow statistics-extraction",
        trace.WithAttributes(
            attribute.String(agent.WorkflowID, uuid.New().String()),
            attribute.String(agent.WorkflowName, "statistics-extraction"),
            attribute.String(agent.WorkflowStatus, agent.StatusRunning),
        ),
    )
    defer span.End()

    // Execute workflow...
}
```

### Event Emission

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

## YAML Model Definitions

Following OTel's approach, we define our conventions in YAML:

```
model/
├── registry.yaml   # Attribute definitions with types, examples, enums
├── spans.yaml      # Span definitions with required/recommended attributes
└── events.yaml     # Event definitions for lifecycle tracking
```

These serve as the source of truth and can generate code for multiple languages.

**Example from registry.yaml:**

```yaml
- id: gen_ai.agent.task.status
  type:
    members:
      - id: pending
        value: "pending"
        brief: Task not yet started.
        stability: development
      - id: running
        value: "running"
        brief: Task currently executing.
        stability: development
      - id: completed
        value: "completed"
        brief: Task finished successfully.
        stability: development
      - id: failed
        value: "failed"
        brief: Task finished with error.
        stability: development
  stability: development
  brief: Current status of the task.
```

## Use Cases

### Debugging Failed Workflows

When a workflow fails, query by `gen_ai.agent.workflow.id` to see:
- Which task failed (`gen_ai.agent.task.error.type`)
- Which agent was responsible (`gen_ai.agent.id`)
- What tool call might have caused it (`gen_ai.agent.tool_call.error.message`)
- How many retries occurred before failure

### Cost Attribution

Aggregate `gen_ai.usage.cost` by:
- Workflow type (`gen_ai.agent.workflow.name`)
- Agent type (`gen_ai.agent.type`)
- Task type (`gen_ai.agent.task.type`)

Identify which agents and tasks consume the most resources.

### Performance Optimization

Analyze `gen_ai.agent.task.duration` and `gen_ai.agent.handoff.latency` to find:
- Slow tasks that need optimization
- Handoff bottlenecks between agents
- Tools with high latency (`gen_ai.agent.tool_call.duration`)

### Agent Coordination Analysis

Use handoff attributes to understand:
- Communication patterns between agents
- Payload sizes being transferred
- Rejection rates and failure modes

## What's Next

These conventions are in **development** status. We're actively seeking:

1. **Feedback** from teams building multi-agent systems
2. **Real-world validation** across different agent frameworks
3. **Collaboration** with OpenTelemetry GenAI SIG for potential upstream adoption

## Get Started

The semantic conventions are available as part of OmniObserve:

```go
import "github.com/plexusone/omniobserve/semconv/agent"
```

Or use the YAML models directly for code generation in other languages.

## Resources

- [OpenTelemetry GenAI Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
- [OpenTelemetry GenAI Agent Spans](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/)
- [OmniObserve Repository](https://github.com/plexusone/omniobserve)

---

*Multi-agent AI systems deserve first-class observability. These semantic conventions are a step toward making that a reality.*
