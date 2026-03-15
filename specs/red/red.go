// Package red provides types for RED (Rate, Errors, Duration) metrics.
//
// RED metrics are request-oriented metrics popularized by the Prometheus ecosystem:
//   - Rate: Request throughput (requests/sec)
//   - Errors: Failed request count
//   - Duration: Request latency distribution
//
// These map to the Traffic, Errors, and Latency Golden Signals.
package red

import "time"

// MetricType identifies a RED metric category.
type MetricType string

const (
	// MetricRate represents request rate (traffic).
	MetricRate MetricType = "rate"

	// MetricErrors represents error count.
	MetricErrors MetricType = "errors"

	// MetricDuration represents request latency.
	MetricDuration MetricType = "duration"
)

// Definition describes a RED metric set for a service endpoint.
type Definition struct {
	// Name is the metric name prefix (e.g., "http.server.request").
	Name string `json:"name"`

	// Description explains what this metric measures.
	Description string `json:"description,omitempty"`

	// Rate configures the rate metric.
	Rate RateConfig `json:"rate"`

	// Errors configures the error metric.
	Errors ErrorsConfig `json:"errors"`

	// Duration configures the duration metric.
	Duration DurationConfig `json:"duration"`

	// Attributes are common attributes for all metrics.
	Attributes []Attribute `json:"attributes,omitempty"`
}

// RateConfig configures rate (request count) metrics.
type RateConfig struct {
	// Metric is the OTel metric name (default: {name}.count).
	Metric string `json:"metric,omitempty"`

	// Unit is the metric unit (default: "{requests}").
	Unit string `json:"unit,omitempty"`

	// GoldenSignal maps to "traffic".
	GoldenSignal string `json:"golden_signal,omitempty"`
}

// ErrorsConfig configures error metrics.
type ErrorsConfig struct {
	// Metric is the OTel metric name (default: {name}.errors).
	Metric string `json:"metric,omitempty"`

	// Filter defines what constitutes an error (e.g., "http.status_code >= 500").
	Filter string `json:"filter,omitempty"`

	// SLICandidate indicates this can be used as an SLI.
	SLICandidate bool `json:"sli_candidate,omitempty"`

	// GoldenSignal maps to "errors".
	GoldenSignal string `json:"golden_signal,omitempty"`
}

// DurationConfig configures latency/duration metrics.
type DurationConfig struct {
	// Metric is the OTel metric name (default: {name}.duration).
	Metric string `json:"metric,omitempty"`

	// Unit is the metric unit (default: "ms").
	Unit string `json:"unit,omitempty"`

	// Buckets are histogram bucket boundaries in milliseconds.
	Buckets []float64 `json:"buckets,omitempty"`

	// Aggregation specifies how to aggregate (p50, p95, p99).
	Aggregation string `json:"aggregation,omitempty"`

	// SLICandidate indicates this can be used as an SLI.
	SLICandidate bool `json:"sli_candidate,omitempty"`

	// GoldenSignal maps to "latency".
	GoldenSignal string `json:"golden_signal,omitempty"`
}

// Attribute defines a metric attribute.
type Attribute struct {
	// Key is the attribute key.
	Key string `json:"key"`

	// Description explains the attribute.
	Description string `json:"description,omitempty"`

	// Required indicates the attribute must be present.
	Required bool `json:"required,omitempty"`
}

// Observation captures a single RED observation.
type Observation struct {
	// Duration of the request.
	Duration time.Duration

	// Error if the request failed.
	Error error

	// StatusCode is the HTTP status code (if applicable).
	StatusCode int

	// Attributes for this observation.
	Attributes map[string]string
}

// IsError returns true if the observation represents an error.
func (o Observation) IsError() bool {
	if o.Error != nil {
		return true
	}
	return o.StatusCode >= 500
}

// DefaultBuckets returns default histogram buckets for latency in milliseconds.
func DefaultBuckets() []float64 {
	return []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
}

// HTTPServerDefinition returns a standard RED definition for HTTP servers.
func HTTPServerDefinition() Definition {
	return Definition{
		Name:        "http.server.request",
		Description: "HTTP server request metrics",
		Rate: RateConfig{
			Metric:       "http.server.request.count",
			Unit:         "{requests}",
			GoldenSignal: "traffic",
		},
		Errors: ErrorsConfig{
			Metric:       "http.server.request.errors",
			Filter:       "http.status_code >= 500",
			SLICandidate: true,
			GoldenSignal: "errors",
		},
		Duration: DurationConfig{
			Metric:       "http.server.request.duration",
			Unit:         "ms",
			Buckets:      DefaultBuckets(),
			Aggregation:  "p95",
			SLICandidate: true,
			GoldenSignal: "latency",
		},
		Attributes: []Attribute{
			{Key: "service.name", Required: true},
			{Key: "http.method", Required: true},
			{Key: "http.route"},
			{Key: "http.status_code"},
		},
	}
}

// GRPCServerDefinition returns a standard RED definition for gRPC servers.
func GRPCServerDefinition() Definition {
	return Definition{
		Name:        "rpc.server.request",
		Description: "gRPC server request metrics",
		Rate: RateConfig{
			Metric:       "rpc.server.request.count",
			Unit:         "{requests}",
			GoldenSignal: "traffic",
		},
		Errors: ErrorsConfig{
			Metric:       "rpc.server.request.errors",
			Filter:       "rpc.grpc.status_code != 0",
			SLICandidate: true,
			GoldenSignal: "errors",
		},
		Duration: DurationConfig{
			Metric:       "rpc.server.request.duration",
			Unit:         "ms",
			Buckets:      DefaultBuckets(),
			Aggregation:  "p95",
			SLICandidate: true,
			GoldenSignal: "latency",
		},
		Attributes: []Attribute{
			{Key: "service.name", Required: true},
			{Key: "rpc.method", Required: true},
			{Key: "rpc.service"},
			{Key: "rpc.grpc.status_code"},
		},
	}
}
