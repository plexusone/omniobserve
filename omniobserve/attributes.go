package omniobserve

import (
	"github.com/plexusone/omniobserve/observops"
)

// Attribute helper functions for creating KeyValue pairs.
// These provide typed wrappers around observops.Attribute for convenience.

// String creates a string attribute.
func String(key, value string) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Int creates an int attribute.
func Int(key string, value int) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Int64 creates an int64 attribute.
func Int64(key string, value int64) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Float64 creates a float64 attribute.
func Float64(key string, value float64) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Bool creates a bool attribute.
func Bool(key string, value bool) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Strings creates a string slice attribute.
func Strings(key string, value []string) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Ints creates an int slice attribute.
func Ints(key string, value []int) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Int64s creates an int64 slice attribute.
func Int64s(key string, value []int64) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Float64s creates a float64 slice attribute.
func Float64s(key string, value []float64) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Bools creates a bool slice attribute.
func Bools(key string, value []bool) observops.KeyValue {
	return observops.Attribute(key, value)
}

// Attr is an alias for observops.Attribute for convenience.
func Attr(key string, value any) observops.KeyValue {
	return observops.Attribute(key, value)
}
