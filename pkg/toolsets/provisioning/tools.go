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
		Description: `Gets a cluster's complete configuration including provisioning and management clusters, the CAPI cluster, CAPI machines, and machine pool configs. 
					  This should be used when a complete overview of the clusters current state and its configuration is required.'

		Parameters:
		cluster (string): The name of the Kubernetes cluster
		namespace (string): The namespace where the resource is located. The default namespace will be used if not provided.
		`},
		t.AnalyzeCluster)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "analyzeClusterMachines",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Gets all Machine related resources for a cluster including Machines, MachineSets, and MachineDeployments.
					  This should be used when a summary or overview of just the existing machine resources is required.'

		Parameters:
		cluster (string): The name of the Kubernetes cluster
		namespace (string): The namespace where the resource is located. The default namespace will be used if not provided.
		`},
		t.AnalyzeClusterMachines)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getClusterMachine",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Gets a specific machine and its parent MachineSet and MachineDeployment.
   					  This should be used when detailed information about a specific machine is required.'

		Parameters:
		cluster (string): The name of the Kubernetes cluster
		machineName (string): The name of the machine to get
		`},
		t.GetClusterMachine)
}
