package utils

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

func NewChildLogger(toolReq *mcp.CallToolRequest, extras map[string]string) *zap.Logger {
	args := []zap.Field{
		zap.String("tool-name", toolReq.Params.Name),
	}
	if toolReq.Session != nil && toolReq.Session.ID() != "" {
		args = append(args, zap.String("mcp-request-id", toolReq.Session.ID()))
	}
	for k, v := range extras {
		args = append(args, zap.String(k, v))
	}
	return zap.L().With(args...)
}
