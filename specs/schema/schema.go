// Package schema provides JSON Schema generation for observability specs.
//
// This package generates JSON Schemas from the Go struct definitions,
// following the Go-first approach where Go types are the source of truth.
package schema

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
	"github.com/plexusone/omniobserve/specs/classes"
	"github.com/plexusone/omniobserve/specs/golden"
	"github.com/plexusone/omniobserve/specs/openslo"
	"github.com/plexusone/omniobserve/specs/red"
	"github.com/plexusone/omniobserve/specs/use"
)

// REDDefinition returns the JSON Schema for red.Definition.
func REDDefinition() *jsonschema.Schema {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}
	return r.Reflect(&red.Definition{})
}

// USEDefinition returns the JSON Schema for use.Definition.
func USEDefinition() *jsonschema.Schema {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}
	return r.Reflect(&use.Definition{})
}

// GoldenDefinition returns the JSON Schema for golden.Definition.
func GoldenDefinition() *jsonschema.Schema {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}
	return r.Reflect(&golden.Definition{})
}

// OpenSLOSLO returns the JSON Schema for openslo.SLO.
func OpenSLOSLO() *jsonschema.Schema {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}
	return r.Reflect(&openslo.SLO{})
}

// OpenSLOService returns the JSON Schema for openslo.Service.
func OpenSLOService() *jsonschema.Schema {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}
	return r.Reflect(&openslo.Service{})
}

// GenerateREDSchema generates JSON Schema for RED definitions as JSON bytes.
func GenerateREDSchema() ([]byte, error) {
	schema := REDDefinition()
	return json.MarshalIndent(schema, "", "  ")
}

// GenerateUSESchema generates JSON Schema for USE definitions as JSON bytes.
func GenerateUSESchema() ([]byte, error) {
	schema := USEDefinition()
	return json.MarshalIndent(schema, "", "  ")
}

// GenerateGoldenSchema generates JSON Schema for Golden Signals as JSON bytes.
func GenerateGoldenSchema() ([]byte, error) {
	schema := GoldenDefinition()
	return json.MarshalIndent(schema, "", "  ")
}

// GenerateOpenSLOSchema generates JSON Schema for OpenSLO SLO as JSON bytes.
func GenerateOpenSLOSchema() ([]byte, error) {
	schema := OpenSLOSLO()
	return json.MarshalIndent(schema, "", "  ")
}

// ServiceSpec returns the JSON Schema for classes.ServiceSpec.
func ServiceSpec() *jsonschema.Schema {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}
	return r.Reflect(&classes.ServiceSpec{})
}

// SLOTemplate returns the JSON Schema for classes.SLOTemplate.
func SLOTemplate() *jsonschema.Schema {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}
	return r.Reflect(&classes.SLOTemplate{})
}

// GenerateClassesSchema generates JSON Schema for service class specs as JSON bytes.
func GenerateClassesSchema() ([]byte, error) {
	schema := ServiceSpec()
	return json.MarshalIndent(schema, "", "  ")
}

// GenerateSLOTemplateSchema generates JSON Schema for SLO templates as JSON bytes.
func GenerateSLOTemplateSchema() ([]byte, error) {
	schema := SLOTemplate()
	return json.MarshalIndent(schema, "", "  ")
}

// All returns all schema generators.
func All() map[string]func() ([]byte, error) {
	return map[string]func() ([]byte, error){
		"red.schema.json":          GenerateREDSchema,
		"use.schema.json":          GenerateUSESchema,
		"golden.schema.json":       GenerateGoldenSchema,
		"openslo.schema.json":      GenerateOpenSLOSchema,
		"classes.schema.json":      GenerateClassesSchema,
		"slo-template.schema.json": GenerateSLOTemplateSchema,
	}
}
