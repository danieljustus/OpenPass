package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"

	configpkg "github.com/danieljustus/OpenPass/internal/config"
)

func main() {
	r := &jsonschema.Reflector{
		DoNotReference: true,
	}

	schema := r.Reflect(&configpkg.Config{})
	schema.ID = "https://openpass.dev/config.schema.json"
	schema.Version = "https://json-schema.org/draft/2020-12/schema"
	schema.Description = "OpenPass configuration schema"

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}
