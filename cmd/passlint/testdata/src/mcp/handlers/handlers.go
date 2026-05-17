package handlers

import (
	"context"
	"fmt"
)

type CallToolRequest struct {
	Arguments map[string]any
}

type CallToolResult struct {
	Text string
}

type Server struct{}

func (s *Server) checkScope(path string) bool {
	return true
}

func (s *Server) handleGetValue(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	path, _ := req.Arguments["path"].(string)
	if !s.checkScope(path) {
		return nil, fmt.Errorf("access denied")
	}
	return &CallToolResult{Text: "ok"}, nil
}

func (s *Server) handleBadGetValue(ctx context.Context, req CallToolRequest) (*CallToolResult, error) { // want "MCP handler handleBadGetValue does not call s.checkScope"
	path, _ := req.Arguments["path"].(string)
	_ = path
	return &CallToolResult{Text: "ok"}, nil
}

func (s *Server) helperFunc(x int) int {
	return x + 1
}
