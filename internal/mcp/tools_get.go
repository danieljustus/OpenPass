package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func vaultServiceErrorResult(err error) (*CallToolResult, error) {
	var cliErr *errorspkg.CLIError
	if errors.As(err, &cliErr) {
		if cliErr.Kind == errorspkg.ErrNotFound || cliErr.Kind == errorspkg.ErrFieldNotFound {
			return NewToolResultError(cliErr.Message), nil
		}
		return nil, fmt.Errorf("vault operation failed: %w", err)
	}
	return nil, err
}

func (s *Server) handleGet(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit(ctx, "get", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	if !s.checkScope(path) {
		s.logAudit(ctx, "get", path, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	svc := vaultsvc.New(s.vault)
	_, span := metrics.StartSpan(ctx, "vault.GetEntry")
	entry, err := svc.GetEntry(path)
	span.End()
	if err != nil {
		s.logAudit(ctx, "get", path, false)
		metrics.RecordVaultOperation("read", "error")
		return vaultServiceErrorResult(err)
	}

	if s.agent != nil && s.agent.RedactFields != nil && len(s.agent.RedactFields) > 0 {
		entry = redactEntry(entry, s.agent.RedactFields)
	}

	s.logAudit(ctx, "get", path, true)
	metrics.RecordVaultOperation("read", "success")

	includeMetadata := req.GetBool("include_metadata", false)

	var result []byte
	if includeMetadata {
		response := map[string]any{
			"data": entry.Data,
			"meta": map[string]any{
				"created": entry.Metadata.Created.Format(time.RFC3339),
				"updated": entry.Metadata.Updated.Format(time.RFC3339),
				"version": entry.Metadata.Version,
			},
		}
		result, err = json.Marshal(response)
	} else {
		result, err = json.Marshal(entry)
	}

	if err != nil {
		return nil, err
	}
	return NewToolResultText(string(result)), nil
}

func (s *Server) handleGetMetadata(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit(ctx, "get_metadata", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	if !s.checkScope(path) {
		s.logAudit(ctx, "get_metadata", path, false)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	svc := vaultsvc.New(s.vault)
	entry, err := svc.GetEntry(path)
	if err != nil {
		s.logAudit(ctx, "get_metadata", path, false)
		return vaultServiceErrorResult(err)
	}
	meta := entry.Metadata

	s.logAudit(ctx, "get_metadata", path, true)

	result := map[string]any{
		"path":    path,
		"exists":  true,
		"created": meta.Created.Format(time.RFC3339),
		"updated": meta.Updated.Format(time.RFC3339),
		"version": meta.Version,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return NewToolResultText(string(resultJSON)), nil
}
