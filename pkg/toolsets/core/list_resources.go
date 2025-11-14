package core

import (
	"context"

	"mcp/internal/middleware"
	"mcp/pkg/client"
	"mcp/pkg/response"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// listKubernetesResourcesParams specifies the parameters needed to list kubernetes resources.
type listKubernetesResourcesParams struct {
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
}

// listKubernetesResources lists Kubernetes resources of a specific kind and namespace.
func (t *Tools) listKubernetesResources(ctx context.Context, toolReq *mcp.CallToolRequest, params listKubernetesResourcesParams) (*mcp.CallToolResult, any, error) {
	zap.L().Debug("listKubernetesResource called")

	resources, err := t.client.GetResources(ctx, client.ListParams{
		Cluster:   params.Cluster,
		Kind:      params.Kind,
		Namespace: params.Namespace,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     middleware.Token(ctx),
	})
	if err != nil {
		zap.L().Error("failed to list resources", zap.String("tool", "listKubernetesResource"), zap.Error(err))
		return nil, nil, err
	}

	mcpResponse, err := response.CreateMcpResponse(resources, params.Cluster)
	if err != nil {
		zap.L().Error("failed to create mcp response", zap.String("tool", "listKubernetesResource"), zap.Error(err))
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}
