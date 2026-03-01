package metrics

import (
	"github.com/plexusone/omniobserve/llmops"
)

// ExactMatchMetric is a code-based metric that checks for exact string match.
// It compares EvalInput.Output to EvalInput.Expected.
type ExactMatchMetric struct {
	// CaseSensitive controls whether comparison is case-sensitive.
	// Default is true.
	CaseSensitive bool

	// TrimWhitespace controls whether to trim leading/trailing whitespace.
	// Default is false.
	TrimWhitespace bool
}

// NewExactMatchMetric creates a new exact match metric with default settings.
func NewExactMatchMetric() *ExactMatchMetric {
	return &ExactMatchMetric{
		CaseSensitive:  true,
		TrimWhitespace: false,
	}
}

// ExactMatchOption configures the ExactMatchMetric.
type ExactMatchOption func(*ExactMatchMetric)

// WithCaseSensitive sets whether the comparison is case-sensitive.
func WithCaseSensitive(sensitive bool) ExactMatchOption {
	return func(m *ExactMatchMetric) {
		m.CaseSensitive = sensitive
	}
}

// WithTrimWhitespace sets whether to trim whitespace before comparison.
func WithTrimWhitespace(trim bool) ExactMatchOption {
	return func(m *ExactMatchMetric) {
		m.TrimWhitespace = trim
	}
}

// NewExactMatchMetricWithOptions creates a new exact match metric with options.
func NewExactMatchMetricWithOptions(opts ...ExactMatchOption) *ExactMatchMetric {
	m := NewExactMatchMetric()
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Name returns the metric name.
func (m *ExactMatchMetric) Name() string {
	return "exact_match"
}

// Evaluate computes the exact match score.
// Returns 1.0 if Output matches Expected, 0.0 otherwise.
func (m *ExactMatchMetric) Evaluate(input llmops.EvalInput) (llmops.MetricScore, error) {
	output := toString(input.Output)
	expected := toString(input.Expected)

	if m.TrimWhitespace {
		output = trimSpace(output)
		expected = trimSpace(expected)
	}

	var match bool
	if m.CaseSensitive {
		match = output == expected
	} else {
		match = toLower(output) == toLower(expected)
	}

	score := 0.0
	reason := "Output does not match expected"
	if match {
		score = 1.0
		reason = "Output matches expected"
	}

	return llmops.MetricScore{
		Name:   m.Name(),
		Score:  score,
		Reason: reason,
	}, nil
}

// trimSpace removes leading and trailing whitespace.
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isSpace checks if a byte is whitespace.
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// toLower converts a string to lowercase.
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
