// Package sevaluation provides integration between structured-evaluation reports
// and omniobserve llmops providers (Opik, Phoenix, Langfuse).
//
// This package enables:
//   - Exporting EvaluationReport scores and findings to observability platforms
//   - Converting platform evaluation results into EvaluationReport format
//   - Correlating evaluations with LLM traces for debugging
//
// # Exporting to Providers
//
// Use Export to send an EvaluationReport to any llmops provider:
//
//	provider, _ := llmops.Open("opik", llmops.WithAPIKey("..."))
//	report := evaluation.NewEvaluationReport("prd", "document.md")
//	// ... populate report ...
//
//	err := sevaluation.Export(ctx, provider, traceID, report)
//
// This will:
//   - Add feedback scores for each category (normalized 0-1)
//   - Create annotations for each finding with severity labels
//   - Add an overall evaluation score with the decision status
//
// # Score Normalization
//
// structured-evaluation uses 0-10 scores while llmops uses 0-1.
// This package automatically normalizes scores during export.
package sevaluation
