package toolsets

import (
	"mcp/pkg/toolsets/rancher"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type toolsAdder interface {
	AddTools(mcpServer *mcp.Server)
}

type ToolSets struct {
	toolsAdders []toolsAdder
}

func NewToolSetsWithAllTools(mcpServer *mcp.Server) *ToolSets {
	toolSets := &ToolSets{}

	toolSets.toolsAdders = []toolsAdder{rancher.NewTools()}

	return toolSets
}

func (t *ToolSets) AddTools(mcpServer *mcp.Server) {
	for _, ta := range t.toolsAdders {
		ta.AddTools(mcpServer)
	}
}
