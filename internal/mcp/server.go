// Package mcp implements the Model Context Protocol (MCP) server for OpenPass.
// It provides AI agent integration via stdio and HTTP transports with
// configurable access control, audit logging, and vault operations.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danieljustus/OpenPass/internal/audit"
	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/vault"
)

const (
	defaultServerName    = "OpenPass MCP"
	defaultServerVersion = "1.0.0"
)

// Server provides the MCP server functionality for OpenPass.
// It handles agent authentication, vault access, and tool execution.
type Server struct {
	vault     *vault.Vault
	agent     *config.AgentProfile
	auditLog  *audit.Logger
	transport string
}

// New creates a new MCP server instance with the specified vault and agent configuration.
func New(v *vault.Vault, agentName string, transport string) (*Server, error) {
	if v == nil {
		return nil, errors.New("nil vault")
	}

	cfg := v.Config
	if cfg == nil {
		if v.Dir == "" {
			return nil, errors.New("vault config unavailable")
		}
		loaded, err := config.Load(filepath.Join(v.Dir, "config.yaml"))
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		cfg = loaded
	}

	if agentName == "" {
		agentName = cfg.DefaultAgent
	}

	agent, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %q not found", agentName)
	}
	agent.Name = agentName

	if cfg.Audit != nil {
		audit.SetConfig(&audit.Config{
			MaxFileSize: cfg.Audit.MaxFileSize,
			MaxBackups:  cfg.Audit.MaxBackups,
			MaxAgeDays:  cfg.Audit.MaxAgeDays,
		})
	}

	auditLog, err := audit.New(agentName, v.Dir)
	if err != nil {
		return nil, err
	}

	return &Server{
		vault:     v,
		agent:     &agent,
		auditLog:  auditLog,
		transport: transport,
	}, nil
}

// Build creates and configures an MCP server instance with tool capabilities.
//
//nolint:revive
func (s *Server) Build() *mcpServer {
	return NewMCPServer(
		defaultServerName,
		defaultServerVersion,
		WithToolCapabilities(true),
		WithLogging(),
	)
}

// ServeStdio runs the MCP server using stdio transport.
func (s *Server) ServeStdio(ctx context.Context) error {
	transport := NewStdioTransport()
	handler := NewProtocolHandler("OpenPass", "1.0.0", s)
	return transport.Start(ctx, handler.HandleMessage)
}

func (s *Server) authorize(path string, write bool, approved bool) error {
	if s == nil || s.agent == nil {
		return errors.New("server not initialized")
	}
	if path == "" {
		return errors.New("empty path")
	}

	if !s.checkScope(path) {
		s.logAudit("scope_denied", path, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return fmt.Errorf("path %q is outside agent scope", path)
	}

	if write && !s.canWrite() {
		s.logAudit("write_denied", path, false)
		metrics.RecordAuthDenial("write_denied", s.agent.Name)
		return fmt.Errorf("agent %q cannot write", s.agent.Name)
	}

	if write && s.requiresApproval() && !approved {
		s.logAudit("approval_required", path, false)
		metrics.RecordAuthDenial("approval_required", s.agent.Name)
		return fmt.Errorf("write to %q requires approval", path)
	}

	action := "read"
	if write {
		action = "write"
	}
	s.logAudit(action, path, approved)
	if write && approved {
		metrics.RecordApproval(s.agent.Name, "granted")
	}
	return nil
}

func (s *Server) logAudit(action, path string, ok bool) {
	if s == nil || s.auditLog == nil {
		return
	}
	reason := ""
	if !ok {
		reason = action // action IS the reason when denied (e.g., "scope_denied", "write_denied")
	}
	s.auditLog.LogEntry(audit.LogEntry{
		Agent:     s.agent.Name,
		Action:    action,
		Path:      path,
		Transport: s.transport,
		OK:        ok,
		Reason:    reason,
	})
}

func (s *Server) checkScope(path string) bool {
	if s == nil || s.agent == nil {
		return false
	}
	if len(s.agent.AllowedPaths) == 0 {
		return false
	}

	normalizedPath := normalizeScopePath(path)
	for _, allowed := range s.agent.AllowedPaths {
		if allowed == "*" {
			return true
		}
		normalizedAllowed := normalizeScopePath(allowed)
		if normalizedPath == normalizedAllowed {
			return true
		}
		if strings.HasPrefix(normalizedPath, normalizedAllowed+string(os.PathSeparator)) {
			return true
		}
	}

	return false
}

func (s *Server) canWrite() bool {
	return s != nil && s.agent != nil && s.agent.CanWrite
}

func (s *Server) requiresApproval() bool {
	if s == nil || s.agent == nil {
		return false
	}
	mode := s.agent.ApprovalMode
	if mode == "" {
		if s.agent.RequireApproval {
			mode = "prompt"
		} else {
			return false
		}
	}
	switch mode {
	case "none":
		return false
	case "deny":
		return true
	case "prompt":
		return true
	default:
		return false
	}
}

func (s *Server) shouldRedactField(field string) bool {
	if s == nil || s.agent == nil || s.agent.RedactFields == nil {
		return false
	}
	for _, pattern := range s.agent.RedactFields {
		if pattern == field || pattern == "*" {
			return true
		}
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, ".*")
			if strings.HasPrefix(field, prefix+".") {
				return true
			}
		}
	}
	return false
}

func redactEntry(entry *vault.Entry, redactFields []string) *vault.Entry {
	if entry == nil || redactFields == nil || len(redactFields) == 0 {
		return entry
	}

	redacted := &vault.Entry{
		Data:     make(map[string]any),
		Metadata: entry.Metadata,
	}

	for k, v := range entry.Data {
		redacted.Data[k] = redactValue(k, v, redactFields)
	}

	return redacted
}

func redactValue(field string, value any, redactFields []string) any {
	switch v := value.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k2, v2 := range v {
			nestedField := field + "." + k2
			result[k2] = redactValue(nestedField, v2, redactFields)
		}
		return result
	default:
		for _, pattern := range redactFields {
			if pattern == field || pattern == "*" {
				return "[REDACTED]"
			}
			if strings.HasSuffix(pattern, ".*") {
				prefix := strings.TrimSuffix(pattern, ".*")
				if strings.HasPrefix(field, prefix+".") {
					return "[REDACTED]"
				}
			}
		}
		return value
	}
}

func normalizeScopePath(path string) string {
	cleaned := filepath.Clean(strings.TrimSpace(filepath.FromSlash(path)))
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func toolError(msg string) *CallToolResult {
	return NewToolResultError(msg)
}

func (s *Server) executeTool(ctx context.Context, name string, args json.RawMessage) (map[string]any, error) {
	start := time.Now()
	agentName := ""
	if s.agent != nil {
		agentName = s.agent.Name
	}

	req, err := decodeToolRequest(args)
	if err != nil {
		metrics.RecordMCPRequest(name, agentName, "error", time.Since(start))
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	def, ok := findToolDefinition(name)
	if !ok {
		metrics.RecordMCPRequest(name, agentName, "error", time.Since(start))
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	if def.Available != nil && !def.Available(s) {
		metrics.RecordMCPRequest(name, agentName, "error", time.Since(start))
		return nil, fmt.Errorf("tool %q is not available in the current environment", name)
	}

	result, err := def.Handler(s, ctx, req)

	duration := time.Since(start)
	if err != nil {
		metrics.RecordMCPRequest(name, agentName, "error", duration)
		return nil, err
	}

	status := "success"
	if result != nil && result.IsError {
		status = "error"
	}
	metrics.RecordMCPRequest(name, agentName, status, duration)

	return callToolResultPayload(result), nil
}

// Close shuts down the server and closes the audit log.
func (s *Server) Close() error {
	if s == nil || s.auditLog == nil {
		return nil
	}
	return s.auditLog.Close()
}
