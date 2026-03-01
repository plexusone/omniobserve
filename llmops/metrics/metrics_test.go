package metrics

import (
	"context"
	"os"
	"testing"

	"github.com/plexusone/omnillm"
	"github.com/plexusone/omniobserve/llmops"
)

// =============================================================================
// Code-based Metrics Tests (no LLM required)
// =============================================================================

func TestExactMatchMetric_Name(t *testing.T) {
	m := NewExactMatchMetric()
	if m.Name() != "exact_match" {
		t.Errorf("expected name 'exact_match', got '%s'", m.Name())
	}
}

func TestExactMatchMetric_Match(t *testing.T) {
	m := NewExactMatchMetric()

	score, err := m.Evaluate(llmops.EvalInput{
		Output:   "Hello, World!",
		Expected: "Hello, World!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0 for exact match, got %f", score.Score)
	}
}

func TestExactMatchMetric_NoMatch(t *testing.T) {
	m := NewExactMatchMetric()

	score, err := m.Evaluate(llmops.EvalInput{
		Output:   "Hello, World!",
		Expected: "Hello, world!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 0.0 {
		t.Errorf("expected score 0.0 for no match, got %f", score.Score)
	}
}

func TestExactMatchMetric_CaseInsensitive(t *testing.T) {
	m := NewExactMatchMetricWithOptions(WithCaseSensitive(false))

	score, err := m.Evaluate(llmops.EvalInput{
		Output:   "Hello, World!",
		Expected: "hello, world!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0 for case-insensitive match, got %f", score.Score)
	}
}

func TestExactMatchMetric_TrimWhitespace(t *testing.T) {
	m := NewExactMatchMetricWithOptions(WithTrimWhitespace(true))

	score, err := m.Evaluate(llmops.EvalInput{
		Output:   "  Hello, World!  ",
		Expected: "Hello, World!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0 with trimmed whitespace, got %f", score.Score)
	}
}

func TestExactMatchMetric_EmptyStrings(t *testing.T) {
	m := NewExactMatchMetric()

	score, err := m.Evaluate(llmops.EvalInput{
		Output:   "",
		Expected: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0 for empty strings match, got %f", score.Score)
	}
}

// =============================================================================
// Regex Metric Tests
// =============================================================================

func TestRegexMetric_Name(t *testing.T) {
	m := MustRegexMetric(`\d+`)
	if m.Name() != "regex_match" {
		t.Errorf("expected name 'regex_match', got '%s'", m.Name())
	}
}

func TestRegexMetric_CustomName(t *testing.T) {
	m, err := NewRegexMetricWithName("phone_number", `\d{3}-\d{4}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name() != "phone_number" {
		t.Errorf("expected name 'phone_number', got '%s'", m.Name())
	}
}

func TestRegexMetric_Match(t *testing.T) {
	m := MustRegexMetric(`\d+`)

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "There are 42 apples",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0 for regex match, got %f", score.Score)
	}

	// Check metadata
	meta, ok := score.Metadata.(map[string]any)
	if !ok {
		t.Fatal("expected metadata to be map[string]any")
	}
	if meta["match_count"].(int) != 1 {
		t.Errorf("expected 1 match, got %v", meta["match_count"])
	}
}

func TestRegexMetric_MultipleMatches(t *testing.T) {
	m := MustRegexMetric(`\d+`)

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "1 + 2 = 3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", score.Score)
	}

	meta := score.Metadata.(map[string]any)
	if meta["match_count"].(int) != 3 {
		t.Errorf("expected 3 matches, got %v", meta["match_count"])
	}
}

func TestRegexMetric_NoMatch(t *testing.T) {
	m := MustRegexMetric(`\d+`)

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "no numbers here",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 0.0 {
		t.Errorf("expected score 0.0 for no match, got %f", score.Score)
	}
}

func TestRegexMetric_InvalidPattern(t *testing.T) {
	_, err := NewRegexMetric(`[invalid`)
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestMustRegexMetric_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid regex pattern")
		}
	}()
	MustRegexMetric(`[invalid`)
}

// =============================================================================
// Contains Metric Tests
// =============================================================================

func TestContainsMetric_Name(t *testing.T) {
	m := NewContainsMetric("hello", true)
	if m.Name() != "contains" {
		t.Errorf("expected name 'contains', got '%s'", m.Name())
	}
}

func TestContainsMetric_Match(t *testing.T) {
	m := NewContainsMetric("world", true)

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "Hello, world!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", score.Score)
	}
}

func TestContainsMetric_NoMatch(t *testing.T) {
	m := NewContainsMetric("World", true) // case sensitive

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "Hello, world!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 0.0 {
		t.Errorf("expected score 0.0 for case mismatch, got %f", score.Score)
	}
}

func TestContainsMetric_CaseInsensitive(t *testing.T) {
	m := NewContainsMetric("WORLD", false) // case insensitive

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "Hello, world!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0 for case-insensitive match, got %f", score.Score)
	}
}

func TestContainsMetric_EmptySubstring(t *testing.T) {
	m := NewContainsMetric("", true)

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "any string",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 1.0 {
		t.Errorf("expected score 1.0 for empty substring, got %f", score.Score)
	}
}

// =============================================================================
// LLM-based Metrics Tests (input validation only, skip LLM calls)
// =============================================================================

func TestHallucinationMetric_Name(t *testing.T) {
	// Create with nil LLM - just testing name
	m := &HallucinationMetric{}
	if m.Name() != "hallucination" {
		t.Errorf("expected name 'hallucination', got '%s'", m.Name())
	}
}

func TestHallucinationMetric_NoContext(t *testing.T) {
	m := &HallucinationMetric{}

	score, err := m.Evaluate(llmops.EvalInput{
		Output:  "Some response",
		Context: nil,
	})
	if err == nil {
		t.Error("expected error for missing context")
	}
	if score.Error == "" {
		t.Error("expected error message in score")
	}
}

func TestHallucinationMetric_NoOutput(t *testing.T) {
	m := &HallucinationMetric{}

	score, err := m.Evaluate(llmops.EvalInput{
		Output:  "",
		Context: []string{"Some context"},
	})
	if err == nil {
		t.Error("expected error for missing output")
	}
	if score.Error == "" {
		t.Error("expected error message in score")
	}
}

func TestRelevanceMetric_Name(t *testing.T) {
	m := &RelevanceMetric{}
	if m.Name() != "relevance" {
		t.Errorf("expected name 'relevance', got '%s'", m.Name())
	}
}

func TestRelevanceMetric_NoQuery(t *testing.T) {
	m := &RelevanceMetric{}

	score, err := m.Evaluate(llmops.EvalInput{
		Input:  "",
		Output: "Some document",
	})
	if err == nil {
		t.Error("expected error for missing query")
	}
	if score.Error == "" {
		t.Error("expected error message in score")
	}
}

func TestRelevanceMetric_NoDocument(t *testing.T) {
	m := &RelevanceMetric{}

	score, err := m.Evaluate(llmops.EvalInput{
		Input:  "What is Go?",
		Output: "",
	})
	if err == nil {
		t.Error("expected error for missing document")
	}
	if score.Error == "" {
		t.Error("expected error message in score")
	}
}

func TestQACorrectnessMetric_Name(t *testing.T) {
	m := &QACorrectnessMetric{}
	if m.Name() != "qa_correctness" {
		t.Errorf("expected name 'qa_correctness', got '%s'", m.Name())
	}
}

func TestQACorrectnessMetric_MissingFields(t *testing.T) {
	m := &QACorrectnessMetric{}

	// Missing all fields
	score, err := m.Evaluate(llmops.EvalInput{})
	if err == nil {
		t.Error("expected error for missing fields")
	}
	if score.Error == "" {
		t.Error("expected error message in score")
	}
}

func TestToxicityMetric_Name(t *testing.T) {
	m := &ToxicityMetric{}
	if m.Name() != "toxicity" {
		t.Errorf("expected name 'toxicity', got '%s'", m.Name())
	}
}

func TestToxicityMetric_NoContent(t *testing.T) {
	m := &ToxicityMetric{}

	score, err := m.Evaluate(llmops.EvalInput{
		Output: "",
	})
	if err == nil {
		t.Error("expected error for missing content")
	}
	if score.Error == "" {
		t.Error("expected error message in score")
	}
}

// =============================================================================
// LLM Integration Tests (skip if no API key)
// =============================================================================

func getTestLLM(t *testing.T) *LLM {
	t.Helper()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping LLM integration test")
	}

	client, err := omnillm.NewClient(omnillm.ClientConfig{
		Providers: []omnillm.ProviderConfig{
			{Provider: omnillm.ProviderNameOpenAI, APIKey: apiKey},
		},
	})
	if err != nil {
		t.Fatalf("failed to create omnillm client: %v", err)
	}

	return NewLLM(client, "gpt-4o-mini")
}

func TestLLM_Classify(t *testing.T) {
	llm := getTestLLM(t)

	result, err := llm.Classify(context.Background(),
		"Is the sky blue? Answer with yes or no.",
		[]string{"yes", "no"},
		false,
	)
	if err != nil {
		t.Fatalf("classification failed: %v", err)
	}

	if result.Label != "yes" && result.Label != "no" {
		t.Errorf("expected label 'yes' or 'no', got '%s'", result.Label)
	}
}

func TestLLM_GenerateText(t *testing.T) {
	llm := getTestLLM(t)

	result, err := llm.GenerateText(context.Background(),
		"Say 'hello' and nothing else.",
	)
	if err != nil {
		t.Fatalf("text generation failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty response")
	}
}

func TestHallucinationMetric_Integration(t *testing.T) {
	llm := getTestLLM(t)
	m := NewHallucinationMetric(llm)

	// Test factual response
	score, err := m.Evaluate(llmops.EvalInput{
		Output:  "The capital of France is Paris.",
		Context: []string{"Paris is the capital city of France."},
	})
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	// Should be factual (score 0.0)
	if score.Score != 0.0 {
		t.Logf("Note: Expected factual (0.0), got %f - %s", score.Score, score.Reason)
	}

	// Test hallucinated response
	score, err = m.Evaluate(llmops.EvalInput{
		Output:  "The capital of France is London.",
		Context: []string{"Paris is the capital city of France."},
	})
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	// Should be hallucinated (score 1.0)
	if score.Score != 1.0 {
		t.Logf("Note: Expected hallucinated (1.0), got %f - %s", score.Score, score.Reason)
	}
}

func TestRelevanceMetric_Integration(t *testing.T) {
	llm := getTestLLM(t)
	m := NewRelevanceMetric(llm)

	// Test relevant document
	score, err := m.Evaluate(llmops.EvalInput{
		Input:  "What is the capital of France?",
		Output: "Paris is the capital and largest city of France.",
	})
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if score.Score != 1.0 {
		t.Logf("Note: Expected relevant (1.0), got %f - %s", score.Score, score.Reason)
	}

	// Test irrelevant document
	score, err = m.Evaluate(llmops.EvalInput{
		Input:  "What is the capital of France?",
		Output: "Python is a popular programming language.",
	})
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if score.Score != 0.0 {
		t.Logf("Note: Expected irrelevant (0.0), got %f - %s", score.Score, score.Reason)
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestToString(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{nil, ""},
		{"hello", "hello"},
		{[]byte("bytes"), "bytes"},
		{42, "42"},
		{3.14, "3.14"},
	}

	for _, tc := range tests {
		result := toString(tc.input)
		if result != tc.expected {
			t.Errorf("toString(%v) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestBuildContextString(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"one"}, "one"},
		{[]string{"one", "two"}, "one\n\ntwo"},
		{[]string{"a", "b", "c"}, "a\n\nb\n\nc"},
	}

	for _, tc := range tests {
		result := buildContextString(tc.input)
		if result != tc.expected {
			t.Errorf("buildContextString(%v) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
