// Package agent provides OpenTelemetry semantic conventions for Agentic AI.
//
// These conventions extend OpenTelemetry Semantic Conventions to support
// multi-agent systems, complementing the existing gen_ai.* namespace with
// agent-specific concepts like workflows, tasks, handoffs, and tool calls.
//
// # Namespace Structure
//
// The conventions use a hierarchical namespace under gen_ai.agent.*:
//
//	gen_ai.agent.*              Aligned with OTel GenAI agent attributes
//	gen_ai.agent.workflow.*     Workflow/session-level attributes
//	gen_ai.agent.task.*         Task-level attributes
//	gen_ai.agent.handoff.*      Agent-to-agent communication
//	gen_ai.agent.tool_call.*    Tool/function invocation attributes
//	gen_ai.agent.event.*        Generic event attributes
//
// # Relationship to OpenTelemetry
//
// This package is designed to work alongside OpenTelemetry's GenAI conventions:
//
//   - gen_ai.agent.id, gen_ai.agent.name align with OTel's agent spans
//   - gen_ai.usage.* attributes are reused for token/cost tracking
//   - gen_ai.agent.tool_call.* is distinct from gen_ai.tool.* (definitions vs invocations)
//
// # Usage
//
// Use the constants in this package when setting span attributes:
//
//	span.SetAttributes(
//	    attribute.String(agent.AgentID, "synthesis-agent-1"),
//	    attribute.String(agent.AgentType, "synthesis"),
//	    attribute.String(agent.TaskName, "extract_statistics"),
//	    attribute.String(agent.TaskStatus, agent.StatusRunning),
//	)
//
// # References
//
//   - OpenTelemetry GenAI Conventions: https://opentelemetry.io/docs/specs/semconv/gen-ai/
//   - OpenTelemetry GenAI Agent Spans: https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/
//   - OmniObserve: https://github.com/plexusone/omniobserve
package agent
