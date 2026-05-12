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

	// Gate 1: capability check — agent must have CanRunCommands (or, in future,
	// a dedicated CanGenerateDynamicSecrets flag). Without this, any agent with
	// tool access can generate arbitrary credentials.
	if !s.canRunCommands() {
		s.logAudit(ctx, "dynamic_secret_denied", provider, false)
		return nil, fmt.Errorf("dynamic secret generation not permitted for this agent")
	}

	// Gate 2: provider/role allowlist — reject requests for providers or roles
	// not explicitly granted in the agent profile. A nil map denies all dynamic
	// secret access (safe default).
	if s.agent.DynamicProviders != nil {
		allowedRoles, ok := s.agent.DynamicProviders[provider]
		if !ok {
			s.logAudit(ctx, "dynamic_secret_denied", provider, false)
			return toolError(fmt.Sprintf("provider %q is not in the agent's allowed dynamic providers", provider)), nil
		}
		if !containsRole(allowedRoles, role) {
			s.logAudit(ctx, "dynamic_secret_denied", provider, false)
			return toolError(fmt.Sprintf("role %q is not allowed for provider %q", role, provider)), nil
		}
	} else {
		// DynamicProviders is nil → all dynamic secret access is denied.
		s.logAudit(ctx, "dynamic_secret_denied", provider, false)
		return toolError("dynamic secret generation is not permitted for this agent"), nil
	}

	// Gate 3: approval gate — require user consent for every dynamic secret
	// generation, since this is a high-risk operation (grants database access,
	// cloud IAM roles, etc.).
	if s.requiresApproval() {
		if err := checkDynamicSecretApproval(s, provider, role, ttlStr); err != nil {
			s.logAudit(ctx, "dynamic_secret_denied", provider, false)
			return toolError(err.Error()), nil
		}
	}

	// All gates passed — proceed with generation.
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
		"lease_id":    secret.LeaseID,
		"engine_type": secret.EngineType,
		"expires_in":  secret.LeaseDuration.String(),
		"created_at":  secret.CreatedAt.Format(time.RFC3339),
		"credentials": secret.Data,
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return NewToolResultText(string(jsonResult)), nil
}

// containsRole checks whether the given role is in the allowlist.
// An empty allowlist (len==0) is treated as "no roles allowed".
func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role || r == "*" {
			return true
		}
	}
	return false
}

// checkDynamicSecretApproval checks the agent's approval mode and either
// allows generation, denies it, or prompts the user for confirmation.
func checkDynamicSecretApproval(s *Server, provider, role, ttl string) error {
	if s == nil || s.agent == nil {
		return fmt.Errorf("server not initialized")
	}

	mode := s.agent.ApprovalMode
	if mode == "" {
		if s.agent.RequireApproval {
			mode = "prompt"
		} else {
			mode = "none"
		}
	}

	switch mode {
	case "none", "auto":
		return nil
	case "deny":
		return fmt.Errorf("dynamic secret generation denied by policy")
	case "prompt":
		timeout := s.agent.ApprovalTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		result := RequestApproval(ApprovalRequest{
			Operation: "generate_dynamic_secret",
			Details:   fmt.Sprintf("Provider: %s, Role: %s, TTL: %s", provider, role, ttl),
			Timeout:   timeout,
		})
		if result.Error != nil {
			return fmt.Errorf("dynamic secret approval failed: %w", result.Error)
		}
		if !result.Approved {
			return fmt.Errorf("dynamic secret denied: user did not approve")
		}
		return nil
	default:
		return nil
	}
}
