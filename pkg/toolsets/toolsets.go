package toolsets

import (
	"mcp/pkg/client"
	"mcp/pkg/toolsets/core"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolsAdder is an interface for types that can add tools to an MCP server.
type toolsAdder interface {
	AddTools(mcpServer *mcp.Server)
}

// ToolSets manages a collection of tool adders for the MCP server.
type ToolSets struct {
	toolsAdders []toolsAdder
}

// NewToolSetsWithAllTools creates a new ToolSets instance with all available toolsets initialized.
func NewToolSetsWithAllTools(client *client.Client) *ToolSets {
	toolSets := &ToolSets{}

	toolSets.toolsAdders = []toolsAdder{core.NewTools(client)}

	return toolSets
}

// AddTools registers all tools from all toolsets to the provided MCP server.
func (t *ToolSets) AddTools(mcpServer *mcp.Server) {
	for _, ta := range t.toolsAdders {
		ta.AddTools(mcpServer)
	}
}
