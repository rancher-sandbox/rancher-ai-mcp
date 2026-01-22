package toolsets

import (
	"mcp/pkg/client"
	"mcp/pkg/toolsets/core"
	"mcp/pkg/toolsets/fleet"
	"mcp/pkg/toolsets/provisioning"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolsAdder is an interface for types that can add tools to an MCP server.
type toolsAdder interface {
	AddTools(mcpServer *mcp.Server)
}

// AddAllTools adds all available tools to the MCP server.
func AddAllTools(client *client.Client, mcpServer *mcp.Server) {
	for _, ta := range allToolSets(client) {
		ta.AddTools(mcpServer)
	}
}

func allToolSets(client *client.Client) []toolsAdder {
	return []toolsAdder{
		core.NewTools(client),
		fleet.NewTools(client),
		provisioning.NewTools(client),
	}
}
