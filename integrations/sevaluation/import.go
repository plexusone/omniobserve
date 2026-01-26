package sevaluation

import (
	"github.com/agentplexus/omniobserve/llmops"
	"github.com/agentplexus/structured-evaluation/evaluation"
)

// ImportOptions configures the import behavior.
type ImportOptions struct {
	// ReviewType sets the review type for the generated report.
	// Default: "llm_evaluation"
	ReviewType string

	// Document sets the document name for the report metadata.
	// Default: ""
	Document string

	// DefaultWeight sets the weight for imported categories.
	// Default: 1.0
	DefaultWeight float64

	// PassCriteria sets the criteria for the report.
	// Default: evaluation.DefaultPassCriteria()
	PassCriteria evaluation.PassCriteria
}

// DefaultImportOptions returns the default import configuration.
func DefaultImportOptions() ImportOptions {
	return ImportOptions{
		ReviewType:    "llm_evaluation",
		Document:      "",
		DefaultWeight: 1.0,
		PassCriteria:  evaluation.DefaultPassCriteria(),
	}
}

// ImportEvalResult converts an llmops EvalResult into an EvaluationReport.
// Each MetricScore becomes a CategoryScore in the report.
func ImportEvalResult(result *llmops.EvalResult, opts ...ImportOptions) *evaluation.EvaluationReport {
	opt := DefaultImportOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	report := evaluation.NewEvaluationReport(opt.ReviewType, opt.Document)
	report.PassCriteria = opt.PassCriteria

	for _, score := range result.Scores {
		cat := evaluation.CategoryScore{
			Category:      score.Name,
			Weight:        opt.DefaultWeight,
			Score:         denormalizeScore(score.Score),
			MaxScore:      10.0,
			Justification: score.Reason,
		}
		cat.ComputeStatus()
		report.AddCategory(cat)

		// If score has an error, add it as a finding
		if score.Error != "" {
			report.AddFinding(evaluation.Finding{
				Severity:       evaluation.SeverityMedium,
				Category:       score.Name,
				Title:          "Evaluation error",
				Description:    score.Error,
				Recommendation: "Review and re-run evaluation",
			})
		}
	}

	return report
}

// ImportMetricScores converts a slice of MetricScores into an EvaluationReport.
func ImportMetricScores(scores []llmops.MetricScore, opts ...ImportOptions) *evaluation.EvaluationReport {
	result := &llmops.EvalResult{Scores: scores}
	return ImportEvalResult(result, opts...)
}

// MetricScoreToCategory converts a single MetricScore to a CategoryScore.
func MetricScoreToCategory(score llmops.MetricScore, weight float64) evaluation.CategoryScore {
	cat := evaluation.CategoryScore{
		Category:      score.Name,
		Weight:        weight,
		Score:         denormalizeScore(score.Score),
		MaxScore:      10.0,
		Justification: score.Reason,
	}
	cat.ComputeStatus()
	return cat
}

// denormalizeScore converts a 0-1 score to 0-10 range.
func denormalizeScore(score float64) float64 {
	return score * 10.0
}

// AnnotationToFinding converts an llmops Annotation to an evaluation Finding.
func AnnotationToFinding(ann llmops.Annotation) evaluation.Finding {
	severity := evaluation.SeverityInfo
	if ann.Label != "" {
		switch ann.Label {
		case "critical", "CRITICAL":
			severity = evaluation.SeverityCritical
		case "high", "HIGH":
			severity = evaluation.SeverityHigh
		case "medium", "MEDIUM":
			severity = evaluation.SeverityMedium
		case "low", "LOW":
			severity = evaluation.SeverityLow
		}
	}

	category := ""
	if ann.Metadata != nil {
		if cat, ok := ann.Metadata["category"].(string); ok {
			category = cat
		}
	}

	return evaluation.Finding{
		Severity:    severity,
		Category:    category,
		Title:       ann.Name,
		Description: ann.Explanation,
	}
}
