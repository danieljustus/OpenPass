package mcp

import (
	"context"
	"encoding/json"
)

func (s *Server) handleHealth(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_, _ = ctx, req
	result := map[string]any{
		"status":    "healthy",
		"server":    defaultServerName,
		"version":   defaultServerVersion,
		"transport": s.transport,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return NewToolResultText(string(resultJSON)), nil
}
