package mcp

import (
	"context"
	"fmt"

	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func (s *Server) handleDelete(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	if !s.canWrite() {
		s.logAudit("delete", "<write-denied>", false)
		metrics.RecordAuthDenial("write_denied", s.agent.Name)
		return nil, fmt.Errorf("delete operations not permitted for this agent")
	}

	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit("delete", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	if !s.checkScope(path) {
		s.logAudit("delete", path, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	if s.requiresApproval() {
		s.logAudit("approval_denied", path, false)
		metrics.RecordApproval(s.agent.Name, "denied")
		return nil, fmt.Errorf("delete of %q denied: approval required but cannot be granted", path)
	}

	svc := vaultsvc.New(s.vault)
	_, span := metrics.StartSpan(ctx, "vault.Delete")
	err = svc.Delete(path)
	span.End()
	if err != nil {
		s.logAudit("delete", path, false)
		metrics.RecordVaultOperation("delete", "error")
		return vaultServiceErrorResult(err)
	}

	s.logAudit("delete", path, true)
	metrics.RecordVaultOperation("delete", "success")
	return NewToolResultText(fmt.Sprintf("Successfully deleted entry: %s", path)), nil
}
