package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON outputs the given value as JSON to stdout.
// JSON encoding errors are written to stderr.
func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
	}
}
