// Example: evaluation
//
// Demonstrates using evaluation metrics to assess LLM outputs.
// Shows both code-based metrics (no LLM required) and LLM-based metrics.
//
// Usage:
//
//	# For code-based metrics only
//	go run main.go
//
//	# For LLM-based metrics
//	export OPENAI_API_KEY=your-key
//	go run main.go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/agentplexus/omnillm"
	"github.com/agentplexus/omniobserve/llmops"
	"github.com/agentplexus/omniobserve/llmops/metrics"
)

func main() {
	fmt.Println("=== Code-Based Metrics (no LLM required) ===")
	fmt.Println()

	// Exact Match Metric
	exactMatch := metrics.NewExactMatchMetric()

	score, _ := exactMatch.Evaluate(llmops.EvalInput{
		Output:   "Paris",
		Expected: "Paris",
	})
	fmt.Printf("Exact Match (Paris vs Paris): %.1f - %s\n", score.Score, score.Reason)

	score, _ = exactMatch.Evaluate(llmops.EvalInput{
		Output:   "paris",
		Expected: "Paris",
	})
	fmt.Printf("Exact Match (paris vs Paris): %.1f - %s\n", score.Score, score.Reason)

	// Case-insensitive exact match
	caseInsensitive := metrics.NewExactMatchMetricWithOptions(
		metrics.WithCaseSensitive(false),
	)

	score, _ = caseInsensitive.Evaluate(llmops.EvalInput{
		Output:   "paris",
		Expected: "Paris",
	})
	fmt.Printf("Case-Insensitive Match: %.1f - %s\n", score.Score, score.Reason)

	fmt.Println()

	// Regex Metric
	emailRegex := metrics.MustRegexMetric(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	score, _ = emailRegex.Evaluate(llmops.EvalInput{
		Output: "Contact me at user@example.com for more info.",
	})
	fmt.Printf("Email Regex Match: %.1f - %s\n", score.Score, score.Reason)
	if meta, ok := score.Metadata.(map[string]any); ok {
		fmt.Printf("  Matches found: %v\n", meta["matches"])
	}

	score, _ = emailRegex.Evaluate(llmops.EvalInput{
		Output: "No email address here.",
	})
	fmt.Printf("Email Regex (no match): %.1f - %s\n", score.Score, score.Reason)

	fmt.Println()

	// Contains Metric
	containsMetric := metrics.NewContainsMetric("error", false) // case-insensitive

	score, _ = containsMetric.Evaluate(llmops.EvalInput{
		Output: "An ERROR occurred during processing.",
	})
	fmt.Printf("Contains 'error': %.1f - %s\n", score.Score, score.Reason)

	score, _ = containsMetric.Evaluate(llmops.EvalInput{
		Output: "Operation completed successfully.",
	})
	fmt.Printf("Contains 'error' (no match): %.1f - %s\n", score.Score, score.Reason)

	// LLM-based metrics (optional)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("\n=== LLM-Based Metrics ===")
		fmt.Println("Set OPENAI_API_KEY to run LLM-based metric examples.")
		return
	}

	fmt.Println()
	fmt.Println("=== LLM-Based Metrics ===")
	fmt.Println()

	// Create LLM client for metrics
	llmClient, err := omnillm.NewClient(omnillm.ClientConfig{
		Providers: []omnillm.ProviderConfig{
			{Provider: omnillm.ProviderNameOpenAI, APIKey: apiKey},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}
	defer llmClient.Close()

	llm := metrics.NewLLM(llmClient, "gpt-4o-mini")

	// Hallucination Detection
	hallucination := metrics.NewHallucinationMetric(llm)

	score, err = hallucination.Evaluate(llmops.EvalInput{
		Output:  "The capital of France is Paris.",
		Context: []string{"Paris is the capital city of France."},
	})
	if err != nil {
		log.Printf("Hallucination check failed: %v", err)
	} else {
		label := "factual"
		if score.Score == 1.0 {
			label = "hallucinated"
		}
		fmt.Printf("Hallucination (factual response): %s\n", label)
		fmt.Printf("  Explanation: %s\n", score.Reason)
	}

	score, err = hallucination.Evaluate(llmops.EvalInput{
		Output:  "The capital of France is London.",
		Context: []string{"Paris is the capital city of France."},
	})
	if err != nil {
		log.Printf("Hallucination check failed: %v", err)
	} else {
		label := "factual"
		if score.Score == 1.0 {
			label = "hallucinated"
		}
		fmt.Printf("Hallucination (incorrect response): %s\n", label)
		fmt.Printf("  Explanation: %s\n", score.Reason)
	}

	fmt.Println()

	// Relevance Scoring
	relevance := metrics.NewRelevanceMetric(llm)

	score, err = relevance.Evaluate(llmops.EvalInput{
		Input:  "What is the capital of France?",
		Output: "Paris is the capital and most populous city of France.",
	})
	if err != nil {
		log.Printf("Relevance check failed: %v", err)
	} else {
		label := "irrelevant"
		if score.Score == 1.0 {
			label = "relevant"
		}
		fmt.Printf("Relevance (on-topic document): %s\n", label)
		fmt.Printf("  Explanation: %s\n", score.Reason)
	}

	score, err = relevance.Evaluate(llmops.EvalInput{
		Input:  "What is the capital of France?",
		Output: "Python is a popular programming language created by Guido van Rossum.",
	})
	if err != nil {
		log.Printf("Relevance check failed: %v", err)
	} else {
		label := "irrelevant"
		if score.Score == 1.0 {
			label = "relevant"
		}
		fmt.Printf("Relevance (off-topic document): %s\n", label)
		fmt.Printf("  Explanation: %s\n", score.Reason)
	}

	fmt.Println("\nEvaluation examples complete!")
}
