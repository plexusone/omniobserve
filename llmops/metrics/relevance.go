package metrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/plexusone/omniobserve/llmops"
)

// RelevanceMetric is an LLM-based metric that evaluates document relevance.
// It determines whether a document is relevant to answering a query.
type RelevanceMetric struct {
	llm                *LLM
	includeExplanation bool
}

// NewRelevanceMetric creates a new document relevance metric.
func NewRelevanceMetric(llm *LLM) *RelevanceMetric {
	return &RelevanceMetric{
		llm:                llm,
		includeExplanation: true,
	}
}

// NewRelevanceMetricWithOptions creates a relevance metric with options.
func NewRelevanceMetricWithOptions(llm *LLM, includeExplanation bool) *RelevanceMetric {
	return &RelevanceMetric{
		llm:                llm,
		includeExplanation: includeExplanation,
	}
}

// Name returns the metric name.
func (m *RelevanceMetric) Name() string {
	return "relevance"
}

// Evaluate determines document relevance.
// Uses EvalInput.Input as the query and EvalInput.Output as the document.
// Alternatively, uses EvalInput.Context[0] as the document if Output is empty.
// Returns score 1.0 for relevant, 0.0 for irrelevant.
func (m *RelevanceMetric) Evaluate(input llmops.EvalInput) (llmops.MetricScore, error) {
	query := toString(input.Input)
	if query == "" {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: "no query provided for relevance evaluation",
		}, fmt.Errorf("no query provided for relevance evaluation")
	}

	// Get document from Output or first Context item
	document := toString(input.Output)
	if document == "" && len(input.Context) > 0 {
		document = input.Context[0]
	}
	if document == "" {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: "no document provided for relevance evaluation",
		}, fmt.Errorf("no document provided for relevance evaluation")
	}

	// Build prompt from template
	prompt := strings.ReplaceAll(RelevanceTemplate, "{{query}}", query)
	prompt = strings.ReplaceAll(prompt, "{{document}}", document)

	// Classify using LLM
	labels := []string{"relevant", "irrelevant"}
	result, err := m.llm.Classify(context.Background(), prompt, labels, m.includeExplanation)
	if err != nil {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: err.Error(),
		}, err
	}

	// Map label to score: relevant = 1.0, irrelevant = 0.0
	score := 0.0
	if result.Label == "relevant" {
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

// QACorrectnessMetric is an LLM-based metric that evaluates answer correctness.
type QACorrectnessMetric struct {
	llm                *LLM
	includeExplanation bool
}

// NewQACorrectnessMetric creates a new Q&A correctness metric.
func NewQACorrectnessMetric(llm *LLM) *QACorrectnessMetric {
	return &QACorrectnessMetric{
		llm:                llm,
		includeExplanation: true,
	}
}

// Name returns the metric name.
func (m *QACorrectnessMetric) Name() string {
	return "qa_correctness"
}

// Evaluate determines if an answer is correct.
// Uses EvalInput.Input as the question, EvalInput.Output as the AI answer,
// and EvalInput.Expected as the reference answer.
// Returns score 1.0 for correct, 0.0 for incorrect.
func (m *QACorrectnessMetric) Evaluate(input llmops.EvalInput) (llmops.MetricScore, error) {
	question := toString(input.Input)
	answer := toString(input.Output)
	reference := toString(input.Expected)

	if question == "" || answer == "" || reference == "" {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: "question, answer, and reference are required",
		}, fmt.Errorf("question, answer, and reference are required for QA correctness")
	}

	// Build prompt from template
	prompt := strings.ReplaceAll(QACorrectnessTemplate, "{{question}}", question)
	prompt = strings.ReplaceAll(prompt, "{{answer}}", answer)
	prompt = strings.ReplaceAll(prompt, "{{reference}}", reference)

	// Classify using LLM
	labels := []string{"correct", "incorrect"}
	result, err := m.llm.Classify(context.Background(), prompt, labels, m.includeExplanation)
	if err != nil {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: err.Error(),
		}, err
	}

	// Map label to score: correct = 1.0, incorrect = 0.0
	score := 0.0
	if result.Label == "correct" {
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

// ToxicityMetric is an LLM-based metric that detects toxic content.
type ToxicityMetric struct {
	llm                *LLM
	includeExplanation bool
}

// NewToxicityMetric creates a new toxicity detection metric.
func NewToxicityMetric(llm *LLM) *ToxicityMetric {
	return &ToxicityMetric{
		llm:                llm,
		includeExplanation: true,
	}
}

// Name returns the metric name.
func (m *ToxicityMetric) Name() string {
	return "toxicity"
}

// Evaluate detects toxic content.
// Uses EvalInput.Output as the content to evaluate.
// Returns score 1.0 for toxic, 0.0 for safe.
func (m *ToxicityMetric) Evaluate(input llmops.EvalInput) (llmops.MetricScore, error) {
	content := toString(input.Output)
	if content == "" {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: "no content provided for toxicity evaluation",
		}, fmt.Errorf("no content provided for toxicity evaluation")
	}

	// Build prompt from template
	prompt := strings.ReplaceAll(ToxicityTemplate, "{{content}}", content)

	// Classify using LLM
	labels := []string{"toxic", "safe"}
	result, err := m.llm.Classify(context.Background(), prompt, labels, m.includeExplanation)
	if err != nil {
		return llmops.MetricScore{
			Name:  m.Name(),
			Error: err.Error(),
		}, err
	}

	// Map label to score: toxic = 1.0, safe = 0.0
	score := 0.0
	if result.Label == "toxic" {
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
