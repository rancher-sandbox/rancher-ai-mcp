package provisioning

import (
	"mcp/pkg/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	toolsSet    = "provisioning"
	toolsSetAnn = "toolset"
	tokenHeader = "R_token"
	urlHeader   = "R_url"
)

type Tools struct {
	client *client.Client
}

func NewTools(client *client.Client) *Tools {
	return &Tools{
		client: client,
	}
}

func (t *Tools) AddTools(mcpServer *mcp.Server) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "analyzeCluster",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Retrieve several resources that represent a cluster and its associated machines'
		Parameters:
		cluster (string): The name of the Kubernetes cluster
		namespace (string): The namespace where the resource is located. This is an optional field that can be omitted if needed.
		`},
		t.AnalyzeCluster)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "analyzeClusterMachines",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Returns a set of kubernetes resources that represent all machine related resources for a cluster.'
		Parameters:
		cluster (string): The name of the Kubernetes cluster
		namespace (string): The namespace where the resource is located. This is an optional field that can be omitted if needed.
		`},
		t.AnalyzeClusterMachines)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getClusterMachine",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Returns a set of kubernetes resources that represent a single machine within a cluster.'
		Parameters:
		cluster (string): The name of the Kubernetes cluster
		machineName (string): The name of the machine to get
		`},
		t.GetClusterMachine)
}
