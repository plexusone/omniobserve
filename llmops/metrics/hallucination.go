package metrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/plexusone/omniobserve/llmops"
)

// HallucinationMetric is an LLM-based metric that detects hallucinations.
// It evaluates whether an AI response contains information not supported by the context.
type HallucinationMetric struct {
	llm                *LLM
	includeExplanation bool
}

// NewHallucinationMetric creates a new hallucination detection metric.
func NewHallucinationMetric(llm *LLM) *HallucinationMetric {
	return &HallucinationMetric{
		llm:                llm,
		includeExplanation: true,
	}
}

// NewHallucinationMetricWithOptions creates a hallucination metric with options.
func NewHallucinationMetricWithOptions(llm *LLM, includeExplanation bool) *HallucinationMetric {
	return &HallucinationMetric{
		llm:                llm,
		includeExplanation: includeExplanation,
	}
}

// Name returns the metric name.
func (m *HallucinationMetric) Name() string {
	return "hallucination"
}

// Evaluate detects hallucinations in the output.
// Uses EvalInput.Output as the response and EvalInput.Context as the reference context.
// Returns score 1.0 for hallucinated, 0.0 for factual.
func (m *HallucinationMetric) Evaluate(input llmops.EvalInput) (llmops.MetricScore, error) {
	// Build context string from Context slice
	contextStr := buildContextString(input.Context)
	if contextStr == "" {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: "no context provided for hallucination detection",
		}, fmt.Errorf("no context provided for hallucination detection")
	}

	response := toString(input.Output)
	if response == "" {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: "no response provided for hallucination detection",
		}, fmt.Errorf("no response provided for hallucination detection")
	}

	// Build prompt from template
	prompt := strings.ReplaceAll(HallucinationTemplate, "{{context}}", contextStr)
	prompt = strings.ReplaceAll(prompt, "{{response}}", response)

	// Classify using LLM
	labels := []string{"hallucinated", "factual"}
	result, err := m.llm.Classify(context.Background(), prompt, labels, m.includeExplanation)
	if err != nil {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: err.Error(),
		}, err
	}

	// Map label to score: hallucinated = 1.0, factual = 0.0
	score := 0.0
	if result.Label == "hallucinated" {
		score = 1.0
	}

	return llmops.MetricScore{
		Name:   m.Name(),
		Score:  score,
		Reason: result.Explanation,
		Metadata: map[string]any{
			"label": result.Label,
		},
	}, nil
}

// buildContextString joins context strings with newlines.
func buildContextString(context []string) string {
	if len(context) == 0 {
		return ""
	}
	return strings.Join(context, "\n\n")
}
