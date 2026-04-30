package mcp

import (
	"testing"
	"time"
)

func TestResolveToolAlias(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     string
	}{
		{
			name:     "real tool returns itself",
			toolName: "delete_entry",
			want:     "delete_entry",
		},
		{
			name:     "alias resolves to canonical",
			toolName: "openpass_delete",
			want:     "delete_entry",
		},
		{
			name:     "unknown tool returns original",
			toolName: "nonexistent_tool",
			want:     "nonexistent_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveToolAlias(tt.toolName)
			if got != tt.want {
				t.Errorf("resolveToolAlias(%q) = %q, want %q", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestIsToolAllowed(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)

	tests := []struct {
		name     string
		token    *ScopedToken
		toolName string
		want     bool
	}{
		{
			name:     "nil token allows all tools",
			token:    nil,
			toolName: "delete_entry",
			want:     true,
		},
		{
			name: "wildcard allows all tools",
			token: &ScopedToken{
				AllowedTools: []string{"*"},
			},
			toolName: "delete_entry",
			want:     true,
		},
		{
			name: "exact match allowed",
			token: &ScopedToken{
				AllowedTools: []string{"delete_entry", "list_entries"},
			},
			toolName: "delete_entry",
			want:     true,
		},
		{
			name: "tool not in allowed list",
			token: &ScopedToken{
				AllowedTools: []string{"list_entries"},
			},
			toolName: "delete_entry",
			want:     false,
		},
		{
			name: "alias allowed when canonical is in list",
			token: &ScopedToken{
				AllowedTools: []string{"delete_entry"},
			},
			toolName: "openpass_delete",
			want:     true,
		},
		{
			name: "canonical allowed when alias is in list",
			token: &ScopedToken{
				AllowedTools: []string{"openpass_delete"},
			},
			toolName: "delete_entry",
			want:     true,
		},
		{
			name: "expired token denies all",
			token: &ScopedToken{
				AllowedTools: []string{"*"},
				ExpiresAt:    &past,
			},
			toolName: "delete_entry",
			want:     false,
		},
		{
			name: "revoked token denies all",
			token: &ScopedToken{
				AllowedTools: []string{"*"},
				Revoked:      true,
			},
			toolName: "delete_entry",
			want:     false,
		},
		{
			name: "empty allowed list denies all",
			token: &ScopedToken{
				AllowedTools: []string{},
			},
			toolName: "delete_entry",
			want:     false,
		},
		{
			name:     "nil token with alias still allows",
			token:    nil,
			toolName: "openpass_delete",
			want:     true,
		},
		{
			name: "non-expired token with exact match allows",
			token: &ScopedToken{
				AllowedTools: []string{"list_entries"},
				ExpiresAt:    &future,
			},
			toolName: "list_entries",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isToolAllowed(tt.token, tt.toolName)
			if got != tt.want {
				t.Errorf("isToolAllowed(token=%v, toolName=%q) = %v, want %v", tt.token, tt.toolName, got, tt.want)
			}
		})
	}
}
