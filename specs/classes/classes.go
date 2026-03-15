// Package classes provides types for class-based SLO management.
//
// Class-based SLOs allow scaling observability by grouping endpoints into
// criticality tiers (critical, normal, best-effort) and mapping each class
// to SLO templates with threshold overrides.
//
// This approach is used by large services to avoid per-endpoint SLO configuration
// while maintaining meaningful SLOs aligned with business impact.
package classes

import (
	"path"
	"strings"

	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/grokify/mogo/pointer"
	"github.com/grokify/slogo/ontology"
)

// ClassLevel represents endpoint criticality tier.
type ClassLevel string

const (
	// ClassCritical is for high-impact endpoints (login, checkout, payments).
	ClassCritical ClassLevel = "critical"

	// ClassNormal is for standard endpoints (profile, search).
	ClassNormal ClassLevel = "normal"

	// ClassBestEffort is for low-priority endpoints (recommendations, analytics).
	ClassBestEffort ClassLevel = "best_effort"
)

// ServiceSpec defines observability configuration for a service.
type ServiceSpec struct {
	// Service is the service name.
	Service string `json:"service"`

	// Owner is the team or person responsible for the service.
	Owner string `json:"owner,omitempty"`

	// MetricsModel specifies which metrics frameworks to use (RED, USE).
	MetricsModel []string `json:"metrics_model,omitempty"`

	// Classes defines endpoint classification and SLO mapping.
	Classes []Class `json:"classes"`

	// DefaultClass is the fallback class for unmatched endpoints.
	DefaultClass ClassLevel `json:"default_class,omitempty"`
}

// Class defines an endpoint class with SLO configuration.
type Class struct {
	// Name is the class identifier (critical, normal, best_effort).
	Name ClassLevel `json:"name"`

	// Description explains the class purpose.
	Description string `json:"description,omitempty"`

	// SLOTemplate references an SLO template to use.
	SLOTemplate string `json:"slo_template"`

	// Endpoints are the endpoint patterns for this class.
	// Supports wildcards: /profile/* matches /profile/settings
	Endpoints []string `json:"endpoints"`

	// ThresholdOverrides customizes the SLO template for this class.
	ThresholdOverrides ThresholdOverrides `json:"threshold_overrides,omitempty"`
}

// ThresholdOverrides allows per-class customization of SLO targets.
type ThresholdOverrides struct {
	// Latency is the maximum acceptable latency (e.g., "200ms").
	Latency string `json:"latency,omitempty"`

	// Availability is the target availability percentage (e.g., 99.9).
	Availability float64 `json:"availability,omitempty"`

	// ErrorRate is the maximum acceptable error rate percentage.
	ErrorRate float64 `json:"error_rate,omitempty"`

	// TimeWindow is the SLO time window (e.g., "30d").
	TimeWindow string `json:"time_window,omitempty"`
}

// SLOTemplate defines a reusable SLO configuration.
type SLOTemplate struct {
	// Name is the template identifier.
	Name string `json:"name"`

	// Description explains the template.
	Description string `json:"description,omitempty"`

	// IndicatorType is "availability", "latency", or "custom".
	IndicatorType string `json:"indicator_type"`

	// Defaults are the default threshold values.
	Defaults ThresholdOverrides `json:"defaults"`
}

// NewServiceSpec creates a new service spec.
func NewServiceSpec(service, owner string) *ServiceSpec {
	return &ServiceSpec{
		Service:      service,
		Owner:        owner,
		MetricsModel: []string{"RED"},
		Classes:      []Class{},
		DefaultClass: ClassNormal,
	}
}

// WithMetricsModel sets the metrics frameworks.
func (s *ServiceSpec) WithMetricsModel(models ...string) *ServiceSpec {
	s.MetricsModel = models
	return s
}

// AddClass adds a class definition.
func (s *ServiceSpec) AddClass(c Class) *ServiceSpec {
	s.Classes = append(s.Classes, c)
	return s
}

// WithDefaultClass sets the fallback class for unmatched endpoints.
func (s *ServiceSpec) WithDefaultClass(level ClassLevel) *ServiceSpec {
	s.DefaultClass = level
	return s
}

// NewCriticalClass creates a critical class with common defaults.
func NewCriticalClass(endpoints ...string) Class {
	return Class{
		Name:        ClassCritical,
		Description: "High-impact endpoints with strict SLOs",
		SLOTemplate: "critical-endpoint",
		Endpoints:   endpoints,
		ThresholdOverrides: ThresholdOverrides{
			Latency:      "200ms",
			Availability: 99.9,
			TimeWindow:   "30d",
		},
	}
}

// NewNormalClass creates a normal class with common defaults.
func NewNormalClass(endpoints ...string) Class {
	return Class{
		Name:        ClassNormal,
		Description: "Standard endpoints with moderate SLOs",
		SLOTemplate: "normal-endpoint",
		Endpoints:   endpoints,
		ThresholdOverrides: ThresholdOverrides{
			Latency:      "500ms",
			Availability: 99.5,
			TimeWindow:   "30d",
		},
	}
}

// NewBestEffortClass creates a best-effort class with relaxed defaults.
func NewBestEffortClass(endpoints ...string) Class {
	return Class{
		Name:        ClassBestEffort,
		Description: "Low-priority endpoints with relaxed SLOs",
		SLOTemplate: "best-effort-endpoint",
		Endpoints:   endpoints,
		ThresholdOverrides: ThresholdOverrides{
			Latency:      "1000ms",
			Availability: 99.0,
			TimeWindow:   "30d",
		},
	}
}

// ClassifyEndpoint returns the class for an endpoint path.
func (s *ServiceSpec) ClassifyEndpoint(endpoint string) *Class {
	for i := range s.Classes {
		c := &s.Classes[i]
		for _, pattern := range c.Endpoints {
			if matchEndpoint(pattern, endpoint) {
				return c
			}
		}
	}
	// Return default class if configured
	if s.DefaultClass != "" {
		for i := range s.Classes {
			if s.Classes[i].Name == s.DefaultClass {
				return &s.Classes[i]
			}
		}
	}
	return nil
}

// matchEndpoint checks if an endpoint matches a pattern.
// Supports:
//   - Exact match: /login matches /login
//   - Wildcard suffix: /profile/* matches /profile/settings
//   - Double wildcard: /api/** matches /api/v1/users/123
func matchEndpoint(pattern, endpoint string) bool {
	// Exact match
	if pattern == endpoint {
		return true
	}

	// Double wildcard (recursive)
	if prefix, ok := strings.CutSuffix(pattern, "/**"); ok {
		// Must have at least one character after prefix (e.g., /api/** matches /api/v1 but not /api)
		return strings.HasPrefix(endpoint, prefix+"/")
	}

	// Single wildcard (one level)
	if prefix, ok := strings.CutSuffix(pattern, "/*"); ok {
		if !strings.HasPrefix(endpoint, prefix+"/") {
			return false
		}
		rest := strings.TrimPrefix(endpoint, prefix+"/")
		// Should not contain another slash
		return !strings.Contains(rest, "/")
	}

	// Path match using standard library
	matched, _ := path.Match(pattern, endpoint)
	return matched
}

// GenerateSLOs generates OpenSLO objects for all endpoints in the service.
func (s *ServiceSpec) GenerateSLOs() []v1.SLO {
	var slos []v1.SLO

	for _, c := range s.Classes {
		for _, ep := range c.Endpoints {
			// Skip wildcard patterns - they need to be expanded first
			if strings.Contains(ep, "*") {
				// Generate a class-level SLO instead
				slo := s.generateClassSLO(c)
				slos = append(slos, slo)
				break
			}
			slo := s.generateEndpointSLO(c, ep)
			slos = append(slos, slo)
		}
	}

	return slos
}

// generateClassSLO generates an SLO for the entire class.
func (s *ServiceSpec) generateClassSLO(c Class) v1.SLO {
	name := s.Service + "-" + string(c.Name) + "-availability"

	window := c.ThresholdOverrides.TimeWindow
	if window == "" {
		window = "30d"
	}
	tw, _ := v1.ParseDurationShorthand(window)

	// Convert availability percentage to ratio (99.9 -> 0.999)
	target := c.ThresholdOverrides.Availability / 100

	return v1.NewSLO(
		v1.Metadata{
			Name: name,
			Labels: ontology.NewLabels(map[string]string{
				"class":   string(c.Name),
				"service": s.Service,
				"owner":   s.Owner,
			}),
		},
		v1.SLOSpec{
			Service:         s.Service,
			Description:     "Availability SLO for " + string(c.Name) + " endpoints",
			BudgetingMethod: v1.SLOBudgetingMethodOccurrences,
			Indicator: &v1.SLOIndicatorInline{
				Spec: v1.SLISpec{
					RatioMetric: &v1.SLIRatioMetric{
						Good: &v1.SLIMetricSpec{
							MetricSource: v1.SLIMetricSource{
								Type: "prometheus",
								Spec: map[string]any{
									"query": "http_requests_total{status!~\"5..\", class=\"" + string(c.Name) + "\"}",
								},
							},
						},
						Total: &v1.SLIMetricSpec{
							MetricSource: v1.SLIMetricSource{
								Type: "prometheus",
								Spec: map[string]any{
									"query": "http_requests_total{class=\"" + string(c.Name) + "\"}",
								},
							},
						},
					},
				},
			},
			TimeWindow: []v1.SLOTimeWindow{{Duration: tw, IsRolling: true}},
			Objectives: []v1.SLOObjective{
				{
					Target: pointer.Pointer(target),
				},
			},
		},
	)
}

// generateEndpointSLO generates an SLO for a specific endpoint.
func (s *ServiceSpec) generateEndpointSLO(c Class, endpoint string) v1.SLO {
	// Sanitize endpoint for name
	safeName := strings.ReplaceAll(endpoint, "/", "-")
	safeName = strings.Trim(safeName, "-")
	name := s.Service + "-" + safeName + "-availability"

	window := c.ThresholdOverrides.TimeWindow
	if window == "" {
		window = "30d"
	}
	tw, _ := v1.ParseDurationShorthand(window)

	// Convert availability percentage to ratio (99.9 -> 0.999)
	target := c.ThresholdOverrides.Availability / 100

	return v1.NewSLO(
		v1.Metadata{
			Name: name,
			Labels: ontology.NewLabels(map[string]string{
				"class":    string(c.Name),
				"endpoint": endpoint,
				"service":  s.Service,
				"owner":    s.Owner,
			}),
		},
		v1.SLOSpec{
			Service:         s.Service,
			Description:     "Availability SLO for " + endpoint,
			BudgetingMethod: v1.SLOBudgetingMethodOccurrences,
			Indicator: &v1.SLOIndicatorInline{
				Spec: v1.SLISpec{
					RatioMetric: &v1.SLIRatioMetric{
						Good: &v1.SLIMetricSpec{
							MetricSource: v1.SLIMetricSource{
								Type: "prometheus",
								Spec: map[string]any{
									"query": "http_requests_total{status!~\"5..\", route=\"" + endpoint + "\"}",
								},
							},
						},
						Total: &v1.SLIMetricSpec{
							MetricSource: v1.SLIMetricSource{
								Type: "prometheus",
								Spec: map[string]any{
									"query": "http_requests_total{route=\"" + endpoint + "\"}",
								},
							},
						},
					},
				},
			},
			TimeWindow: []v1.SLOTimeWindow{{Duration: tw, IsRolling: true}},
			Objectives: []v1.SLOObjective{
				{
					Target: pointer.Pointer(target),
				},
			},
		},
	)
}

// StandardTemplates returns commonly used SLO templates.
func StandardTemplates() []SLOTemplate {
	return []SLOTemplate{
		{
			Name:          "critical-endpoint",
			Description:   "Strict SLOs for high-impact endpoints",
			IndicatorType: "availability",
			Defaults: ThresholdOverrides{
				Latency:      "200ms",
				Availability: 99.9,
				TimeWindow:   "30d",
			},
		},
		{
			Name:          "normal-endpoint",
			Description:   "Standard SLOs for general endpoints",
			IndicatorType: "availability",
			Defaults: ThresholdOverrides{
				Latency:      "500ms",
				Availability: 99.5,
				TimeWindow:   "30d",
			},
		},
		{
			Name:          "best-effort-endpoint",
			Description:   "Relaxed SLOs for low-priority endpoints",
			IndicatorType: "availability",
			Defaults: ThresholdOverrides{
				Latency:      "1000ms",
				Availability: 99.0,
				TimeWindow:   "30d",
			},
		},
		{
			Name:          "latency-sensitive",
			Description:   "Latency-focused SLOs for performance-critical endpoints",
			IndicatorType: "latency",
			Defaults: ThresholdOverrides{
				Latency:    "100ms",
				TimeWindow: "7d",
			},
		},
	}
}
