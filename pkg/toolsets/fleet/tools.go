package fleet

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rancher/rancher-ai-mcp/pkg/client"
)

const (
	toolsSet    = "fleet"
	toolsSetAnn = "toolset"
	tokenHeader = "R_token"
	urlHeader   = "R_url"
)

// Tools contains all tools for the MCP server
type Tools struct {
	client *client.Client
}

// NewTools creates and returns a new Tools instance.
func NewTools(client *client.Client) *Tools {
	return &Tools{
		client: client,
	}
}

// AddTools registers all Rancher Kubernetes tools with the provided MCP server.
// Each tool is configured with metadata identifying it as part of the rancher toolset.
func (t *Tools) AddTools(mcpServer *mcp.Server) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "listGitRepos",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `List GitRepos.
		Parameters:
		workspace (string, required): The workspace of the GitRepos.
		
		Returns:
		List of all GitRepos in the workspace.`},
		t.listGitRepos,
	)
}
