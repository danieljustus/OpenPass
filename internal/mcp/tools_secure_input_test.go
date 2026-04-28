package mcp

import (
	"os"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
)

func TestSecureInputToolAvailabilityInRegistry(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", "")

	originalOpenSecureTTY := openSecureTTY
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	openSecureTTY = func() (secureInputDevice, error) {
		return nil, os.ErrNotExist
	}
	tools := toolsListPayload(srv)
	if toolNamesContain(tools, "secure_input") {
		t.Fatal("secure_input should be hidden when no TTY is available")
	}

	openSecureTTY = func() (secureInputDevice, error) {
		return &mockSecureInputDevice{}, nil
	}

	tools = toolsListPayload(srv)
	if !toolNamesContain(tools, "secure_input") {
		t.Fatal("secure_input should be listed when stdio and TTY are available")
	}
}

func toolNamesContain(tools []map[string]any, target string) bool {
	for _, tool := range tools {
		if name, _ := tool["name"].(string); name == target {
			return true
		}
	}
	return false
}
