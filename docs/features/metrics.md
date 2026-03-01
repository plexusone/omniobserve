# Evaluation Metrics

OmniObserve provides both code-based and LLM-based evaluation metrics in the `llmops/metrics` package.

## Code-Based Metrics

### Exact Match

```go
import "github.com/plexusone/omniobserve/llmops/metrics"

metric := metrics.ExactMatchMetric()

result, _ := metric.Evaluate(ctx, llmops.EvalInput{
    Output:   "Hello, world!",
    Expected: "Hello, world!",
})
// result.Score = 1.0 (exact match)
```

### Contains

```go
metric := metrics.ContainsMetric("error")

result, _ := metric.Evaluate(ctx, llmops.EvalInput{
    Output: "An error occurred in the system",
})
// result.Score = 1.0 (contains "error")
```

### Regex Match

```go
metric, _ := metrics.RegexMetric(`\d{3}-\d{3}-\d{4}`)

result, _ := metric.Evaluate(ctx, llmops.EvalInput{
    Output: "Call us at 555-123-4567",
})
// result.Score = 1.0 (matches phone pattern)
```

## LLM-Based Metrics

These metrics use an LLM to evaluate outputs. Requires an API key.

### Hallucination Detection

```go
metric := metrics.HallucinationMetric(metrics.LLMConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o-mini",
})

result, _ := metric.Evaluate(ctx, llmops.EvalInput{
    Output:  "The Eiffel Tower is in Berlin.",
    Context: "The Eiffel Tower is located in Paris, France.",
})
// result.Score = 0.0 (hallucinated)
```

### Relevance

```go
metric := metrics.RelevanceMetric(metrics.LLMConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

result, _ := metric.Evaluate(ctx, llmops.EvalInput{
    Input:  "What is the capital of France?",
    Output: "Paris is the capital of France.",
})
// result.Score = 1.0 (relevant)
```

### QA Correctness

```go
metric := metrics.QACorrectnessMetric(metrics.LLMConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

result, _ := metric.Evaluate(ctx, llmops.EvalInput{
    Input:    "What is 2+2?",
    Output:   "The answer is 4.",
    Expected: "4",
})
```

### Toxicity

```go
metric := metrics.ToxicityMetric(metrics.LLMConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

result, _ := metric.Evaluate(ctx, llmops.EvalInput{
    Output: "This is a friendly message.",
})
// result.Score = 0.0 (not toxic)
```

## Adding Feedback Scores to Traces

```go
// Evaluate and add to span
result, _ := metric.Evaluate(ctx, input)

span.AddFeedbackScore(ctx, metric.Name(), result.Score,
    llmops.WithFeedbackReason(result.Reason),
)
```

## Custom Metrics

Implement the `Metric` interface:

```go
type Metric interface {
    Name() string
    Evaluate(ctx context.Context, input EvalInput) (*EvalResult, error)
}
```
