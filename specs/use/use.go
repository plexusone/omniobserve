// Package use provides types for USE (Utilization, Saturation, Errors) metrics.
//
// USE metrics are resource-oriented metrics defined by Brendan Gregg:
//   - Utilization: Percentage of time the resource is busy
//   - Saturation: Degree to which the resource has extra work it can't service (queue depth)
//   - Errors: Count of error events
//
// These map to the Saturation Golden Signal.
package use

// MetricType identifies a USE metric category.
type MetricType string

const (
	// MetricUtilization represents resource utilization.
	MetricUtilization MetricType = "utilization"

	// MetricSaturation represents resource saturation.
	MetricSaturation MetricType = "saturation"

	// MetricErrors represents resource errors.
	MetricErrors MetricType = "errors"
)

// ResourceType identifies the type of resource being measured.
type ResourceType string

const (
	// ResourceCPU represents CPU resources.
	ResourceCPU ResourceType = "cpu"

	// ResourceMemory represents memory resources.
	ResourceMemory ResourceType = "memory"

	// ResourceDisk represents disk resources.
	ResourceDisk ResourceType = "disk"

	// ResourceNetwork represents network resources.
	ResourceNetwork ResourceType = "network"

	// ResourceQueue represents queue resources.
	ResourceQueue ResourceType = "queue"

	// ResourcePool represents connection pool resources.
	ResourcePool ResourceType = "pool"

	// ResourceThread represents thread pool resources.
	ResourceThread ResourceType = "thread"

	// ResourceGoroutine represents goroutine resources.
	ResourceGoroutine ResourceType = "goroutine"
)

// Definition describes USE metrics for a resource.
type Definition struct {
	// Resource is the resource being measured.
	Resource ResourceType `json:"resource"`

	// Name is the metric name prefix.
	Name string `json:"name"`

	// Description explains what this measures.
	Description string `json:"description,omitempty"`

	// Utilization metric configuration.
	Utilization *UtilizationConfig `json:"utilization,omitempty"`

	// Saturation metric configuration.
	Saturation *SaturationConfig `json:"saturation,omitempty"`

	// Errors metric configuration.
	Errors *ErrorsConfig `json:"errors,omitempty"`

	// Attributes are common attributes.
	Attributes []Attribute `json:"attributes,omitempty"`
}

// UtilizationConfig configures utilization metrics.
type UtilizationConfig struct {
	// Metric is the OTel metric name.
	Metric string `json:"metric"`

	// Unit is typically "1" for ratio or "%" for percentage.
	Unit string `json:"unit,omitempty"`

	// GoldenSignal maps to "saturation".
	GoldenSignal string `json:"golden_signal,omitempty"`
}

// SaturationConfig configures saturation metrics.
type SaturationConfig struct {
	// Metric is the OTel metric name.
	Metric string `json:"metric"`

	// Unit is typically count or items.
	Unit string `json:"unit,omitempty"`

	// GoldenSignal maps to "saturation".
	GoldenSignal string `json:"golden_signal,omitempty"`
}

// ErrorsConfig configures resource error metrics.
type ErrorsConfig struct {
	// Metric is the OTel metric name.
	Metric string `json:"metric"`

	// Unit is typically count.
	Unit string `json:"unit,omitempty"`
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

// Observation captures a single USE observation.
type Observation struct {
	// Resource identifies the resource type.
	Resource ResourceType

	// Utilization as a ratio 0.0-1.0.
	Utilization float64

	// Saturation as queue depth or count.
	Saturation float64

	// ErrorCount number of errors.
	ErrorCount int64

	// Attributes for this observation.
	Attributes map[string]string
}

// CPUDefinition returns a standard USE definition for CPU.
func CPUDefinition() Definition {
	return Definition{
		Resource:    ResourceCPU,
		Name:        "system.cpu",
		Description: "CPU resource metrics",
		Utilization: &UtilizationConfig{
			Metric:       "system.cpu.utilization",
			Unit:         "1",
			GoldenSignal: "saturation",
		},
		Saturation: &SaturationConfig{
			Metric:       "system.cpu.load",
			Unit:         "1",
			GoldenSignal: "saturation",
		},
		Attributes: []Attribute{
			{Key: "service.name", Required: true},
			{Key: "host.name"},
		},
	}
}

// MemoryDefinition returns a standard USE definition for memory.
func MemoryDefinition() Definition {
	return Definition{
		Resource:    ResourceMemory,
		Name:        "system.memory",
		Description: "Memory resource metrics",
		Utilization: &UtilizationConfig{
			Metric:       "system.memory.utilization",
			Unit:         "1",
			GoldenSignal: "saturation",
		},
		Saturation: &SaturationConfig{
			Metric:       "system.memory.usage",
			Unit:         "By",
			GoldenSignal: "saturation",
		},
		Attributes: []Attribute{
			{Key: "service.name", Required: true},
			{Key: "host.name"},
		},
	}
}

// QueueDefinition returns a USE definition for a named queue.
func QueueDefinition(queueName string) Definition {
	return Definition{
		Resource:    ResourceQueue,
		Name:        "queue." + queueName,
		Description: "Queue resource metrics for " + queueName,
		Utilization: &UtilizationConfig{
			Metric:       "queue." + queueName + ".utilization",
			Unit:         "1",
			GoldenSignal: "saturation",
		},
		Saturation: &SaturationConfig{
			Metric:       "queue." + queueName + ".depth",
			Unit:         "{items}",
			GoldenSignal: "saturation",
		},
		Errors: &ErrorsConfig{
			Metric: "queue." + queueName + ".errors",
			Unit:   "{errors}",
		},
		Attributes: []Attribute{
			{Key: "service.name", Required: true},
			{Key: "queue.name", Required: true},
		},
	}
}

// ConnectionPoolDefinition returns a USE definition for a connection pool.
func ConnectionPoolDefinition(poolName string) Definition {
	return Definition{
		Resource:    ResourcePool,
		Name:        "pool." + poolName,
		Description: "Connection pool metrics for " + poolName,
		Utilization: &UtilizationConfig{
			Metric:       "pool." + poolName + ".utilization",
			Unit:         "1",
			GoldenSignal: "saturation",
		},
		Saturation: &SaturationConfig{
			Metric:       "pool." + poolName + ".pending",
			Unit:         "{connections}",
			GoldenSignal: "saturation",
		},
		Errors: &ErrorsConfig{
			Metric: "pool." + poolName + ".errors",
			Unit:   "{errors}",
		},
		Attributes: []Attribute{
			{Key: "service.name", Required: true},
			{Key: "pool.name", Required: true},
		},
	}
}

// GoroutineDefinition returns a USE definition for goroutines.
func GoroutineDefinition() Definition {
	return Definition{
		Resource:    ResourceGoroutine,
		Name:        "runtime.goroutine",
		Description: "Go runtime goroutine metrics",
		Saturation: &SaturationConfig{
			Metric:       "runtime.goroutine.count",
			Unit:         "{goroutines}",
			GoldenSignal: "saturation",
		},
		Attributes: []Attribute{
			{Key: "service.name", Required: true},
		},
	}
}
