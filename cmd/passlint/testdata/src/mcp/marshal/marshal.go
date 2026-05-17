package marshal

import (
	"encoding/json"
)

type Entry struct {
	Data map[string]any
}

func safeMarshal() {
	entry := &Entry{Data: map[string]any{"key": "value"}}
	_, _ = json.Marshal(entry) // want "json.Marshal in MCP code"
}

func unsafeMarshalInHandler() {
	entry := &Entry{Data: map[string]any{"key": "value"}}
	_, _ = json.Marshal(entry) // want "json.Marshal in MCP code"
}
