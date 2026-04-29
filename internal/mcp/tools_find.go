package mcp

import (
	"context"
	"encoding/json"

	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/vault"
	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

func (s *Server) handleFind(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	_ = ctx
	query, err := req.RequireString("query")
	if err != nil {
		s.logAudit("find", "<invalid>", false)
		return NewToolResultError(err.Error()), nil
	}

	matches, err := s.findEntries(ctx, query)
	if err != nil {
		s.logAudit("find", query, false)
		return nil, err
	}

	s.logAudit("find", query, true)
	result, err := json.Marshal(matches)
	if err != nil {
		return nil, err
	}
	return NewToolResultText(string(result)), nil
}

// findEntries searches vault entries matching a query.
// It delegates to vaultsvc for concurrent search with scope filtering applied
// before decryption.
func (s *Server) findEntries(ctx context.Context, query string) ([]vault.Match, error) {
	svc := vaultsvc.New(s.vault)
	_, span := metrics.StartSpan(ctx, "vault.Find")
	defer span.End()
	return svc.Find(query, vaultsvc.FindOptions{
		MaxWorkers:  4,
		ScopeFilter: s.checkScope,
	})
}
