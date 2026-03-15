// Command genspecs generates JSON Schema files from Go struct definitions.
//
// Usage:
//
//	go run ./cmd/genspecs -output ./specs/schema
//
// This generates the following schema files:
//   - red.schema.json
//   - use.schema.json
//   - golden.schema.json
//   - openslo.schema.json
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/omniobserve/specs/schema"
)

func main() {
	outputDir := flag.String("output", "./specs/schema", "Output directory for schema files")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	generators := schema.All()

	for filename, generator := range generators {
		data, err := generator()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to generate %s: %v\n", filename, err)
			os.Exit(1)
		}

		path := filepath.Join(*outputDir, filename)
		if err := os.WriteFile(path, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", path, err)
			os.Exit(1)
		}

		fmt.Printf("Generated %s\n", path)
	}

	fmt.Println("Done.")
}
