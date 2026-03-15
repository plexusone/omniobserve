// Package specs provides observability standard definitions.
//
// This package contains Go-first definitions for observability standards:
//   - red: RED metrics (Rate, Errors, Duration)
//   - use: USE metrics (Utilization, Saturation, Errors)
//   - golden: 4 Golden Signals mapping
//   - openslo: OpenSLO SLI/SLO definitions
//   - recorder: Integration with observops.Provider
//
// # Architecture
//
// Applications emit metrics using the specs types, which are then recorded
// through the observops.Provider abstraction. This keeps applications
// vendor-agnostic while ensuring consistent metric naming and semantics.
//
//	┌─────────────────────────────────────────┐
//	│          Application Code               │
//	│   recorder.RecordRED(ctx, obs)          │
//	└─────────────────────────────────────────┘
//	                    │
//	                    ▼
//	┌─────────────────────────────────────────┐
//	│         specs/recorder                  │
//	│   Maps RED/USE to OTel metrics          │
//	└─────────────────────────────────────────┘
//	                    │
//	                    ▼
//	┌─────────────────────────────────────────┐
//	│         observops.Provider              │
//	│   OTLP, Datadog, New Relic, etc.        │
//	└─────────────────────────────────────────┘
//
// # Metric Models
//
// RED (Request-oriented):
//   - Rate: Request throughput
//   - Errors: Failed requests
//   - Duration: Request latency
//
// USE (Resource-oriented):
//   - Utilization: % time resource is busy
//   - Saturation: Queue depth / backlog
//   - Errors: Resource error count
//
// 4 Golden Signals (Service health):
//   - Latency: Time to service a request (from RED.Duration)
//   - Traffic: Demand on the system (from RED.Rate)
//   - Errors: Rate of failed requests (from RED.Errors)
//   - Saturation: How "full" the system is (from USE)
//
// # Usage
//
//	import (
//	    "github.com/plexusone/omniobserve/observops"
//	    _ "github.com/plexusone/omniobserve/observops/otlp"
//	    "github.com/plexusone/omniobserve/specs/recorder"
//	    "github.com/plexusone/omniobserve/specs/red"
//	)
//
//	func main() {
//	    provider, _ := observops.Open("otlp",
//	        observops.WithEndpoint("localhost:4317"),
//	        observops.WithServiceName("my-service"),
//	    )
//	    defer provider.Shutdown(context.Background())
//
//	    rec := recorder.New(provider, "my-service")
//
//	    // Record a request
//	    start := time.Now()
//	    err := handleRequest()
//	    rec.RecordRED(ctx, "http.server.request", red.Observation{
//	        Duration: time.Since(start),
//	        Error:    err,
//	        Attributes: map[string]string{
//	            "http.method": "POST",
//	            "http.route":  "/api/users",
//	        },
//	    })
//	}
package specs
