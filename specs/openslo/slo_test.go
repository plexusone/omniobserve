package openslo

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
)

// floatEquals compares two float64 values with a tolerance for floating point arithmetic.
func floatEquals(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestNewSLO(t *testing.T) {
	slo := NewSLO("test-slo", "test-service").
		WithRatioMetric(
			MetricSource{Metric: "good"},
			MetricSource{Metric: "total"},
		).
		AddObjective(99.9, "30d").
		Build()

	if slo.APIVersion != APIVersion {
		t.Errorf("unexpected APIVersion: %s", slo.APIVersion)
	}

	if slo.Kind != openslo.KindSLO {
		t.Errorf("unexpected Kind: %s", slo.Kind)
	}

	if slo.Metadata.Name != "test-slo" {
		t.Errorf("unexpected Name: %s", slo.Metadata.Name)
	}

	if slo.Spec.Service != "test-service" {
		t.Errorf("unexpected Service: %s", slo.Spec.Service)
	}
}

func TestNewAvailabilitySLO(t *testing.T) {
	slo := NewAvailabilitySLO("my-service", "availability-slo", 99.9, "30d")

	if slo.Spec.Indicator == nil || slo.Spec.Indicator.Spec.RatioMetric == nil {
		t.Fatal("expected RatioMetric to be set")
	}

	if len(slo.Spec.Objectives) != 1 {
		t.Fatalf("expected 1 objective, got %d", len(slo.Spec.Objectives))
	}

	// Target is now a ratio (0.999) not percentage
	// Use tolerance for floating point comparison (99.9/100 may produce 0.9990000000000001)
	if slo.Spec.Objectives[0].Target == nil {
		t.Error("expected target to be set")
	} else if !floatEquals(*slo.Spec.Objectives[0].Target, 0.999, 1e-9) {
		t.Errorf("unexpected target: %v (expected 0.999)", *slo.Spec.Objectives[0].Target)
	}
}

func TestNewLatencySLO(t *testing.T) {
	slo := NewLatencySLO("my-service", "latency-slo", "<200ms", 99.0, "7d")

	if slo.Spec.Indicator == nil || slo.Spec.Indicator.Spec.ThresholdMetric == nil {
		t.Fatal("expected ThresholdMetric to be set")
	}

	if slo.Metadata.Name != "latency-slo" {
		t.Errorf("unexpected name: %s", slo.Metadata.Name)
	}
}

func TestSLO_JSON(t *testing.T) {
	slo := NewAvailabilitySLO("checkout", "checkout-availability", 99.9, "30d")

	data, err := json.MarshalIndent(slo, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal SLO: %v", err)
	}

	// Verify it contains expected fields
	jsonStr := string(data)
	expectedFields := []string{
		`"apiVersion"`,
		`"kind"`,
		`"name"`,
		`"service"`,
		`"target"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing expected field: %s\nJSON: %s", field, jsonStr)
		}
	}
}

func TestSLOBuilder_Fluent(t *testing.T) {
	slo := NewSLO("test", "svc").
		WithDescription("Test SLO").
		WithRatioMetric(
			MetricSource{Metric: "good_requests"},
			MetricSource{Metric: "total_requests"},
		).
		AddObjective(99.9, "30d").
		Build()

	if slo.Spec.Description != "Test SLO" {
		t.Errorf("unexpected description: %s", slo.Spec.Description)
	}

	if len(slo.Spec.Objectives) != 1 {
		t.Errorf("expected 1 objective, got %d", len(slo.Spec.Objectives))
	}
}

func TestSLOBuilder_WithLabels(t *testing.T) {
	slo := NewSLO("test", "svc").
		WithLabels(map[string]string{
			"team":        "platform",
			"environment": "production",
		}).
		WithRatioMetric(
			MetricSource{Metric: "good"},
			MetricSource{Metric: "total"},
		).
		AddObjective(99.9, "30d").
		Build()

	if len(slo.Metadata.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(slo.Metadata.Labels))
	}
}

func TestNewService(t *testing.T) {
	svc := NewService("my-service", "A test service")

	if svc.Kind != openslo.KindService {
		t.Errorf("unexpected Kind: %s", svc.Kind)
	}

	if svc.Metadata.Name != "my-service" {
		t.Errorf("unexpected Name: %s", svc.Metadata.Name)
	}

	if svc.Spec.Description != "A test service" {
		t.Errorf("unexpected Description: %s", svc.Spec.Description)
	}

	// Validate service
	if err := svc.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}

func TestAlertHelpers(t *testing.T) {
	// Test StandardBurnRateAlerts
	alerts := StandardBurnRateAlerts()
	if len(alerts) != 2 {
		t.Errorf("expected 2 standard alerts, got %d", len(alerts))
	}

	for _, alert := range alerts {
		if err := alert.Validate(); err != nil {
			t.Errorf("validation failed for %s: %v", alert.Metadata.Name, err)
		}
	}

	// Test NewAlertCondition
	ac := NewAlertCondition("test-condition", SeverityCritical, 14.0, "1h", "5m")
	if err := ac.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}

	// Test NewAlertNotificationTarget
	target := NewAlertNotificationTarget("slack", "https://hooks.slack.com/xxx", "Slack alerts")
	if err := target.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}

	// Test NewAlertPolicyBuilder
	policy := NewAlertPolicyBuilder("test-policy").
		AddConditionRef("test-condition").
		AddNotificationTargetRef("slack").
		Build()
	if err := policy.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}
