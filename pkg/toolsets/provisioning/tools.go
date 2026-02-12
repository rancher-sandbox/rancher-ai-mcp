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
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "listK3kClusters",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `List K3k virtual clusters deployed across downstream clusters.

		Parameters:
		clusters (array of strings): List of clusters to get virtual clusters from. Empty for return virtual clusters for all clusters.
		`},
		t.getK3kClusters)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "createK3kCluster",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Create a new K3k cluster in a specific downstream cluster.

		Parameters:
		name (string): The name of the K3k cluster.
		namespace (string): The namespace where the K3k cluster will be created.
		targetCluster (string): The downstream cluster where the K3k resource will be applied.
		version (string): Optional. The k3s/k8s version for the cluster (e.g., 'v1.33.1-k3s1'). Defaults to 'host cluster version'.
		mode (string): Optional. Cluster mode (e.g., 'shared' or 'virtual'). Defaults to 'shared'.
		servers (int): Optional. Number of server (control plane) nodes. Defaults to 1.
		agents (int): Optional. Number of agent (worker) nodes. Defaults to 0.
		sync (object): Optional. shared mode only. Resource synchronization options with boolean flags for 'priorityClasses' and 'ingresses'.
		serverLimit (object): Optional. Resource constraints for server nodes (contains 'cpu' and 'memory' strings).
		workerLimit (object): Optional. Resource constraints for worker nodes (contains 'cpu' and 'memory' strings).
		persistence (object): Optional. Storage settings for etcd data (contains 'type' ('dynamic' or 'ephemeral'), 'storageClassName', 'storageRequest' strings).
		`},
		t.createK3kCluster)
}
