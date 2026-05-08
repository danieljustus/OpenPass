package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/danieljustus/OpenPass/internal/dynamicsecret"
	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func (s *Server) handleGenerateDynamicSecret(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	provider, err := req.RequireString("provider")
	if err != nil {
		return toolError("provider is required"), nil
	}

	role, err := req.RequireString("role")
	if err != nil {
		return toolError("role is required"), nil
	}

	ttlStr := req.GetString("ttl", "1h")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		return toolError(fmt.Sprintf("invalid ttl: %v", err)), nil
	}

	permissions := req.GetString("permissions", "")

	_ = s.checkScope("")

	svc := vaultsvc.New(slog.Default(), s.vault)
	mgr := dynamicsecret.NewManager(svc)
	generateReq := dynamicsecret.GenerateRequest{
		Role:        role,
		TTL:         ttl,
		Permissions: permissions,
	}

	secret, err := mgr.Generate(ctx, provider, generateReq)
	if err != nil {
		s.logAudit(ctx, "dynamic_secret_failed", provider, false)
		return toolError(fmt.Sprintf("generate dynamic secret: %v", err)), nil
	}

	s.logAudit(ctx, "dynamic_secret", provider, true)

	result := map[string]any{
		"lease_id":     secret.LeaseID,
		"engine_type":  secret.EngineType,
		"expires_in":   secret.LeaseDuration.String(),
		"created_at":   secret.CreatedAt.Format(time.RFC3339),
		"credentials":  secret.Data,
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewToolResultText(string(jsonResult)), nil
}
