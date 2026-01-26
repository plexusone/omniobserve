package sevaluation

import (
	"context"
	"fmt"

	"github.com/agentplexus/omniobserve/llmops"
	"github.com/agentplexus/structured-evaluation/evaluation"
)

// ExportOptions configures the export behavior.
type ExportOptions struct {
	// IncludeFindings controls whether findings are exported as annotations.
	// Default: true
	IncludeFindings bool

	// IncludeCategories controls whether category scores are exported as feedback scores.
	// Default: true
	IncludeCategories bool

	// IncludeOverall controls whether the overall score and decision are exported.
	// Default: true
	IncludeOverall bool

	// ScorePrefix is prepended to score names (e.g., "eval_" -> "eval_problem_definition").
	// Default: ""
	ScorePrefix string

	// Source identifies the evaluation source in annotations.
	// Default: "structured-evaluation"
	Source string
}

// DefaultExportOptions returns the default export configuration.
func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		IncludeFindings:   true,
		IncludeCategories: true,
		IncludeOverall:    true,
		ScorePrefix:       "",
		Source:            "structured-evaluation",
	}
}

// exportTarget specifies whether to export to a trace or span.
type exportTarget struct {
	traceID string
	spanID  string
}

// Export sends an EvaluationReport to an llmops provider, attaching to a trace.
// It exports category scores as feedback scores and findings as annotations.
func Export(ctx context.Context, provider llmops.Provider, traceID string, report *evaluation.EvaluationReport, opts ...ExportOptions) error {
	return exportReport(ctx, provider, exportTarget{traceID: traceID}, report, opts...)
}

// ExportToSpan exports an EvaluationReport to a specific span instead of a trace.
func ExportToSpan(ctx context.Context, provider llmops.Provider, spanID string, report *evaluation.EvaluationReport, opts ...ExportOptions) error {
	return exportReport(ctx, provider, exportTarget{spanID: spanID}, report, opts...)
}

// exportReport is the internal implementation for exporting to either trace or span.
func exportReport(ctx context.Context, provider llmops.Provider, target exportTarget, report *evaluation.EvaluationReport, opts ...ExportOptions) error {
	opt := DefaultExportOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	var errs []error

	// Export category scores as feedback scores
	if opt.IncludeCategories {
		for _, cat := range report.Categories {
			scoreOpts := llmops.FeedbackScoreOpts{
				TraceID:  target.traceID,
				SpanID:   target.spanID,
				Name:     opt.ScorePrefix + cat.Category,
				Score:    NormalizeScore(cat.Score),
				Category: report.ReviewType,
				Reason:   cat.Justification,
				Source:   opt.Source,
			}
			if err := provider.AddFeedbackScore(ctx, scoreOpts); err != nil {
				errs = append(errs, fmt.Errorf("export category %s: %w", cat.Category, err))
			}
		}
	}

	// Export findings as annotations
	if opt.IncludeFindings {
		for i, f := range report.Findings {
			ann := llmops.Annotation{
				TraceID:     target.traceID,
				SpanID:      target.spanID,
				Name:        fmt.Sprintf("%s%s_%d", opt.ScorePrefix, "finding", i+1),
				Label:       string(f.Severity),
				Explanation: formatFinding(f),
				Source:      llmops.AnnotatorKindLLM,
				Metadata: map[string]any{
					"category":       f.Category,
					"title":          f.Title,
					"severity":       string(f.Severity),
					"recommendation": f.Recommendation,
					"owner":          f.Owner,
					"blocking":       f.IsBlocking(),
				},
			}
			if err := provider.CreateAnnotation(ctx, ann); err != nil {
				errs = append(errs, fmt.Errorf("export finding %d: %w", i+1, err))
			}
		}
	}

	// Export overall evaluation score and decision
	if opt.IncludeOverall {
		scoreOpts := llmops.FeedbackScoreOpts{
			TraceID:  target.traceID,
			SpanID:   target.spanID,
			Name:     opt.ScorePrefix + "overall_score",
			Score:    NormalizeScore(report.WeightedScore),
			Category: "evaluation",
			Reason:   report.Summary,
			Source:   opt.Source,
		}
		if err := provider.AddFeedbackScore(ctx, scoreOpts); err != nil {
			errs = append(errs, fmt.Errorf("export overall score: %w", err))
		}

		// Export decision as annotation
		ann := llmops.Annotation{
			TraceID:     target.traceID,
			SpanID:      target.spanID,
			Name:        opt.ScorePrefix + "decision",
			Label:       string(report.Decision.Status),
			Explanation: report.Summary,
			Source:      llmops.AnnotatorKindLLM,
			Metadata: map[string]any{
				"status":         string(report.Decision.Status),
				"weighted_score": report.WeightedScore,
				"critical_count": report.Decision.FindingCounts.Critical,
				"high_count":     report.Decision.FindingCounts.High,
				"medium_count":   report.Decision.FindingCounts.Medium,
				"low_count":      report.Decision.FindingCounts.Low,
				"review_type":    report.ReviewType,
				"document":       report.Metadata.Document,
			},
		}
		if err := provider.CreateAnnotation(ctx, ann); err != nil {
			errs = append(errs, fmt.Errorf("export decision: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("export errors: %v", errs)
	}
	return nil
}

// NormalizeScore converts a 0-10 score to 0-1 range.
func NormalizeScore(score float64) float64 {
	return score / 10.0
}

// DenormalizeScore converts a 0-1 score to 0-10 range.
func DenormalizeScore(score float64) float64 {
	return score * 10.0
}

// formatFinding creates a human-readable finding description.
func formatFinding(f evaluation.Finding) string {
	return fmt.Sprintf("[%s] %s: %s - %s",
		f.Severity,
		f.Category,
		f.Title,
		f.Recommendation,
	)
}
