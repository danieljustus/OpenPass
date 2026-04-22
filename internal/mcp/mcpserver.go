package mcp

import "context"

// mcpServer is a minimal compatibility shim that replaces the upstream
// mark3labs/mcp-go server package. OpenPass implements its own MCP
// protocol handling in protocol.go and transport.go; this type is
// preserved only to satisfy existing Build() signatures.
type mcpServer struct {
	Name    string
	Version string
	tools   []Tool
}

// serverOption configures an mcpServer
type serverOption func(*mcpServer)

// NewMCPServer creates a new mcpServer instance
//
//nolint:revive
func NewMCPServer(name, version string, opts ...serverOption) *mcpServer {
	s := &mcpServer{Name: name, Version: version}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

// WithToolCapabilities is a no-op placeholder
//
//nolint:revive
func WithToolCapabilities(_ bool) serverOption {
	return func(*mcpServer) {}
}

// WithLogging is a no-op placeholder
//
//nolint:revive
func WithLogging() serverOption {
	return func(*mcpServer) {}
}

// WithPromptCapabilities is a no-op placeholder
//
//nolint:revive
func WithPromptCapabilities(_ bool) serverOption {
	return func(*mcpServer) {}
}

// WithResourceCapabilities is a no-op placeholder
//
//nolint:revive
func WithResourceCapabilities(_, _ bool) serverOption {
	return func(*mcpServer) {}
}

// AddTool registers a tool with the server
func (s *mcpServer) AddTool(tool Tool, _ any) {
	s.tools = append(s.tools, tool)
}

// ServeStdio is a no-op stub; OpenPass uses its own stdio transport
func ServeStdio(_ *mcpServer) error {
	return nil
}

// Shutdown gracefully shuts down the server
func (s *mcpServer) Shutdown(_ context.Context) error {
	return nil
}
