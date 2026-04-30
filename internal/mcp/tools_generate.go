package mcp

import (
	"context"

	"github.com/danieljustus/OpenPass/internal/crypto"
)

func (s *Server) handleGenerate(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	length := 16
	if v, err := req.RequireFloat("length"); err == nil {
		length = int(v)
	}
	symbols := req.GetBool("symbols", true)

	_ = s.checkScope("")

	password, err := generatePassword(length, symbols)
	if err != nil {
		s.logAudit(ctx, "generate", "password", false)
		return nil, err
	}

	s.logAudit(ctx, "generate", "password", true)
	return NewToolResultText(password), nil
}

func generatePassword(length int, symbols bool) (string, error) {
	return crypto.GeneratePassword(length, symbols)
}
