package metrics

import (
	"regexp"

	"github.com/plexusone/omniobserve/llmops"
)

// RegexMetric is a code-based metric that checks if output matches a regex pattern.
type RegexMetric struct {
	pattern *regexp.Regexp
	name    string
}

// NewRegexMetric creates a new regex metric with the given pattern.
// The pattern is compiled once at creation time.
func NewRegexMetric(pattern string) (*RegexMetric, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &RegexMetric{
		pattern: re,
		name:    "regex_match",
	}, nil
}

// MustRegexMetric creates a new regex metric, panicking on invalid pattern.
func MustRegexMetric(pattern string) *RegexMetric {
	m, err := NewRegexMetric(pattern)
	if err != nil {
		panic(err)
	}
	return m
}

// NewRegexMetricWithName creates a regex metric with a custom name.
func NewRegexMetricWithName(name, pattern string) (*RegexMetric, error) {
	m, err := NewRegexMetric(pattern)
	if err != nil {
		return nil, err
	}
	m.name = name
	return m, nil
}

// Name returns the metric name.
func (m *RegexMetric) Name() string {
	return m.name
}

// Evaluate checks if the output matches the regex pattern.
// Returns 1.0 if there's a match, 0.0 otherwise.
func (m *RegexMetric) Evaluate(input llmops.EvalInput) (llmops.MetricScore, error) {
	output := toString(input.Output)

	matches := m.pattern.FindAllString(output, -1)
	hasMatch := len(matches) > 0

	score := 0.0
	reason := "Output does not match pattern"
	var metadata any

	if hasMatch {
		score = 1.0
		reason = "Output matches pattern"
		metadata = map[string]any{
			"matches":     matches,
			"match_count": len(matches),
		}
	}

	return llmops.MetricScore{
		Name:     m.name,
		Score:    score,
		Reason:   reason,
		Metadata: metadata,
	}, nil
}

// ContainsMetric checks if output contains a specific substring.
type ContainsMetric struct {
	substring     string
	caseSensitive bool
	name          string
}

// NewContainsMetric creates a metric that checks for substring presence.
func NewContainsMetric(substring string, caseSensitive bool) *ContainsMetric {
	return &ContainsMetric{
		substring:     substring,
		caseSensitive: caseSensitive,
		name:          "contains",
	}
}

// Name returns the metric name.
func (m *ContainsMetric) Name() string {
	return m.name
}

// Evaluate checks if output contains the substring.
func (m *ContainsMetric) Evaluate(input llmops.EvalInput) (llmops.MetricScore, error) {
	output := toString(input.Output)
	substr := m.substring

	if !m.caseSensitive {
		output = toLower(output)
		substr = toLower(substr)
	}

	contains := containsSubstring(output, substr)

	score := 0.0
	reason := "Output does not contain substring"
	if contains {
		score = 1.0
		reason = "Output contains substring"
	}

	return llmops.MetricScore{
		Name:   m.name,
		Score:  score,
		Reason: reason,
	}, nil
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
