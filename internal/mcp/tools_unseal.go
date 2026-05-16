package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/vault/taint"
	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func (s *Server) handleSecretUnseal(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	handleStr, err := req.RequireString("handle")
	if err != nil {
		s.logAudit(ctx, "secret_unseal", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	handle, ok := taint.ParseSecretHandle(handleStr)
	if !ok {
		s.logAudit(ctx, "secret_unseal", handleStr, false)
		return NewToolResultError(fmt.Sprintf("invalid handle format: %s", handleStr)), nil
	}

	entryPath := handle.Path
	if handle.Field != "" {
		entryPath = handle.Path + "/" + handle.Field
	}

	cacheKey := approvalCacheKey(s.agent.Name, "secret_unseal", handleStr)
	if s.approvalCache == nil || !s.approvalCache.isRemembered(cacheKey) {
		if approvalErr := s.requireApproval(ctx, Intent{
			Action:    "secret_unseal",
			EntryPath: entryPath,
			Summary:   RenderSummary("unseal secret", entryPath, ""),
		}); approvalErr != nil {
			s.logAudit(ctx, "secret_unseal", entryPath, false)
			return NewToolResultError(approvalErr.Error()), nil
		}
		if s.approvalCache != nil {
			s.approvalCache.setRemembered(cacheKey)
			s.logAudit(ctx, "secret_unseal_remembered", entryPath, true)
		}
	}

	if s.agent.MaxSecretsInSession > 0 {
		accessed := s.secretsAccessed.Load()
		if accessed >= int64(s.agent.MaxSecretsInSession) {
			return NewToolResultError(
				fmt.Sprintf("max secrets per session exceeded (%d/%d)", accessed, s.agent.MaxSecretsInSession)), nil
		}
	}

	svc := vaultsvc.New(slog.Default(), s.vault)
	entry, vaultErr := svc.GetEntry(handle.Path)
	if vaultErr != nil {
		s.logAudit(ctx, "secret_unseal", entryPath, false)
		metrics.RecordVaultOperation("read", "error")
		return NewToolResultError(fmt.Sprintf("entry not found: %s", handle.Path)), nil
	}

	var value string
	if handle.Field == "" {
		dataJSON, marshalErr := json.Marshal(entry.Data)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal entry data: %w", marshalErr)
		}
		value = string(dataJSON)
	} else {
		val, found := entry.Data[handle.Field]
		if !found {
			s.logAudit(ctx, "secret_unseal", entryPath, false)
			return NewToolResultError(fmt.Sprintf("field %q not found in entry %s", handle.Field, handle.Path)), nil
		}
		value = fmt.Sprintf("%v", val)
	}

	s.secretsAccessed.Add(1)
	s.logAudit(ctx, "secret_unseal", entryPath, true)
	metrics.RecordVaultOperation("read", "success")

	return NewToolResultText(value), nil
}
