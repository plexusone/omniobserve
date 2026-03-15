# Observability Specs Package Design

## Overview

This document describes the design for the `specs` package in OmniObserve, which provides Go-first definitions for observability standards including RED, USE, 4 Golden Signals, OpenTelemetry conventions, and OpenSLO.

## Goals

1. **Go-first approach**: Go structs are the source of truth; JSON Schemas are generated
2. **Vendor agnostic**: CoreAuth/CoreAPI use OmniObserve specs without knowing about Datadog/New Relic
3. **Standards compliance**: Align with OpenTelemetry semantic conventions and OpenSLO
4. **Easy integration**: Minimal API surface for applications to emit standardized metrics

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  CoreAuth / CoreAPI                         │
│        (emit metrics using OmniObserve specs)               │
│                                                             │
│   specs.RecordRED(ctx, "auth.token", red.Observation{...})  │
│   specs.RecordUSE(ctx, "auth.sessions", use.Observation{...})│
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     OmniObserve specs/                      │
│                                                             │
│   red/           RED metrics (Rate, Errors, Duration)       │
│   use/           USE metrics (Utilization, Saturation, Err) │
│   golden/        4 Golden Signals mapping                   │
│   openslo/       OpenSLO SLI/SLO definitions                │
│   classes/       Class-based SLO management (at scale)      │
│   otel/          OTel semantic conventions                  │
│   recorder/      Integration with observops.Provider        │
│                                                             │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   observops.Provider                        │
│        (OTLP, Datadog, New Relic, Dynatrace)                │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
omniobserve/
├── specs/
│   ├── red/
│   │   ├── red.go           # RED metric types
│   │   └── red_test.go
│   ├── use/
│   │   ├── use.go           # USE metric types
│   │   └── use_test.go
│   ├── golden/
│   │   ├── golden.go        # 4 Golden Signals types
│   │   └── golden_test.go
│   ├── openslo/
│   │   ├── slo.go           # OpenSLO types
│   │   ├── sli.go           # SLI definitions
│   │   └── openslo_test.go
│   ├── classes/
│   │   ├── classes.go       # Class-based SLO management
│   │   └── classes_test.go
│   ├── otel/
│   │   ├── conventions.go   # OTel semantic conventions
│   │   ├── http.go          # HTTP-specific conventions
│   │   └── system.go        # System-specific conventions
│   ├── recorder/
│   │   ├── recorder.go      # Integration with observops
│   │   └── middleware.go    # HTTP middleware
│   └── schema/
│       └── generate.go      # JSON Schema generation
```

## Type Definitions

### RED Metrics (specs/red/red.go)

```go
package red

import "time"

// MetricType identifies a RED metric category.
type MetricType string

const (
    MetricRate     MetricType = "rate"
    MetricErrors   MetricType = "errors"
    MetricDuration MetricType = "duration"
)

// Definition describes a RED metric set for a service endpoint.
type Definition struct {
    // Name is the metric name prefix (e.g., "http.server.request")
    Name string `json:"name"`

    // Description explains what this metric measures
    Description string `json:"description,omitempty"`

    // Rate configures the rate metric
    Rate RateConfig `json:"rate"`

    // Errors configures the error metric
    Errors ErrorsConfig `json:"errors"`

    // Duration configures the duration metric
    Duration DurationConfig `json:"duration"`

    // Attributes are common attributes for all metrics
    Attributes []Attribute `json:"attributes,omitempty"`
}

// RateConfig configures rate (request count) metrics.
type RateConfig struct {
    // Metric is the OTel metric name (default: {name}.count)
    Metric string `json:"metric,omitempty"`

    // Unit is the metric unit (default: "{requests}")
    Unit string `json:"unit,omitempty"`

    // GoldenSignal maps to traffic
    GoldenSignal string `json:"golden_signal,omitempty"`
}

// ErrorsConfig configures error metrics.
type ErrorsConfig struct {
    // Metric is the OTel metric name (default: {name}.errors)
    Metric string `json:"metric,omitempty"`

    // Filter defines what constitutes an error (e.g., "http.status_code >= 500")
    Filter string `json:"filter,omitempty"`

    // SLICandidate indicates this can be an SLI
    SLICandidate bool `json:"sli_candidate,omitempty"`

    // GoldenSignal maps to errors
    GoldenSignal string `json:"golden_signal,omitempty"`
}

// DurationConfig configures latency/duration metrics.
type DurationConfig struct {
    // Metric is the OTel metric name (default: {name}.duration)
    Metric string `json:"metric,omitempty"`

    // Unit is the metric unit (default: "ms")
    Unit string `json:"unit,omitempty"`

    // Buckets are histogram bucket boundaries
    Buckets []float64 `json:"buckets,omitempty"`

    // Aggregation specifies how to aggregate (p50, p95, p99)
    Aggregation string `json:"aggregation,omitempty"`

    // SLICandidate indicates this can be an SLI
    SLICandidate bool `json:"sli_candidate,omitempty"`

    // GoldenSignal maps to latency
    GoldenSignal string `json:"golden_signal,omitempty"`
}

// Attribute defines a metric attribute.
type Attribute struct {
    Key         string `json:"key"`
    Description string `json:"description,omitempty"`
    Required    bool   `json:"required,omitempty"`
}

// Observation captures a single RED observation.
type Observation struct {
    // Duration of the request (for histogram)
    Duration time.Duration

    // Error if the request failed
    Error error

    // Attributes for this observation
    Attributes map[string]string
}
```

### USE Metrics (specs/use/use.go)

```go
package use

// MetricType identifies a USE metric category.
type MetricType string

const (
    MetricUtilization MetricType = "utilization"
    MetricSaturation  MetricType = "saturation"
    MetricErrors      MetricType = "errors"
)

// ResourceType identifies the type of resource being measured.
type ResourceType string

const (
    ResourceCPU     ResourceType = "cpu"
    ResourceMemory  ResourceType = "memory"
    ResourceDisk    ResourceType = "disk"
    ResourceNetwork ResourceType = "network"
    ResourceQueue   ResourceType = "queue"
    ResourcePool    ResourceType = "pool"
)

// Definition describes USE metrics for a resource.
type Definition struct {
    // Resource is the resource being measured
    Resource ResourceType `json:"resource"`

    // Name is the metric name prefix
    Name string `json:"name"`

    // Description explains what this measures
    Description string `json:"description,omitempty"`

    // Utilization metric configuration
    Utilization *UtilizationConfig `json:"utilization,omitempty"`

    // Saturation metric configuration
    Saturation *SaturationConfig `json:"saturation,omitempty"`

    // Errors metric configuration
    Errors *ErrorsConfig `json:"errors,omitempty"`

    // Attributes are common attributes
    Attributes []Attribute `json:"attributes,omitempty"`
}

// UtilizationConfig configures utilization metrics.
type UtilizationConfig struct {
    // Metric is the OTel metric name
    Metric string `json:"metric"`

    // Unit is typically "1" for ratio or "%" for percentage
    Unit string `json:"unit,omitempty"`

    // GoldenSignal maps to saturation
    GoldenSignal string `json:"golden_signal,omitempty"`
}

// SaturationConfig configures saturation metrics.
type SaturationConfig struct {
    // Metric is the OTel metric name
    Metric string `json:"metric"`

    // Unit is typically count or items
    Unit string `json:"unit,omitempty"`

    // GoldenSignal maps to saturation
    GoldenSignal string `json:"golden_signal,omitempty"`
}

// ErrorsConfig configures resource error metrics.
type ErrorsConfig struct {
    // Metric is the OTel metric name
    Metric string `json:"metric"`

    // Unit is typically count
    Unit string `json:"unit,omitempty"`
}

// Attribute defines a metric attribute.
type Attribute struct {
    Key         string `json:"key"`
    Description string `json:"description,omitempty"`
    Required    bool   `json:"required,omitempty"`
}

// Observation captures a single USE observation.
type Observation struct {
    // Utilization as a ratio 0.0-1.0
    Utilization float64

    // Saturation as queue depth or count
    Saturation float64

    // ErrorCount number of errors
    ErrorCount int64

    // Attributes for this observation
    Attributes map[string]string
}
```

### 4 Golden Signals (specs/golden/golden.go)

```go
package golden

import (
    "github.com/plexusone/omniobserve/specs/red"
    "github.com/plexusone/omniobserve/specs/use"
)

// Signal represents one of the 4 Golden Signals.
type Signal string

const (
    SignalLatency    Signal = "latency"
    SignalTraffic    Signal = "traffic"
    SignalErrors     Signal = "errors"
    SignalSaturation Signal = "saturation"
)

// Definition maps RED/USE metrics to Golden Signals.
type Definition struct {
    // Latency signal configuration
    Latency LatencyConfig `json:"latency"`

    // Traffic signal configuration
    Traffic TrafficConfig `json:"traffic"`

    // Errors signal configuration
    Errors ErrorsConfig `json:"errors"`

    // Saturation signal configuration
    Saturation SaturationConfig `json:"saturation"`
}

// LatencyConfig configures the latency golden signal.
type LatencyConfig struct {
    // Source is "RED.duration"
    Source string `json:"source"`

    // Metric is the OTel metric name
    Metric string `json:"metric"`

    // Aggregation is p50, p95, p99
    Aggregation string `json:"aggregation"`
}

// TrafficConfig configures the traffic golden signal.
type TrafficConfig struct {
    // Source is "RED.rate"
    Source string `json:"source"`

    // Metric is the OTel metric name
    Metric string `json:"metric"`

    // Aggregation is rate
    Aggregation string `json:"aggregation"`
}

// ErrorsConfig configures the errors golden signal.
type ErrorsConfig struct {
    // Source is "RED.errors"
    Source string `json:"source"`

    // Metric is the OTel metric name
    Metric string `json:"metric"`

    // Filter for error conditions
    Filter string `json:"filter,omitempty"`
}

// SaturationConfig configures the saturation golden signal.
type SaturationConfig struct {
    // Source is "USE.utilization" or "USE.saturation"
    Source string `json:"source"`

    // Metrics are the OTel metric names (multiple resources)
    Metrics []string `json:"metrics"`
}

// FromRED creates Golden Signals mapping from RED definition.
func FromRED(def red.Definition) Definition {
    return Definition{
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
    }
}

// WithUSE adds USE metrics to saturation signal.
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
```

### OpenSLO (specs/openslo/slo.go)

```go
package openslo

// APIVersion is the OpenSLO API version.
const APIVersion = "openslo/v1"

// Kind represents the OpenSLO object kind.
type Kind string

const (
    KindSLO     Kind = "SLO"
    KindSLI     Kind = "SLI"
    KindService Kind = "Service"
)

// SLO represents a Service Level Objective.
type SLO struct {
    APIVersion string   `json:"apiVersion"`
    Kind       Kind     `json:"kind"`
    Metadata   Metadata `json:"metadata"`
    Spec       SLOSpec  `json:"spec"`
}

// Metadata contains object metadata.
type Metadata struct {
    Name        string            `json:"name"`
    DisplayName string            `json:"displayName,omitempty"`
    Labels      map[string]string `json:"labels,omitempty"`
}

// SLOSpec defines the SLO specification.
type SLOSpec struct {
    Service    string       `json:"service"`
    Indicator  SLISpec      `json:"indicator"`
    Objectives []Objective  `json:"objectives"`
}

// SLISpec defines the Service Level Indicator.
type SLISpec struct {
    Metadata      Metadata     `json:"metadata,omitempty"`
    RatioMetric   *RatioMetric `json:"ratioMetric,omitempty"`
    ThresholdMetric *ThresholdMetric `json:"thresholdMetric,omitempty"`
}

// RatioMetric defines a ratio-based SLI (good/total).
type RatioMetric struct {
    Good  MetricSource `json:"good"`
    Total MetricSource `json:"total"`
}

// ThresholdMetric defines a threshold-based SLI.
type ThresholdMetric struct {
    Metric    string `json:"metric"`
    Threshold string `json:"threshold"`
}

// MetricSource identifies a metric for SLI calculation.
type MetricSource struct {
    Metric    string `json:"metric"`
    Filter    string `json:"filter,omitempty"`
    Threshold string `json:"threshold,omitempty"`
}

// Objective defines an SLO target.
type Objective struct {
    Target     float64 `json:"target"`
    TimeWindow string  `json:"timeWindow"`
}

// NewAvailabilitySLO creates an availability SLO from RED metrics.
func NewAvailabilitySLO(service, name string, target float64, window string) SLO {
    return SLO{
        APIVersion: APIVersion,
        Kind:       KindSLO,
        Metadata:   Metadata{Name: name},
        Spec: SLOSpec{
            Service: service,
            Indicator: SLISpec{
                RatioMetric: &RatioMetric{
                    Good:  MetricSource{Metric: "http.server.request.count", Filter: "http.status_code < 500"},
                    Total: MetricSource{Metric: "http.server.request.count"},
                },
            },
            Objectives: []Objective{{Target: target, TimeWindow: window}},
        },
    }
}

// NewLatencySLO creates a latency SLO from RED metrics.
func NewLatencySLO(service, name string, threshold string, target float64, window string) SLO {
    return SLO{
        APIVersion: APIVersion,
        Kind:       KindSLO,
        Metadata:   Metadata{Name: name},
        Spec: SLOSpec{
            Service: service,
            Indicator: SLISpec{
                ThresholdMetric: &ThresholdMetric{
                    Metric:    "http.server.request.duration",
                    Threshold: threshold,
                },
            },
            Objectives: []Objective{{Target: target, TimeWindow: window}},
        },
    }
}
```

### Recorder Integration (specs/recorder/recorder.go)

```go
package recorder

import (
    "context"
    "time"

    "github.com/plexusone/omniobserve/observops"
    "github.com/plexusone/omniobserve/specs/red"
    "github.com/plexusone/omniobserve/specs/use"
)

// Recorder records RED and USE metrics using an observops.Provider.
type Recorder struct {
    provider observops.Provider

    // RED metrics
    requestCount    observops.Counter
    requestErrors   observops.Counter
    requestDuration observops.Histogram

    // USE metrics
    utilization observops.Gauge
    saturation  observops.Gauge
    errors      observops.Counter
}

// New creates a Recorder with the given provider.
func New(provider observops.Provider, serviceName string) (*Recorder, error) {
    meter := provider.Meter()

    requestCount, err := meter.Counter(serviceName+".request.count",
        observops.WithDescription("Total requests"),
        observops.WithUnit("{requests}"))
    if err != nil {
        return nil, err
    }

    requestErrors, err := meter.Counter(serviceName+".request.errors",
        observops.WithDescription("Failed requests"),
        observops.WithUnit("{requests}"))
    if err != nil {
        return nil, err
    }

    requestDuration, err := meter.Histogram(serviceName+".request.duration",
        observops.WithDescription("Request latency"),
        observops.WithUnit("ms"))
    if err != nil {
        return nil, err
    }

    utilization, err := meter.Gauge(serviceName+".utilization",
        observops.WithDescription("Resource utilization"),
        observops.WithUnit("1"))
    if err != nil {
        return nil, err
    }

    saturation, err := meter.Gauge(serviceName+".saturation",
        observops.WithDescription("Resource saturation"),
        observops.WithUnit("{items}"))
    if err != nil {
        return nil, err
    }

    resourceErrors, err := meter.Counter(serviceName+".resource.errors",
        observops.WithDescription("Resource errors"),
        observops.WithUnit("{errors}"))
    if err != nil {
        return nil, err
    }

    return &Recorder{
        provider:        provider,
        requestCount:    requestCount,
        requestErrors:   requestErrors,
        requestDuration: requestDuration,
        utilization:     utilization,
        saturation:      saturation,
        errors:          resourceErrors,
    }, nil
}

// RecordRED records a RED observation.
func (r *Recorder) RecordRED(ctx context.Context, obs red.Observation) {
    attrs := toKeyValues(obs.Attributes)
    opts := observops.WithAttributes(attrs...)

    // Always record count (rate is derived)
    r.requestCount.Add(ctx, 1, opts)

    // Record duration
    r.requestDuration.Record(ctx, float64(obs.Duration.Milliseconds()), opts)

    // Record errors if present
    if obs.Error != nil {
        r.requestErrors.Add(ctx, 1, opts)
    }
}

// RecordUSE records a USE observation.
func (r *Recorder) RecordUSE(ctx context.Context, obs use.Observation) {
    attrs := toKeyValues(obs.Attributes)
    opts := observops.WithAttributes(attrs...)

    r.utilization.Record(ctx, obs.Utilization, opts)
    r.saturation.Record(ctx, obs.Saturation, opts)

    if obs.ErrorCount > 0 {
        r.errors.Add(ctx, float64(obs.ErrorCount), opts)
    }
}

func toKeyValues(attrs map[string]string) []observops.KeyValue {
    kvs := make([]observops.KeyValue, 0, len(attrs))
    for k, v := range attrs {
        kvs = append(kvs, observops.Attribute(k, v))
    }
    return kvs
}
```

## JSON Schema Generation

JSON Schemas will be generated from Go structs using `github.com/invopop/jsonschema`:

```go
// specs/schema/generate.go
package schema

import (
    "encoding/json"

    "github.com/invopop/jsonschema"
    "github.com/plexusone/omniobserve/specs/red"
    "github.com/plexusone/omniobserve/specs/use"
    "github.com/plexusone/omniobserve/specs/golden"
    "github.com/plexusone/omniobserve/specs/openslo"
)

// GenerateREDSchema generates JSON Schema for RED definitions.
func GenerateREDSchema() ([]byte, error) {
    r := jsonschema.Reflector{}
    schema := r.Reflect(&red.Definition{})
    return json.MarshalIndent(schema, "", "  ")
}

// Similarly for USE, Golden, OpenSLO...
```

## Usage Example

```go
package main

import (
    "context"
    "time"

    "github.com/plexusone/omniobserve/observops"
    _ "github.com/plexusone/omniobserve/observops/otlp"
    "github.com/plexusone/omniobserve/specs/recorder"
    "github.com/plexusone/omniobserve/specs/red"
)

func main() {
    // Open provider
    provider, _ := observops.Open("otlp",
        observops.WithEndpoint("localhost:4317"),
        observops.WithServiceName("coreauth"),
    )
    defer provider.Shutdown(context.Background())

    // Create recorder
    rec, _ := recorder.New(provider, "coreauth")

    // Record a request (RED metrics)
    start := time.Now()
    err := processRequest()
    rec.RecordRED(ctx, red.Observation{
        Duration: time.Since(start),
        Error:    err,
        Attributes: map[string]string{
            "endpoint": "/oauth/token",
            "method":   "POST",
        },
    })
}
```

## Implementation Plan

1. **Phase 1: Core Types**
   - Create `specs/red/red.go` with RED metric types
   - Create `specs/use/use.go` with USE metric types
   - Create `specs/golden/golden.go` with 4GS mapping
   - Add unit tests

2. **Phase 2: OpenSLO**
   - Create `specs/openslo/slo.go` with OpenSLO types
   - Create helper functions for common SLO patterns
   - Add validation

3. **Phase 3: Recorder**
   - Create `specs/recorder/recorder.go` for observops integration
   - Create HTTP middleware for automatic RED recording
   - Add examples

4. **Phase 4: Schema Generation**
   - Add `github.com/invopop/jsonschema` dependency
   - Create `specs/schema/generate.go`
   - Generate and commit JSON schemas

## Design Decisions

1. **Go structs as source of truth**: Ensures type safety and IDE support
2. **Separate packages per model**: Allows independent versioning and clear imports
3. **Recorder abstraction**: Applications don't need to know about metric implementation
4. **OpenSLO compatibility**: SLO definitions can be exported for external tools
5. **JSON Schema generation**: Enables CI validation and external tooling

### Class-Based SLOs (specs/classes/classes.go)

For scaling observability across many services/endpoints, the classes package provides class-based SLO management:

```go
package classes

// ClassLevel represents endpoint criticality tier.
type ClassLevel string

const (
    ClassCritical   ClassLevel = "critical"    // High-impact: login, checkout, payments
    ClassNormal     ClassLevel = "normal"      // Standard: profile, search
    ClassBestEffort ClassLevel = "best_effort" // Low-priority: recommendations
)

// ServiceSpec defines observability configuration for a service.
type ServiceSpec struct {
    Service      string     `json:"service"`
    Owner        string     `json:"owner,omitempty"`
    MetricsModel []string   `json:"metrics_model,omitempty"`
    Classes      []Class    `json:"classes"`
    DefaultClass ClassLevel `json:"default_class,omitempty"`
}

// Class defines an endpoint class with SLO configuration.
type Class struct {
    Name               ClassLevel         `json:"name"`
    Description        string             `json:"description,omitempty"`
    SLOTemplate        string             `json:"slo_template"`
    Endpoints          []string           `json:"endpoints"`
    ThresholdOverrides ThresholdOverrides `json:"threshold_overrides,omitempty"`
}

// ThresholdOverrides allows per-class customization of SLO targets.
type ThresholdOverrides struct {
    Latency      string  `json:"latency,omitempty"`      // e.g., "200ms"
    Availability float64 `json:"availability,omitempty"` // e.g., 99.9
    ErrorRate    float64 `json:"error_rate,omitempty"`
    TimeWindow   string  `json:"time_window,omitempty"`  // e.g., "30d"
}
```

**Usage Example:**

```go
spec := classes.NewServiceSpec("acme-web", "web-platform").
    WithMetricsModel("RED", "USE").
    AddClass(classes.NewCriticalClass("/login", "/checkout", "/payments")).
    AddClass(classes.NewNormalClass("/profile/*", "/search")).
    AddClass(classes.NewBestEffortClass("/recommendations", "/analytics/**"))

// Classify an endpoint
class := spec.ClassifyEndpoint("/profile/settings")
// class.Name = "normal"

// Generate OpenSLO objects
slos := spec.GenerateSLOs()
```

**Endpoint Patterns:**

- Exact match: `/login` matches `/login`
- Single wildcard: `/profile/*` matches `/profile/settings` (one level)
- Double wildcard: `/api/**` matches `/api/v1/users/123` (recursive)

**Benefits:**

1. **Scalable**: Hundreds of endpoints share a few SLO templates
2. **Maintainable**: Changing a class template updates all endpoints
3. **Flexible**: Threshold overrides allow fine-grained tuning
4. **Automated**: Generate OpenSLO objects from class definitions

## Future Considerations

1. **Terraform generation**: Generate Terraform for New Relic/Datadog dashboards and alerts
2. **Dashboard templates**: Pre-built Golden Signals dashboards
3. **SLO validation**: Runtime SLO tracking and error budget calculation
4. **Service catalog**: Registry of services with their observability specs
5. **Central spec repo**: Pattern for centralized specs with service-specific overrides
