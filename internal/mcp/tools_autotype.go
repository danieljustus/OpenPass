package mcp

import (
	"context"
	"fmt"

	"github.com/danieljustus/OpenPass/internal/autotype"
	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func (s *Server) handleAutotype(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	if !s.canUseAutotype() {
		s.logAudit(ctx, "autotype", "<autotype-denied>", false)
		metrics.RecordAuthDenial("autotype_denied", s.agent.Name)
		return nil, fmt.Errorf("autotype operations not permitted for this agent")
	}

	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit(ctx, "autotype", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	field := req.GetString("field", "password")

	if !s.checkScope(path) {
		s.logAudit(ctx, "autotype", path, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	if s.requiresApproval() {
		s.logAudit(ctx, "autotype", path, false)
		metrics.RecordApproval(s.agent.Name, "denied")
		return nil, fmt.Errorf("autotype denied: approval required but cannot be granted")
	}

	svc := vaultsvc.New(s.vault)
	value, err := svc.GetField(path, field)
	if err != nil {
		s.logAudit(ctx, "autotype", path, false)
		metrics.RecordVaultOperation("read", "error")
		return vaultServiceErrorResult(err)
	}

	strValue, ok := value.(string)
	if !ok {
		s.logAudit(ctx, "autotype", path, false)
		return NewToolResultError(fmt.Sprintf("field %q is not a string", field)), nil
	}

	at := autotype.DefaultAutotype()
	if at == nil {
		return NewToolResultError("autotype not available on this platform"), nil
	}

	if err := at.Type(strValue); err != nil {
		s.logAudit(ctx, "autotype", path, false)
		return NewToolResultError(fmt.Sprintf("autotype failed: %v", err)), nil
	}

	s.logAudit(ctx, "autotype", path, true)
	metrics.RecordVaultOperation("read", "success")

	return NewToolResultText(`{"success": true}`), nil
}
