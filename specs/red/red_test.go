package red

import (
	"errors"
	"testing"
	"time"
)

func TestObservation_IsError(t *testing.T) {
	tests := []struct {
		name string
		obs  Observation
		want bool
	}{
		{
			name: "no error",
			obs:  Observation{Duration: 100 * time.Millisecond},
			want: false,
		},
		{
			name: "with error",
			obs:  Observation{Duration: 100 * time.Millisecond, Error: errors.New("test")},
			want: true,
		},
		{
			name: "status 200",
			obs:  Observation{Duration: 100 * time.Millisecond, StatusCode: 200},
			want: false,
		},
		{
			name: "status 400",
			obs:  Observation{Duration: 100 * time.Millisecond, StatusCode: 400},
			want: false,
		},
		{
			name: "status 500",
			obs:  Observation{Duration: 100 * time.Millisecond, StatusCode: 500},
			want: true,
		},
		{
			name: "status 503",
			obs:  Observation{Duration: 100 * time.Millisecond, StatusCode: 503},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.obs.IsError(); got != tt.want {
				t.Errorf("Observation.IsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultBuckets(t *testing.T) {
	buckets := DefaultBuckets()
	if len(buckets) == 0 {
		t.Error("DefaultBuckets() returned empty slice")
	}

	// Verify buckets are in ascending order
	for i := 1; i < len(buckets); i++ {
		if buckets[i] <= buckets[i-1] {
			t.Errorf("Buckets not in ascending order: %v <= %v", buckets[i], buckets[i-1])
		}
	}
}

func TestHTTPServerDefinition(t *testing.T) {
	def := HTTPServerDefinition()

	if def.Name != "http.server.request" {
		t.Errorf("unexpected name: %s", def.Name)
	}

	if def.Rate.Metric != "http.server.request.count" {
		t.Errorf("unexpected rate metric: %s", def.Rate.Metric)
	}

	if def.Errors.Metric != "http.server.request.errors" {
		t.Errorf("unexpected errors metric: %s", def.Errors.Metric)
	}

	if def.Duration.Metric != "http.server.request.duration" {
		t.Errorf("unexpected duration metric: %s", def.Duration.Metric)
	}

	if !def.Errors.SLICandidate {
		t.Error("errors should be SLI candidate")
	}

	if !def.Duration.SLICandidate {
		t.Error("duration should be SLI candidate")
	}
}

func TestGRPCServerDefinition(t *testing.T) {
	def := GRPCServerDefinition()

	if def.Name != "rpc.server.request" {
		t.Errorf("unexpected name: %s", def.Name)
	}

	if def.Errors.Filter != "rpc.grpc.status_code != 0" {
		t.Errorf("unexpected error filter: %s", def.Errors.Filter)
	}
}
