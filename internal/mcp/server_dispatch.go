package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/danieljustus/OpenPass/internal/metrics"
)

func toolError(msg string) *CallToolResult {
	return NewToolResultError(msg)
}

func (s *Server) executeTool(ctx context.Context, name string, args json.RawMessage) (map[string]any, error) {
	start := time.Now()
	agentName := ""
	if s.agent != nil {
		agentName = s.agent.Name
	}

	ctx, span := metrics.StartSpan(ctx, "executeTool",
		attribute.String("tool.name", name),
		attribute.String("agent.name", agentName),
		attribute.String("transport", s.transport),
	)
	defer span.End()

	req, err := decodeToolRequest(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		metrics.RecordMCPRequest(name, agentName, "error", time.Since(start))
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	if path, _ := req.RequireString("path"); path != "" {
		span.SetAttributes(attribute.String("entry.path", metrics.HashEntryPath(path)))
	}

	def, ok := findToolDefinition(name)
	if !ok {
		span.SetStatus(codes.Error, "unknown tool")
		metrics.RecordMCPRequest(name, agentName, "error", time.Since(start))
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	if def.Available != nil && !def.Available(s) {
		span.SetStatus(codes.Error, "tool not available")
		metrics.RecordMCPRequest(name, agentName, "error", time.Since(start))
		return nil, fmt.Errorf("tool %q is not available in the current environment", name)
	}

	// Check token tool scope
	if token, ok := TokenFromContext(ctx); ok {
		if !isToolAllowed(token, name) {
			span.SetStatus(codes.Error, "tool scope denied")
			metrics.RecordAuthDenial("tool_scope_denied", agentName)
			s.logAuditWithToken(ctx, "tool_scope_denied", name, false)
			return nil, fmt.Errorf("tool %q not allowed by token scope", name)
		}
		token.UpdateLastUsed()
	}

	result, err := def.Handler(s, ctx, req)

	duration := time.Since(start)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("status", "error"))
		metrics.RecordMCPRequest(name, agentName, "error", duration)
		return nil, err
	}

	status := "success"
	if result != nil && result.IsError {
		status = "error"
		span.SetStatus(codes.Error, "tool returned error")
	}
	span.SetAttributes(attribute.String("status", status))
	metrics.RecordMCPRequest(name, agentName, status, duration)

	return callToolResultPayload(result), nil
}
