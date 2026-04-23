package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/danieljustus/OpenPass/internal/crypto"
	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/vault"
)

func (s *Server) RegisterTools(srv *mcpServer) {
	for _, def := range availableToolDefinitions(s) {
		srv.AddTool(Tool{Name: def.Name, Description: def.Description}, def.Handler)
	}
}

func (s *Server) handleList(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	prefix, err := req.RequireString("prefix")
	if err != nil {
		prefix = ""
	}

	if !s.checkScope(prefix) {
		s.logAudit("list", prefix, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", prefix)
	}

	entries, err := vault.List(s.vault.Dir, prefix)
	if err != nil {
		s.logAudit("list", prefix, false)
		metrics.RecordVaultOperation("list", "error")
		return nil, err
	}

	s.logAudit("list", prefix, true)
	metrics.RecordVaultOperation("list", "success")
	result, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	return NewToolResultText(string(result)), nil
}

func (s *Server) handleGet(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit("get", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	if !s.checkScope(path) {
		s.logAudit("get", path, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	entry, err := vault.ReadEntry(s.vault.Dir, path, s.vault.Identity)
	if err != nil {
		s.logAudit("get", path, false)
		metrics.RecordVaultOperation("read", "error")
		return nil, err
	}

	if s.agent != nil && s.agent.RedactFields != nil && len(s.agent.RedactFields) > 0 {
		entry = redactEntry(entry, s.agent.RedactFields)
	}

	s.logAudit("get", path, true)
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
	_ = ctx
	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit("get_metadata", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	if !s.checkScope(path) {
		s.logAudit("get_metadata", path, false)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	meta, err := vault.GetEntryMetadata(s.vault.Dir, path, s.vault.Identity)
	if err != nil {
		s.logAudit("get_metadata", path, false)
		return nil, err
	}

	s.logAudit("get_metadata", path, true)

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

func (s *Server) handleFind(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	query, err := req.RequireString("query")
	if err != nil {
		s.logAudit("find", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	matches, err := s.findEntries(query)
	if err != nil {
		s.logAudit("find", query, false)
		return nil, err
	}

	filtered := make([]vault.Match, 0, len(matches))
	for _, match := range matches {
		if s.checkScope(match.Path) {
			filtered = append(filtered, match)
		}
	}

	s.logAudit("find", query, true)
	result, err := json.Marshal(filtered)
	if err != nil {
		return nil, err
	}
	return NewToolResultText(string(result)), nil
}

func (s *Server) handleSet(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	if !s.canWrite() {
		s.logAudit("set", "<write-denied>", false)
		metrics.RecordAuthDenial("write_denied", s.agent.Name)
		return nil, fmt.Errorf("write operations not permitted for this agent")
	}

	path, err := req.RequireString("path")
	if err != nil {
		s.logAudit("set", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}
	field, err := req.RequireString("field")
	if err != nil {
		s.logAudit("set", path, false)
		return NewToolResultError(err.Error()), nil
	}
	value, err := req.RequireString("value")
	if err != nil {
		s.logAudit("set", path, false)
		return NewToolResultError(err.Error()), nil
	}

	if !s.checkScope(path) {
		s.logAudit("set", path, false)
		metrics.RecordAuthDenial("scope_denied", s.agent.Name)
		return nil, fmt.Errorf("access denied: path %q outside allowed scope", path)
	}

	if s.requiresApproval() {
		s.logAudit("approval_denied", path, false)
		metrics.RecordApproval(s.agent.Name, "denied")
		return nil, fmt.Errorf("write to %q denied: approval required but cannot be granted", path)
	}

	// Prepare the partial data to merge
	partialData := make(map[string]any)
	if field == "totp" {
		var totpData map[string]any
		if err := json.Unmarshal([]byte(value), &totpData); err != nil {
			return NewToolResultError(fmt.Sprintf("invalid TOTP JSON: %v", err)), nil
		}
		partialData[field] = totpData
	} else {
		partialData[field] = value
	}

	if err := s.upsertEntry(path, partialData, "set"); err != nil {
		return nil, err
	}

	s.logAudit("set", path, true)
	metrics.RecordVaultOperation("write", "success")
	return NewToolResultText(fmt.Sprintf("Set %s.%s = ***", path, field)), nil
}

func (s *Server) handleGenerate(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	length := 16
	if v, err := req.RequireFloat("length"); err == nil {
		length = int(v)
	}
	symbols := req.GetBool("symbols", true)

	_ = s.checkScope("")

	password, err := generatePassword(length, symbols)
	if err != nil {
		s.logAudit("generate", "password", false)
		return nil, err
	}

	s.logAudit("generate", "password", true)
	return NewToolResultText(password), nil
}

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

	_, err = vault.ReadEntry(s.vault.Dir, path, s.vault.Identity)
	if err != nil {
		s.logAudit("delete", path, false)
		metrics.RecordVaultOperation("delete", "error")
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("entry not found: %s", path)
		}
		return nil, fmt.Errorf("cannot read entry: %w", err)
	}

	if err := vault.DeleteEntry(s.vault.Dir, path); err != nil {
		s.logAudit("delete", path, false)
		metrics.RecordVaultOperation("delete", "error")
		return nil, fmt.Errorf("failed to delete entry: %w", err)
	}

	s.logAudit("delete", path, true)
	metrics.RecordVaultOperation("delete", "success")
	return NewToolResultText(fmt.Sprintf("Successfully deleted entry: %s", path)), nil
}

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

	entry, err := vault.ReadEntry(s.vault.Dir, path, s.vault.Identity)
	if err != nil {
		s.logAudit("totp", path, false)
		return nil, err
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

func (s *Server) handleHealth(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_, _ = ctx, req
	result := map[string]any{
		"status":    "healthy",
		"server":    defaultServerName,
		"version":   defaultServerVersion,
		"transport": s.transport,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return NewToolResultText(string(resultJSON)), nil
}

// findEntries searches vault entries matching a query.
// Performance: Uses path-only fast path to avoid decrypting entries when possible.
// If query appears in a path, entry is returned without decryption.
// Only entries where path doesn't match are decrypted to search field content.
func (s *Server) findEntries(query string) ([]vault.Match, error) {
	paths, err := vault.List(s.vault.Dir, "")
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(query)
	var matches []vault.Match
	pathOnlyMatches := make([]vault.Match, 0)
	pathsNeedingDecrypt := make([]string, 0, len(paths))

	// First pass: separate path-only matches from paths needing field search
	for _, path := range paths {
		if !s.checkScope(path) {
			continue
		}

		if needle == "" || strings.Contains(strings.ToLower(path), needle) {
			// Path matches - no decryption needed
			pathOnlyMatches = append(pathOnlyMatches, vault.Match{Path: path, Fields: []string{"path"}})
		} else {
			// Path doesn't match, need to decrypt and search fields
			pathsNeedingDecrypt = append(pathsNeedingDecrypt, path)
		}
	}

	// Second pass: only decrypt entries where path didn't match
	for _, path := range pathsNeedingDecrypt {
		entry, err := vault.ReadEntry(s.vault.Dir, path, s.vault.Identity)
		if err != nil {
			return nil, err
		}

		fields := make(map[string]struct{})
		vault.CollectFieldMatches(fields, "", entry.Data, needle)
		if len(fields) == 0 {
			continue
		}

		matchFields := make([]string, 0, len(fields))
		for field := range fields {
			matchFields = append(matchFields, field)
		}
		sort.Strings(matchFields)
		matches = append(matches, vault.Match{Path: path, Fields: matchFields})
	}

	// Combine path-only matches with field matches
	matches = append(matches, pathOnlyMatches...)

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Path < matches[j].Path
	})
	return matches, nil
}

func generatePassword(length int, symbols bool) (string, error) {
	return crypto.GeneratePassword(length, symbols)
}

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

	if err := s.upsertEntry(path, partialData, "secure_input"); err != nil {
		return nil, err
	}

	s.logAudit("secure_input", path, true)
	metrics.RecordVaultOperation("write", "success")
	return NewToolResultText(fmt.Sprintf("Securely stored %s.%s = *** (value hidden from agent)", path, field)), nil
}

func (s *Server) upsertEntry(path string, partialData map[string]any, action string) error {
	_, readErr := vault.ReadEntry(s.vault.Dir, path, s.vault.Identity)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			now := time.Now().UTC()
			newEntry := &vault.Entry{
				Data: partialData,
				Metadata: vault.EntryMetadata{
					Created: now,
					Updated: now,
					Version: 1,
				},
			}
			if err := vault.WriteEntryWithRecipients(s.vault.Dir, path, newEntry, s.vault.Identity); err != nil {
				s.logAudit(action, path, false)
				metrics.RecordVaultOperation("write", "error")
				return fmt.Errorf("create entry: %w", err)
			}
			return nil
		}
		s.logAudit(action, path, false)
		metrics.RecordVaultOperation("write", "error")
		return fmt.Errorf("read entry: %w", readErr)
	}

	if _, err := vault.MergeEntryWithRecipients(s.vault.Dir, path, partialData, s.vault.Identity); err != nil {
		s.logAudit(action, path, false)
		metrics.RecordVaultOperation("write", "error")
		return fmt.Errorf("update entry: %w", err)
	}
	return nil
}
