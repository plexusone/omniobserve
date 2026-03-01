# Structured Evaluation Integration

OmniObserve integrates with [structured-evaluation](https://github.com/grokify/structured-evaluation) (sevaluation) to connect evaluation workflows with observability traces.

## Installation

```bash
go get github.com/plexusone/omniobserve/integrations/sevaluation
```

## Overview

The sevaluation integration bridges:

- **Evaluation workflows** - Running structured evaluation suites
- **Trace recording** - Capturing evaluation results in observability providers
- **Feedback scores** - Adding evaluation scores to traces and spans

## Basic Usage

```go
import (
    "github.com/plexusone/omniobserve/integrations/sevaluation"
    "github.com/plexusone/omniobserve/llmops"
)

// Initialize provider
provider, _ := llmops.Open("opik",
    llmops.WithAPIKey("..."),
    llmops.WithProjectName("evaluations"),
)

// Create evaluation integration
eval := sevaluation.New(provider)

// Run evaluations and record results
results, err := eval.Run(ctx, suite)
```

## Use Cases

### LLM Output Evaluation

Evaluate LLM outputs and record results to your observability provider:

```go
ctx, trace, _ := provider.StartTrace(ctx, "evaluation-run")
defer trace.End()

// Run evaluation suite
results, _ := eval.Evaluate(ctx, sevaluation.EvalConfig{
    Suite: mySuite,
    Input: llmOutput,
})

// Scores are automatically added to the trace
```

### RAG Pipeline Evaluation

Evaluate retrieval and generation quality:

```go
// Retrieval relevance
results, _ := eval.EvaluateRetrieval(ctx, sevaluation.RetrievalConfig{
    Query:     query,
    Documents: retrievedDocs,
})

// Generation quality
results, _ := eval.EvaluateGeneration(ctx, sevaluation.GenerationConfig{
    Input:    query,
    Context:  retrievedDocs,
    Output:   generatedResponse,
    Expected: expectedAnswer,
})
```

## Recording to Traces

Evaluation results are automatically recorded as feedback scores:

```go
// Results include scores that can be added to spans
for _, result := range results {
    span.AddFeedbackScore(ctx, result.MetricName, result.Score,
        llmops.WithFeedbackReason(result.Reason),
    )
}
```

## Provider Support

| Provider | Evaluation Recording |
|----------|:--------------------:|
| Opik | :white_check_mark: |
| Langfuse | :white_check_mark: |
| Phoenix | :white_check_mark: |
| slog | :x: |
