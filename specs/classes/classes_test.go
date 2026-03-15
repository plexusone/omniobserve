package classes

import (
	"testing"
)

func TestMatchEndpoint(t *testing.T) {
	tests := []struct {
		pattern  string
		endpoint string
		want     bool
	}{
		// Exact match
		{"/login", "/login", true},
		{"/login", "/logout", false},
		{"/checkout", "/checkout", true},

		// Single wildcard
		{"/profile/*", "/profile/settings", true},
		{"/profile/*", "/profile/avatar", true},
		{"/profile/*", "/profile/settings/advanced", false}, // Too deep
		{"/profile/*", "/profile", false},                   // No suffix
		{"/api/v1/*", "/api/v1/users", true},
		{"/api/v1/*", "/api/v2/users", false},

		// Double wildcard (recursive)
		{"/api/**", "/api/v1/users", true},
		{"/api/**", "/api/v1/users/123", true},
		{"/api/**", "/api", false},
		{"/admin/**", "/admin/users/edit/123", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"->"+tt.endpoint, func(t *testing.T) {
			got := matchEndpoint(tt.pattern, tt.endpoint)
			if got != tt.want {
				t.Errorf("matchEndpoint(%q, %q) = %v, want %v", tt.pattern, tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestClassifyEndpoint(t *testing.T) {
	spec := NewServiceSpec("acme-web", "web-platform").
		AddClass(NewCriticalClass("/login", "/checkout", "/payments")).
		AddClass(NewNormalClass("/profile/*", "/search")).
		AddClass(NewBestEffortClass("/recommendations", "/analytics/**"))

	tests := []struct {
		endpoint  string
		wantClass ClassLevel
	}{
		{"/login", ClassCritical},
		{"/checkout", ClassCritical},
		{"/payments", ClassCritical},
		{"/profile/settings", ClassNormal},
		{"/profile/avatar", ClassNormal},
		{"/search", ClassNormal},
		{"/recommendations", ClassBestEffort},
		{"/analytics/events", ClassBestEffort},
		{"/analytics/reports/daily", ClassBestEffort},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			c := spec.ClassifyEndpoint(tt.endpoint)
			if c == nil {
				t.Fatalf("ClassifyEndpoint(%q) returned nil", tt.endpoint)
			}
			if c.Name != tt.wantClass {
				t.Errorf("ClassifyEndpoint(%q) = %v, want %v", tt.endpoint, c.Name, tt.wantClass)
			}
		})
	}
}

func TestClassifyEndpoint_DefaultClass(t *testing.T) {
	// Create spec with a normal class already present
	spec := NewServiceSpec("acme-web", "web-platform").
		AddClass(NewCriticalClass("/login")).
		AddClass(NewNormalClass("/search")).
		WithDefaultClass(ClassNormal)

	// Should return the normal class as default for unmatched endpoint
	// (since normal class exists in the classes list)
	c := spec.ClassifyEndpoint("/unknown")
	if c == nil {
		t.Errorf("Expected default class for unmatched endpoint, got nil")
	} else if c.Name != ClassNormal {
		t.Errorf("Expected default class 'normal' for unmatched endpoint, got %v", c.Name)
	}

	// Test without default class
	specNoDefault := NewServiceSpec("acme-web", "web-platform").
		AddClass(NewCriticalClass("/login"))

	c = specNoDefault.ClassifyEndpoint("/unknown")
	if c != nil {
		t.Errorf("Expected nil for unmatched endpoint with no default class, got %v", c.Name)
	}
}

func TestGenerateSLOs(t *testing.T) {
	spec := NewServiceSpec("acme-web", "web-platform").
		AddClass(NewCriticalClass("/login", "/checkout")).
		AddClass(NewNormalClass("/search"))

	slos := spec.GenerateSLOs()

	// Should generate SLOs for each specific endpoint
	if len(slos) < 3 {
		t.Errorf("Expected at least 3 SLOs, got %d", len(slos))
	}

	// Check SLO names
	names := make(map[string]bool)
	for _, slo := range slos {
		names[slo.Metadata.Name] = true
	}

	expectedNames := []string{
		"acme-web-login-availability",
		"acme-web-checkout-availability",
		"acme-web-search-availability",
	}

	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected SLO with name %q", name)
		}
	}
}

func TestGenerateSLOs_WithWildcards(t *testing.T) {
	spec := NewServiceSpec("acme-web", "web-platform").
		AddClass(Class{
			Name:        ClassNormal,
			SLOTemplate: "normal-endpoint",
			Endpoints:   []string{"/api/*", "/profile/**"},
			ThresholdOverrides: ThresholdOverrides{
				Availability: 99.5,
				TimeWindow:   "30d",
			},
		})

	slos := spec.GenerateSLOs()

	// Should generate class-level SLO for wildcards
	if len(slos) != 1 {
		t.Errorf("Expected 1 class-level SLO for wildcards, got %d", len(slos))
	}

	slo := slos[0]
	// Check that labels exist (Labels is a map of label key to []string values)
	if len(slo.Metadata.Labels) == 0 {
		t.Error("Expected labels to be set")
	}
}

func TestNewCriticalClass(t *testing.T) {
	c := NewCriticalClass("/login", "/checkout")

	if c.Name != ClassCritical {
		t.Errorf("Expected ClassCritical, got %v", c.Name)
	}
	if len(c.Endpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(c.Endpoints))
	}
	if c.ThresholdOverrides.Availability != 99.9 {
		t.Errorf("Expected 99.9 availability, got %v", c.ThresholdOverrides.Availability)
	}
}

func TestStandardTemplates(t *testing.T) {
	templates := StandardTemplates()

	if len(templates) < 3 {
		t.Errorf("Expected at least 3 standard templates, got %d", len(templates))
	}

	names := make(map[string]bool)
	for _, tmpl := range templates {
		names[tmpl.Name] = true
	}

	expectedTemplates := []string{
		"critical-endpoint",
		"normal-endpoint",
		"best-effort-endpoint",
	}

	for _, name := range expectedTemplates {
		if !names[name] {
			t.Errorf("Expected template %q", name)
		}
	}
}

func TestServiceSpec_MetricsModel(t *testing.T) {
	spec := NewServiceSpec("test-svc", "platform").
		WithMetricsModel("RED", "USE")

	if len(spec.MetricsModel) != 2 {
		t.Errorf("Expected 2 metrics models, got %d", len(spec.MetricsModel))
	}
	if spec.MetricsModel[0] != "RED" || spec.MetricsModel[1] != "USE" {
		t.Errorf("Unexpected metrics models: %v", spec.MetricsModel)
	}
}
