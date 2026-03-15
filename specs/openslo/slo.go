// Package openslo provides types for OpenSLO (Service Level Objectives).
//
// OpenSLO is an open standard for defining SLOs in a vendor-agnostic way.
// See: https://github.com/OpenSLO/OpenSLO
//
// This package wraps the official OpenSLO Go SDK via slogo, providing
// helper functions for common SLO patterns.
package openslo

import (
	"github.com/OpenSLO/go-sdk/pkg/openslo"
	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/grokify/mogo/pointer"
	"github.com/grokify/slogo"
	"github.com/grokify/slogo/ontology"
)

// Re-export commonly used types from OpenSLO SDK
type (
	// SLO represents a Service Level Objective.
	SLO = v1.SLO

	// SLOSpec defines the SLO specification.
	SLOSpec = v1.SLOSpec

	// Metadata contains object metadata.
	Metadata = v1.Metadata

	// Labels is a map of label key to values.
	Labels = v1.Labels

	// Service represents an OpenSLO Service definition.
	Service = v1.Service

	// AlertPolicy represents an OpenSLO AlertPolicy.
	AlertPolicy = v1.AlertPolicy

	// AlertCondition represents an OpenSLO AlertCondition.
	AlertCondition = v1.AlertCondition

	// AlertNotificationTarget represents an OpenSLO AlertNotificationTarget.
	AlertNotificationTarget = v1.AlertNotificationTarget

	// Objects is a collection of OpenSLO objects.
	Objects = slogo.Objects
)

// Re-export kind constants
const (
	KindSLO                     = openslo.KindSLO
	KindSLI                     = openslo.KindSLI
	KindService                 = openslo.KindService
	KindAlertPolicy             = openslo.KindAlertPolicy
	KindAlertCondition          = openslo.KindAlertCondition
	KindAlertNotificationTarget = openslo.KindAlertNotificationTarget
)

// APIVersion is the OpenSLO API version.
const APIVersion = v1.APIVersion

// MetricSource identifies a metric for SLI calculation.
// This is a simplified wrapper for building SLIs.
type MetricSource struct {
	// Metric is the metric name/query.
	Metric string

	// Filter is an optional filter expression.
	Filter string
}

// SLOBuilder provides a fluent interface for building SLO objects.
type SLOBuilder struct {
	name        string
	service     string
	description string
	labels      v1.Labels

	ratioMetric     *v1.SLIRatioMetric
	thresholdMetric *v1.SLIMetricSpec
	objectives      []v1.SLOObjective
	timeWindows     []v1.SLOTimeWindow
	alertPolicies   []v1.SLOAlertPolicy
}

// NewSLO creates a new SLO builder with the given name and service.
func NewSLO(name, service string) *SLOBuilder {
	return &SLOBuilder{
		name:    name,
		service: service,
	}
}

// WithDescription sets the SLO description.
func (b *SLOBuilder) WithDescription(desc string) *SLOBuilder {
	b.description = desc
	return b
}

// WithLabels sets the SLO labels.
func (b *SLOBuilder) WithLabels(labels map[string]string) *SLOBuilder {
	b.labels = ontology.NewLabels(labels)
	return b
}

// WithRatioMetric sets a ratio-based SLI (good/total).
func (b *SLOBuilder) WithRatioMetric(good, total MetricSource) *SLOBuilder {
	b.ratioMetric = &v1.SLIRatioMetric{
		Good: &v1.SLIMetricSpec{
			MetricSource: v1.SLIMetricSource{
				Type: "prometheus",
				Spec: map[string]any{
					"query": good.Metric,
				},
			},
		},
		Total: &v1.SLIMetricSpec{
			MetricSource: v1.SLIMetricSource{
				Type: "prometheus",
				Spec: map[string]any{
					"query": total.Metric,
				},
			},
		},
	}
	return b
}

// WithThresholdMetric sets a threshold-based SLI.
func (b *SLOBuilder) WithThresholdMetric(metric, threshold, aggregation string) *SLOBuilder {
	b.thresholdMetric = &v1.SLIMetricSpec{
		MetricSource: v1.SLIMetricSource{
			Type: "prometheus",
			Spec: map[string]any{
				"query": metric,
			},
		},
	}
	return b
}

// AddObjective adds an objective to the SLO.
func (b *SLOBuilder) AddObjective(target float64, timeWindow string) *SLOBuilder {
	tw, _ := v1.ParseDurationShorthand(timeWindow)
	b.objectives = append(b.objectives, v1.SLOObjective{
		Target: pointer.Pointer(target / 100), // Convert percentage to ratio
	})
	b.timeWindows = append(b.timeWindows, v1.SLOTimeWindow{
		Duration:  tw,
		IsRolling: true,
	})
	return b
}

// AddAlertPolicyRef adds a reference to an alert policy.
func (b *SLOBuilder) AddAlertPolicyRef(ref string) *SLOBuilder {
	b.alertPolicies = append(b.alertPolicies, v1.SLOAlertPolicy{
		SLOAlertPolicyRef: &v1.SLOAlertPolicyRef{
			AlertPolicyRef: ref,
		},
	})
	return b
}

// Build creates the SLO.
func (b *SLOBuilder) Build() v1.SLO {
	// Use only the first time window (OpenSLO requires exactly 1)
	var timeWindow []v1.SLOTimeWindow
	if len(b.timeWindows) > 0 {
		timeWindow = []v1.SLOTimeWindow{b.timeWindows[0]}
	}

	spec := v1.SLOSpec{
		Service:         b.service,
		Description:     b.description,
		BudgetingMethod: v1.SLOBudgetingMethodOccurrences,
		Objectives:      b.objectives,
		TimeWindow:      timeWindow,
		AlertPolicies:   b.alertPolicies,
	}

	if b.ratioMetric != nil {
		spec.Indicator = &v1.SLOIndicatorInline{
			Metadata: v1.Metadata{Name: b.name + "-indicator"},
			Spec: v1.SLISpec{
				RatioMetric: b.ratioMetric,
			},
		}
	} else if b.thresholdMetric != nil {
		spec.Indicator = &v1.SLOIndicatorInline{
			Metadata: v1.Metadata{Name: b.name + "-indicator"},
			Spec: v1.SLISpec{
				ThresholdMetric: b.thresholdMetric,
			},
		}
	}

	return v1.NewSLO(
		v1.Metadata{
			Name:   b.name,
			Labels: b.labels,
		},
		spec,
	)
}

// NewAvailabilitySLO creates an availability SLO.
func NewAvailabilitySLO(service, name string, target float64, window string) v1.SLO {
	return NewSLO(name, service).
		WithDescription("Availability SLO: percentage of successful requests").
		WithRatioMetric(
			MetricSource{Metric: "http_requests_total{status!~\"5..\"}"},
			MetricSource{Metric: "http_requests_total"},
		).
		AddObjective(target, window).
		Build()
}

// NewLatencySLO creates a latency SLO.
func NewLatencySLO(service, name, threshold string, target float64, window string) v1.SLO {
	return NewSLO(name, service).
		WithDescription("Latency SLO: percentage of requests below threshold").
		WithThresholdMetric("http_request_duration_seconds", threshold, "p95").
		AddObjective(target, window).
		Build()
}

// NewService creates a new Service definition.
func NewService(name, description string) v1.Service {
	return v1.NewService(
		v1.Metadata{Name: name},
		v1.ServiceSpec{Description: description},
	)
}

// NewLabels creates OpenSLO labels from a map.
func NewLabels(labels map[string]string) v1.Labels {
	return ontology.NewLabels(labels)
}

// Alert helper re-exports from slogo
var (
	// NewAlertCondition creates a new AlertCondition with burn rate configuration.
	NewAlertCondition = slogo.NewAlertCondition

	// NewAlertNotificationTarget creates a new AlertNotificationTarget.
	NewAlertNotificationTarget = slogo.NewAlertNotificationTarget

	// NewAlertPolicyBuilder creates a new AlertPolicyBuilder.
	NewAlertPolicyBuilder = slogo.NewAlertPolicyBuilder

	// StandardBurnRateAlerts returns standard multi-window burn rate alert conditions.
	StandardBurnRateAlerts = slogo.StandardBurnRateAlerts
)

// Severity constants re-exported from slogo
const (
	SeverityCritical = slogo.SeverityCritical
	SeverityHigh     = slogo.SeverityHigh
	SeverityMedium   = slogo.SeverityMedium
	SeverityLow      = slogo.SeverityLow
	SeverityInfo     = slogo.SeverityInfo
)
