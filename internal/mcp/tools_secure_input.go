package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/danieljustus/OpenPass/internal/metrics"
)

func (s *Server) handleSecureInput(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	if !s.canWrite() {
		s.logAudit("secure_input", "<write-denied>", false)
		metrics.RecordAuthDenial("write_denied", s.agent.Name)
		return nil, fmt.Errorf("write operations not permitted for this agent")
	}

	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit("secure_input", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}
	field, err := req.RequireString("field")
	if err != nil {
		s.logAudit("secure_input", path, false)
		return NewToolResultError(err.Error()), nil
	}
	description := req.GetString("description", "")

	if !s.checkScope(path) {
		s.logAudit("secure_input", path, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	if s.requiresApproval() {
		s.logAudit("approval_denied", path, false)
		metrics.RecordApproval(s.agent.Name, "denied")
		return nil, fmt.Errorf("secure input for %q denied: approval required but cannot be granted", path)
	}

	prompt := buildSecureInputPrompt(path, field, description)
	value, inputErr := SecureInputPrompt(prompt, 60*time.Second)
	if inputErr != nil {
		s.logAudit("secure_input", path, false)
		metrics.RecordVaultOperation("secure_input", "error")
		return nil, fmt.Errorf("secure input failed: %w", inputErr)
	}

	if value == "" {
		s.logAudit("secure_input", path, false)
		return NewToolResultError("secure input canceled: empty value provided"), nil
	}

	partialData := make(map[string]any)
	partialData[field] = value

	if err := s.upsertEntry(ctx, path, partialData, "secure_input"); err != nil {
		return nil, err
	}

	s.logAudit("secure_input", path, true)
	metrics.RecordVaultOperation("write", "success")
	return NewToolResultText(fmt.Sprintf("Securely stored %s.%s = *** (value hidden from agent)", path, field)), nil
}
