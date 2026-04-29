package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/danieljustus/OpenPass/internal/crypto"
	"github.com/danieljustus/OpenPass/internal/vault"
	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func (s *Server) handleGenerateTOTP(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit("totp", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	if !s.checkScope(path) {
		s.logAudit("totp", path, false)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	svc := vaultsvc.New(s.vault)
	entry, err := svc.GetEntry(path)
	if err != nil {
		s.logAudit("totp", path, false)
		return vaultServiceErrorResult(err)
	}

	secret, algorithm, digits, period, hasTOTP := vault.ExtractTOTP(entry.Data)
	if !hasTOTP {
		s.logAudit("totp", path, false)
		return nil, fmt.Errorf("entry %q does not have TOTP configuration", path)
	}

	totpCode, err := crypto.GenerateTOTP(secret, algorithm, digits, period)
	if err != nil {
		s.logAudit("totp", path, false)
		return nil, fmt.Errorf("failed to generate TOTP code: %w", err)
	}

	s.logAudit("totp", path, true)
	result := map[string]any{
		"code":       totpCode.Code,
		"expires_at": totpCode.ExpiresAt,
		"period":     totpCode.Period,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal totp result: %w", err)
	}
	return NewToolResultText(string(resultJSON)), nil
}
