// Package golden provides types for the 4 Golden Signals.
//
// The 4 Golden Signals are defined by Google Site Reliability Engineering:
//   - Latency: Time to service a request
//   - Traffic: Demand on the system (requests/sec)
//   - Errors: Rate of failed requests
//   - Saturation: How "full" the system is
//
// This package maps RED and USE metrics to the Golden Signals for unified dashboards.
package golden

import (
	"github.com/plexusone/omniobserve/specs/red"
	"github.com/plexusone/omniobserve/specs/use"
)

// Signal represents one of the 4 Golden Signals.
type Signal string

const (
	// SignalLatency represents request latency.
	SignalLatency Signal = "latency"

	// SignalTraffic represents request rate/throughput.
	SignalTraffic Signal = "traffic"

	// SignalErrors represents error rate.
	SignalErrors Signal = "errors"

	// SignalSaturation represents resource utilization/saturation.
	SignalSaturation Signal = "saturation"
)

// Definition maps RED/USE metrics to Golden Signals.
type Definition struct {
	// ServiceName is the service this definition applies to.
	ServiceName string `json:"service_name"`

	// Latency signal configuration.
	Latency LatencyConfig `json:"latency"`

	// Traffic signal configuration.
	Traffic TrafficConfig `json:"traffic"`

	// Errors signal configuration.
	Errors ErrorsConfig `json:"errors"`

	// Saturation signal configuration.
	Saturation SaturationConfig `json:"saturation"`
}

// LatencyConfig configures the latency golden signal.
type LatencyConfig struct {
	// Source is "RED.duration".
	Source string `json:"source"`

	// Metric is the OTel metric name.
	Metric string `json:"metric"`

	// Aggregation is p50, p95, p99.
	Aggregation string `json:"aggregation"`
}

// TrafficConfig configures the traffic golden signal.
type TrafficConfig struct {
	// Source is "RED.rate".
	Source string `json:"source"`

	// Metric is the OTel metric name.
	Metric string `json:"metric"`

	// Aggregation is "rate".
	Aggregation string `json:"aggregation"`
}

// ErrorsConfig configures the errors golden signal.
type ErrorsConfig struct {
	// Source is "RED.errors".
	Source string `json:"source"`

	// Metric is the OTel metric name.
	Metric string `json:"metric"`

	// Filter for error conditions.
	Filter string `json:"filter,omitempty"`
}

// SaturationConfig configures the saturation golden signal.
type SaturationConfig struct {
	// Source is "USE.utilization" or "USE.saturation".
	Source string `json:"source"`

	// Metrics are the OTel metric names (multiple resources).
	Metrics []string `json:"metrics"`
}

// FromRED creates Golden Signals mapping from a RED definition.
func FromRED(serviceName string, def red.Definition) Definition {
	return Definition{
		ServiceName: serviceName,
		Latency: LatencyConfig{
			Source:      "RED.duration",
			Metric:      def.Duration.Metric,
			Aggregation: def.Duration.Aggregation,
		},
		Traffic: TrafficConfig{
			Source:      "RED.rate",
			Metric:      def.Rate.Metric,
			Aggregation: "rate",
		},
		Errors: ErrorsConfig{
			Source: "RED.errors",
			Metric: def.Errors.Metric,
			Filter: def.Errors.Filter,
		},
		Saturation: SaturationConfig{
			Source:  "USE.utilization",
			Metrics: []string{},
		},
	}
}

// WithUSE adds USE metrics to the saturation signal.
func (d Definition) WithUSE(defs ...use.Definition) Definition {
	metrics := make([]string, 0)
	for _, def := range defs {
		if def.Utilization != nil {
			metrics = append(metrics, def.Utilization.Metric)
		}
		if def.Saturation != nil {
			metrics = append(metrics, def.Saturation.Metric)
		}
	}
	d.Saturation = SaturationConfig{
		Source:  "USE.utilization",
		Metrics: metrics,
	}
	return d
}

// AddSaturationMetric adds a metric to the saturation signal.
func (d *Definition) AddSaturationMetric(metric string) {
	d.Saturation.Metrics = append(d.Saturation.Metrics, metric)
}

// NewHTTPServerGoldenSignals creates Golden Signals for an HTTP server.
func NewHTTPServerGoldenSignals(serviceName string) Definition {
	redDef := red.HTTPServerDefinition()
	d := FromRED(serviceName, redDef)
	return d.WithUSE(
		use.CPUDefinition(),
		use.MemoryDefinition(),
	)
}

// NewGRPCServerGoldenSignals creates Golden Signals for a gRPC server.
func NewGRPCServerGoldenSignals(serviceName string) Definition {
	redDef := red.GRPCServerDefinition()
	d := FromRED(serviceName, redDef)
	return d.WithUSE(
		use.CPUDefinition(),
		use.MemoryDefinition(),
		use.GoroutineDefinition(),
	)
}
