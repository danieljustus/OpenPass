package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
)

func TestToolsList_FiltersGetEntryValue_WhenExposeValueToolsFalse(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:             "test",
		AllowedPaths:     []string{"*"},
		CanReadValues:    true,
		ExposeValueTools: false,
		ApprovalMode:     "prompt",
	}, "http", "")

	tools := toolsListPayload(srv)
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool["name"].(string)] = true
	}

	if names["get_entry_value"] {
		t.Error("get_entry_value should NOT be in tools/list when ExposeValueTools=false")
	}
	if !names["get_entry"] {
		t.Error("get_entry should still be in tools/list when ExposeValueTools=false")
	}
	if !names["list_entries"] {
		t.Error("list_entries should still be in tools/list")
	}
}

func TestToolsList_ShowsGetEntryValue_WhenExposeValueToolsTrue(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:             "test",
		AllowedPaths:     []string{"*"},
		CanReadValues:    true,
		ExposeValueTools: true,
		ApprovalMode:     "prompt",
	}, "http", "")

	tools := toolsListPayload(srv)
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool["name"].(string)] = true
	}

	if !names["get_entry_value"] {
		t.Error("get_entry_value should be in tools/list when ExposeValueTools=true")
	}
}

func TestExecuteTool_BlocksGetEntryValue_WhenExposeValueToolsFalse(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:             "test",
		AllowedPaths:     []string{"*"},
		ExposeValueTools: false,
		ApprovalMode:     "none",
	}, "stdio", "")

	args := json.RawMessage(`{"path": "test"}`)
	_, err := srv.executeTool(context.Background(), "get_entry_value", args)
	if err == nil {
		t.Fatal("executeTool() expected error for get_entry_value when ExposeValueTools=false, got nil")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("executeTool() error = %v, want 'unknown tool'", err)
	}
}

func TestAvailableToolDefinitions_FiltersGetEntryValue_WhenExposeValueToolsFalse(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:             "test",
		AllowedPaths:     []string{"*"},
		ExposeValueTools: false,
		ApprovalMode:     "none",
	}, "http", "")

	defs := availableToolDefinitions(srv)
	names := make(map[string]bool, len(defs))
	for _, def := range defs {
		names[def.Name] = true
	}

	if names["get_entry_value"] {
		t.Error("get_entry_value should NOT be available when ExposeValueTools=false")
	}
	if !names["get_entry"] {
		t.Error("get_entry should still be available when ExposeValueTools=false")
	}
}
