// Package recorder provides integration between specs and observops.Provider.
//
// The Recorder simplifies emitting RED and USE metrics by wrapping the observops
// metric instruments and providing a high-level API for common observability patterns.
package recorder

import (
	"context"
	"sync"

	"github.com/plexusone/omniobserve/observops"
	"github.com/plexusone/omniobserve/specs/red"
	"github.com/plexusone/omniobserve/specs/use"
)

// Recorder records RED and USE metrics using an observops.Provider.
type Recorder struct {
	provider    observops.Provider
	serviceName string

	// Cached metric instruments
	mu                 sync.RWMutex
	requestCounters    map[string]observops.Counter
	errorCounters      map[string]observops.Counter
	durationHistograms map[string]observops.Histogram
	utilizationGauges  map[string]observops.Gauge
	saturationGauges   map[string]observops.Gauge
	resourceErrors     map[string]observops.Counter
}

// New creates a Recorder with the given provider.
func New(provider observops.Provider, serviceName string) *Recorder {
	return &Recorder{
		provider:           provider,
		serviceName:        serviceName,
		requestCounters:    make(map[string]observops.Counter),
		errorCounters:      make(map[string]observops.Counter),
		durationHistograms: make(map[string]observops.Histogram),
		utilizationGauges:  make(map[string]observops.Gauge),
		saturationGauges:   make(map[string]observops.Gauge),
		resourceErrors:     make(map[string]observops.Counter),
	}
}

// RecordRED records a RED observation.
func (r *Recorder) RecordRED(ctx context.Context, name string, obs red.Observation) error {
	attrs := toKeyValues(obs.Attributes)
	opts := observops.WithAttributes(attrs...)

	// Get or create counter for request count
	counter, err := r.getOrCreateCounter(name+".count", "Total requests", "{requests}")
	if err != nil {
		return err
	}
	counter.Add(ctx, 1, opts)

	// Get or create histogram for duration
	histogram, err := r.getOrCreateHistogram(name+".duration", "Request latency", "ms")
	if err != nil {
		return err
	}
	histogram.Record(ctx, float64(obs.Duration.Milliseconds()), opts)

	// Record errors if present
	if obs.IsError() {
		errorCounter, err := r.getOrCreateCounter(name+".errors", "Failed requests", "{requests}")
		if err != nil {
			return err
		}
		errorCounter.Add(ctx, 1, opts)
	}

	return nil
}

// RecordUSE records a USE observation.
func (r *Recorder) RecordUSE(ctx context.Context, name string, obs use.Observation) error {
	attrs := toKeyValues(obs.Attributes)
	opts := observops.WithAttributes(attrs...)

	// Record utilization if present
	if obs.Utilization > 0 {
		gauge, err := r.getOrCreateGauge(name+".utilization", "Resource utilization", "1")
		if err != nil {
			return err
		}
		gauge.Record(ctx, obs.Utilization, opts)
	}

	// Record saturation if present
	if obs.Saturation > 0 {
		gauge, err := r.getOrCreateSaturationGauge(name+".saturation", "Resource saturation", "{items}")
		if err != nil {
			return err
		}
		gauge.Record(ctx, obs.Saturation, opts)
	}

	// Record errors if present
	if obs.ErrorCount > 0 {
		counter, err := r.getOrCreateResourceErrorCounter(name+".errors", "Resource errors", "{errors}")
		if err != nil {
			return err
		}
		counter.Add(ctx, float64(obs.ErrorCount), opts)
	}

	return nil
}

// RecordHTTPRequest is a convenience method for recording HTTP request metrics.
func (r *Recorder) RecordHTTPRequest(ctx context.Context, obs red.Observation) error {
	return r.RecordRED(ctx, "http.server.request", obs)
}

// RecordGRPCRequest is a convenience method for recording gRPC request metrics.
func (r *Recorder) RecordGRPCRequest(ctx context.Context, obs red.Observation) error {
	return r.RecordRED(ctx, "rpc.server.request", obs)
}

// RecordCPU records CPU utilization metrics.
func (r *Recorder) RecordCPU(ctx context.Context, utilization float64) error {
	return r.RecordUSE(ctx, "system.cpu", use.Observation{
		Resource:    use.ResourceCPU,
		Utilization: utilization,
		Attributes: map[string]string{
			"service.name": r.serviceName,
		},
	})
}

// RecordMemory records memory utilization metrics.
func (r *Recorder) RecordMemory(ctx context.Context, utilization float64, usage float64) error {
	return r.RecordUSE(ctx, "system.memory", use.Observation{
		Resource:    use.ResourceMemory,
		Utilization: utilization,
		Saturation:  usage,
		Attributes: map[string]string{
			"service.name": r.serviceName,
		},
	})
}

// RecordQueue records queue metrics.
func (r *Recorder) RecordQueue(ctx context.Context, queueName string, utilization, depth float64) error {
	return r.RecordUSE(ctx, "queue."+queueName, use.Observation{
		Resource:    use.ResourceQueue,
		Utilization: utilization,
		Saturation:  depth,
		Attributes: map[string]string{
			"service.name": r.serviceName,
			"queue.name":   queueName,
		},
	})
}

// RecordConnectionPool records connection pool metrics.
func (r *Recorder) RecordConnectionPool(ctx context.Context, poolName string, utilization float64, pending int) error {
	return r.RecordUSE(ctx, "pool."+poolName, use.Observation{
		Resource:    use.ResourcePool,
		Utilization: utilization,
		Saturation:  float64(pending),
		Attributes: map[string]string{
			"service.name": r.serviceName,
			"pool.name":    poolName,
		},
	})
}

// getOrCreateCounter returns an existing counter or creates a new one.
func (r *Recorder) getOrCreateCounter(name, desc, unit string) (observops.Counter, error) {
	r.mu.RLock()
	if c, ok := r.requestCounters[name]; ok {
		r.mu.RUnlock()
		return c, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if c, ok := r.requestCounters[name]; ok {
		return c, nil
	}

	counter, err := r.provider.Meter().Counter(name,
		observops.WithDescription(desc),
		observops.WithUnit(unit))
	if err != nil {
		return nil, err
	}
	r.requestCounters[name] = counter
	return counter, nil
}

// getOrCreateHistogram returns an existing histogram or creates a new one.
func (r *Recorder) getOrCreateHistogram(name, desc, unit string) (observops.Histogram, error) {
	r.mu.RLock()
	if h, ok := r.durationHistograms[name]; ok {
		r.mu.RUnlock()
		return h, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if h, ok := r.durationHistograms[name]; ok {
		return h, nil
	}

	histogram, err := r.provider.Meter().Histogram(name,
		observops.WithDescription(desc),
		observops.WithUnit(unit))
	if err != nil {
		return nil, err
	}
	r.durationHistograms[name] = histogram
	return histogram, nil
}

// getOrCreateGauge returns an existing gauge or creates a new one.
func (r *Recorder) getOrCreateGauge(name, desc, unit string) (observops.Gauge, error) {
	r.mu.RLock()
	if g, ok := r.utilizationGauges[name]; ok {
		r.mu.RUnlock()
		return g, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if g, ok := r.utilizationGauges[name]; ok {
		return g, nil
	}

	gauge, err := r.provider.Meter().Gauge(name,
		observops.WithDescription(desc),
		observops.WithUnit(unit))
	if err != nil {
		return nil, err
	}
	r.utilizationGauges[name] = gauge
	return gauge, nil
}

// getOrCreateSaturationGauge returns an existing saturation gauge or creates a new one.
func (r *Recorder) getOrCreateSaturationGauge(name, desc, unit string) (observops.Gauge, error) {
	r.mu.RLock()
	if g, ok := r.saturationGauges[name]; ok {
		r.mu.RUnlock()
		return g, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if g, ok := r.saturationGauges[name]; ok {
		return g, nil
	}

	gauge, err := r.provider.Meter().Gauge(name,
		observops.WithDescription(desc),
		observops.WithUnit(unit))
	if err != nil {
		return nil, err
	}
	r.saturationGauges[name] = gauge
	return gauge, nil
}

// getOrCreateResourceErrorCounter returns an existing error counter or creates a new one.
func (r *Recorder) getOrCreateResourceErrorCounter(name, desc, unit string) (observops.Counter, error) {
	r.mu.RLock()
	if c, ok := r.resourceErrors[name]; ok {
		r.mu.RUnlock()
		return c, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.resourceErrors[name]; ok {
		return c, nil
	}

	counter, err := r.provider.Meter().Counter(name,
		observops.WithDescription(desc),
		observops.WithUnit(unit))
	if err != nil {
		return nil, err
	}
	r.resourceErrors[name] = counter
	return counter, nil
}

func toKeyValues(attrs map[string]string) []observops.KeyValue {
	kvs := make([]observops.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		kvs = append(kvs, observops.Attribute(k, v))
	}
	return kvs
}
