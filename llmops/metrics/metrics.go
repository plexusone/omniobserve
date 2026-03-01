// Package metrics provides evaluation metrics for LLM observability.
//
// This package implements the llmops.Metric interface with common evaluation
// metrics used in LLM applications:
//
//   - Hallucination detection (LLM-based)
//   - Document relevance scoring (LLM-based)
//   - Exact match comparison (code-based)
//   - Regex pattern matching (code-based)
//
// LLM-based metrics require an LLM client from omnillm to perform evaluations.
// Code-based metrics run locally without LLM calls.
//
// # Usage
//
//	import (
//	    "github.com/plexusone/omniobserve/llmops/metrics"
//	    "github.com/plexusone/omnillm"
//	)
//
//	// Create LLM client for LLM-based metrics
//	client, _ := omnillm.NewClient(omnillm.ClientConfig{
//	    Provider: omnillm.ProviderNameOpenAI,
//	    APIKey:   os.Getenv("OPENAI_API_KEY"),
//	})
//	llm := metrics.NewLLM(client, "gpt-4o")
//
//	// Create metrics
//	hallucination := metrics.NewHallucinationMetric(llm)
//	exactMatch := metrics.NewExactMatchMetric()
//
//	// Evaluate
//	score, err := hallucination.Evaluate(llmops.EvalInput{
//	    Output:  "The capital of France is London.",
//	    Context: []string{"Paris is the capital of France."},
//	})
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plexusone/omnillm"
	"github.com/plexusone/omnillm/provider"
)

// LLM wraps an omnillm.ChatClient for use with LLM-based metrics.
type LLM struct {
	client *omnillm.ChatClient
	model  string
}

// NewLLM creates a new LLM wrapper for metrics evaluation.
func NewLLM(client *omnillm.ChatClient, model string) *LLM {
	return &LLM{
		client: client,
		model:  model,
	}
}

// Classification represents the result of a classification call.
type Classification struct {
	Label       string `json:"label"`
	Explanation string `json:"explanation,omitempty"`
}

// Classify asks the LLM to classify the input into one of the given labels.
// It uses tool calling to ensure structured output.
func (l *LLM) Classify(ctx context.Context, prompt string, labels []string, includeExplanation bool) (*Classification, error) {
	// Build the classification tool
	tool := buildClassificationTool(labels, includeExplanation)

	req := &provider.ChatCompletionRequest{
		Model: l.model,
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: prompt},
		},
		Tools:      []provider.Tool{tool},
		ToolChoice: map[string]any{"type": "function", "function": map[string]any{"name": "classify"}},
	}

	resp, err := l.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("classification request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Extract tool call result
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) == 0 {
		// Fallback: try to parse from content
		return parseClassificationFromContent(msg.Content, labels)
	}

	// Parse tool call arguments
	var result Classification
	if err := json.Unmarshal([]byte(msg.ToolCalls[0].Function.Arguments), &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool call result: %w", err)
	}

	return &result, nil
}

// GenerateText generates text completion from the LLM.
func (l *LLM) GenerateText(ctx context.Context, prompt string) (string, error) {
	req := &provider.ChatCompletionRequest{
		Model: l.model,
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: prompt},
		},
	}

	resp, err := l.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("text generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// buildClassificationTool creates a tool definition for classification.
func buildClassificationTool(labels []string, includeExplanation bool) provider.Tool {
	properties := map[string]any{
		"label": map[string]any{
			"type":        "string",
			"enum":        labels,
			"description": "The classification label",
		},
	}
	required := []string{"label"}

	if includeExplanation {
		properties["explanation"] = map[string]any{
			"type":        "string",
			"description": "Brief explanation for the classification",
		}
		required = append(required, "explanation")
	}

	return provider.Tool{
		Type: "function",
		Function: provider.ToolSpec{
			Name:        "classify",
			Description: "Classify the input into one of the given categories",
			Parameters: map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}

// parseClassificationFromContent attempts to extract classification from plain text.
func parseClassificationFromContent(content string, labels []string) (*Classification, error) {
	content = strings.ToLower(strings.TrimSpace(content))

	for _, label := range labels {
		if strings.Contains(content, strings.ToLower(label)) {
			return &Classification{Label: label}, nil
		}
	}

	return nil, fmt.Errorf("could not parse classification from content: %s", content)
}

// toString converts any value to string for comparison.
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
