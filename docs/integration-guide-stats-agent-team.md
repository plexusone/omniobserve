# Observability Strategy for stats-agent-team

Analysis and recommendations for implementing observability in stats-agent-team using Comet Opik and New Relic.

## Executive Summary

| Backend | Profile Value | Recommendation |
|---------|---------------|----------------|
| Opik only | Low | Use Opik's native SDK |
| New Relic only | Medium | Use OTel with agentic-ai conventions |
| Opik + New Relic | High | Use profile for cross-backend consistency |

## stats-agent-team Architecture

stats-agent-team is a multi-agent system for finding and verifying statistics:

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI / MCP Client                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Orchestration Agent                           │
│                    (Port 8000 - Eino/ADK)                        │
└─────────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ Research Agent  │  │ Synthesis Agent │  │ Verification    │
│ (Port 8001)     │  │ (Port 8004)     │  │ Agent (8002)    │
│ No LLM          │  │ LLM-heavy       │  │ LLM-light       │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
   Web Search API      LLM Providers         URL Fetching
   (Serper/SerpAPI)    (Gemini/Claude/       + LLM Validation
                        OpenAI/Grok)
```

### Current Observability State

- LLM observability via OmniObserve (Opik/Langfuse/Phoenix) - optional, disabled by default
- Standard `log.Printf()` logging (not structured)
- No distributed tracing across agents
- No metrics collection

## Analysis: Opik Native SDK vs. Profile-Based Approach

### Opik's Data Model

Opik uses a custom SDK with first-class fields, not OpenTelemetry attributes:

```go
// Opik native fields - rendered in UI
span.SetModel("gpt-4")
span.SetProvider("openai")
span.SetUsage(map[string]int{"prompt_tokens": 100})
span.SetInput(messages)
span.SetOutput(response)
span.AddFeedbackScore("accuracy", 0.95, "verified")

// vs. Profile-based metadata - buried in JSON blob
span.SetMetadata(map[string]any{
    "gen_ai.request.model":      "gpt-4",
    "gen_ai.usage.input_tokens": 100,
    "gen_ai.agent.workflow.id":  "wf-123",
})
```

### UI Support Comparison

| Data Point | Opik Native | Opik UI Support | Profile Attribute | Opik UI Support |
|------------|-------------|-----------------|-------------------|-----------------|
| Model name | `model` | Filterable column | `gen_ai.request.model` | Metadata JSON |
| Token usage | `usage.prompt_tokens` | Charts/totals | `gen_ai.usage.input_tokens` | Not recognized |
| Span type | `type` (LLM/TOOL/AGENT) | Icon + filter | `gen_ai.agent.task.type` | Text only |
| Input/Output | `input`/`output` | Collapsible viewer | N/A | N/A |
| Scores | `feedback_scores` | Visualization | N/A | N/A |

### Verdict: Profile Value by Scenario

#### Scenario 1: Opik Only

**Profile Value: Low**

When using Opik as the sole observability backend, the agentic-ai profile provides minimal benefit:

- Opik's UI is optimized for its native fields
- Profile attributes get stored as unstructured metadata
- No query/filter support for profile attributes
- Added complexity with no practical benefit

**Recommendation:** Use Opik's native SDK directly.

```go
// Recommended for Opik-only
ctx, trace, _ := provider.StartTrace(ctx, "synthesis-workflow",
    llmops.WithTraceInput(request),
    llmops.WithTraceTags("synthesis", "stats-agent-team"),
)

ctx, span, _ := trace.StartSpan(ctx, "extract-statistics",
    llmops.WithSpanType(llmops.SpanTypeLLM),
    llmops.WithSpanModel("gemini-2.0-flash"),
    llmops.WithSpanProvider("google"),
)
```

#### Scenario 2: New Relic Only

**Profile Value: Medium**

New Relic uses OpenTelemetry, so standardized attributes are queryable:

- NRQL can query any attribute: `SELECT * FROM Span WHERE gen_ai.agent.id = 'synthesis'`
- Consistent naming helps dashboard creation
- Profile mappings useful if ingesting from multiple sources

**Recommendation:** Use agentic-ai conventions directly in OTel spans.

```go
// Recommended for New Relic only
ctx, span := tracer.Start(ctx, "synthesis-workflow",
    observops.WithAttributes(
        observops.String("gen_ai.agent.id", "synthesis-agent"),
        observops.String("gen_ai.agent.name", "SynthesisAgent"),
        observops.String("gen_ai.agent.workflow.id", workflowID),
        observops.String("gen_ai.agent.task.type", "synthesis"),
    ),
)
```

#### Scenario 3: Opik + New Relic (Dual Backend)

**Profile Value: High**

When using both backends, the profile provides significant value:

- **Consistent naming** across backends for cross-referencing
- **Attribute mappings** normalize different conventions
- **Single source of truth** for what attributes to capture
- **Future-proofing** if backends change

**Recommendation:** Use profile-based approach with adapter layer.

```go
// Profile-driven dual-backend observability
observer, _ := profiles.New(ctx,
    "observability-profiles/profiles/agentic-ai-standard.json",
    profiles.WithLLMProvider(opikProvider),
    profiles.WithServiceProvider(newRelicProvider),
)

// Attributes normalized automatically
ctx, span, _ := observer.StartAgentSpan(ctx, "synthesis", map[string]any{
    "agent_id":    "synthesis-agent",
    "workflow_id": workflowID,
    "task_type":   "synthesis",
})
// Both backends receive consistent, mapped attributes
```

## Recommended Implementation for stats-agent-team

### Phase 1: Opik Native (Immediate)

Enable existing OmniObserve integration with Opik's native features:

```bash
export OBSERVABILITY_ENABLED=true
export OBSERVABILITY_PROVIDER=opik
export OBSERVABILITY_API_KEY=your-opik-key
export OBSERVABILITY_PROJECT=stats-agent-team
```

Enhance agent instrumentation:

```go
// agents/synthesis/main.go
func (s *SynthesisService) Synthesize(ctx context.Context, req SynthesisRequest) (*SynthesisResponse, error) {
    ctx, trace, _ := s.llmProvider.StartTrace(ctx, "synthesis",
        llmops.WithTraceInput(map[string]any{
            "topic":     req.Topic,
            "url_count": len(req.URLs),
        }),
        llmops.WithTraceTags("synthesis", req.Topic),
    )
    defer trace.End(llmops.WithEndOutput(response))

    for _, url := range req.URLs {
        ctx, span, _ := trace.StartSpan(ctx, "extract-from-url",
            llmops.WithSpanType(llmops.SpanTypeLLM),
            llmops.WithSpanModel(s.config.LLMModel),
            llmops.WithSpanInput(map[string]any{"url": url}),
        )

        // ... extraction logic ...

        span.SetOutput(extracted)
        span.SetUsage(llmops.TokenUsage{
            PromptTokens:     usage.Input,
            CompletionTokens: usage.Output,
        })
        span.End()
    }

    return response, nil
}
```

### Phase 2: Add Structured Logging

Replace `log.Printf` with `slog` for better correlation:

```go
// pkg/observability/logging.go
package observability

import (
    "context"
    "log/slog"
)

func Logger(ctx context.Context) *slog.Logger {
    attrs := []any{}

    if corrID := GetCorrelationID(ctx); corrID != "" {
        attrs = append(attrs, "correlation_id", corrID)
    }
    if agent := GetAgentName(ctx); agent != "" {
        attrs = append(attrs, "agent", agent)
    }

    return slog.Default().With(attrs...)
}

// Usage in agents:
// observability.Logger(ctx).Info("extracting statistics", "url", url)
```

### Phase 3: Add New Relic (If Needed)

If production monitoring with alerting is required:

```go
// pkg/observability/dual.go
package observability

import (
    "github.com/agentplexus/omniobserve/llmops"
    "github.com/agentplexus/omniobserve/observops"
)

type DualObserver struct {
    llm llmops.Provider   // Opik
    svc observops.Provider // New Relic
}

func NewDual(ctx context.Context, cfg Config) (*DualObserver, error) {
    llm, _ := llmops.Open("opik",
        llmops.WithAPIKey(cfg.OpikAPIKey),
        llmops.WithProjectName(cfg.ProjectName),
    )

    svc, _ := observops.Open("newrelic",
        observops.WithAPIKey(cfg.NewRelicAPIKey),
        observops.WithServiceName(cfg.ServiceName),
    )

    return &DualObserver{llm: llm, svc: svc}, nil
}

// StartAgentWorkflow creates correlated spans in both backends
func (d *DualObserver) StartAgentWorkflow(ctx context.Context, agent string, attrs map[string]any) (context.Context, *AgentSpan, error) {
    // New Relic span with agentic-ai conventions
    ctx, svcSpan := d.svc.Tracer().Start(ctx, agent+".workflow",
        observops.WithAttributes(
            observops.String("gen_ai.agent.id", agent),
            observops.String("gen_ai.agent.workflow.id", attrs["workflow_id"].(string)),
        ),
    )

    // Opik trace with native fields
    ctx, llmTrace, _ := d.llm.StartTrace(ctx, agent,
        llmops.WithTraceMetadata(attrs),
    )

    return ctx, &AgentSpan{svc: svcSpan, llm: llmTrace}, nil
}
```

### Phase 4: Profile-Based (If Multi-Backend)

Only if using both Opik and New Relic, add profile-based normalization:

```go
// pkg/observability/profiled.go
package observability

import (
    "github.com/agentplexus/semconv-compose/pkg/compose"
)

type ProfiledObserver struct {
    *DualObserver
    mappings map[string]string
}

func NewProfiled(ctx context.Context, profilePath string, cfg Config) (*ProfiledObserver, error) {
    dual, err := NewDual(ctx, cfg)
    if err != nil {
        return nil, err
    }

    // Load profile for attribute mappings
    composer := compose.New()
    profile, _ := composer.Compose(ctx, profilePath)

    mappings := make(map[string]string)
    for _, m := range profile.Profile.Spec.Mappings {
        mappings[m.From] = m.To
    }

    return &ProfiledObserver{
        DualObserver: dual,
        mappings:     mappings,
    }, nil
}

// MapAttribute normalizes attribute names per profile
func (p *ProfiledObserver) MapAttribute(name string) string {
    if mapped, ok := p.mappings[name]; ok {
        return mapped
    }
    return name
}
```

## When to Use semconv-compose Profiles

### Use Profiles When

1. **Multiple observability backends** - Need consistent attributes across Opik + New Relic/Datadog
2. **Team standardization** - Multiple teams/projects need shared conventions
3. **Backend migration** - Planning to switch from one backend to another
4. **Custom analytics** - Building dashboards that query multiple sources
5. **Compliance requirements** - Need documented, versioned attribute schemas

### Skip Profiles When

1. **Single backend** - Opik-only or New Relic-only usage
2. **Rapid prototyping** - Just need basic observability quickly
3. **Native features suffice** - Backend's native SDK covers your needs
4. **Small team** - No cross-team coordination needed

## Summary: stats-agent-team Recommendation

| Phase | Backend(s) | Profile? | Priority |
|-------|------------|----------|----------|
| 1 | Opik only | No | High - Enable existing integration |
| 2 | Opik + slog | No | High - Add structured logging |
| 3 | Opik + New Relic | Optional | Medium - If prod monitoring needed |
| 4 | Opik + New Relic + Profile | Yes | Low - If cross-backend queries needed |

For stats-agent-team today, **start with Opik native SDK** and add complexity only when the use case demands it. The agentic-ai profile becomes valuable when you need cross-backend consistency, not for single-backend usage.

## Related Resources

- [observability-profiles](https://github.com/agentplexus/observability-profiles) - OTel-based semantic convention profiles
- [semconv-compose](https://github.com/agentplexus/semconv-compose) - Profile composition and validation tool
- [omniobserve](https://github.com/agentplexus/omniobserve) - Multi-backend observability library
- [Comet Opik Documentation](https://www.comet.com/docs/opik/)
- [New Relic OpenTelemetry](https://docs.newrelic.com/docs/opentelemetry/get-started/opentelemetry-get-started-intro/)
