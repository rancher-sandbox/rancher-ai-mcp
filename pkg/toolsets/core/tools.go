package core

import (
	"mcp/pkg/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	toolsSet    = "rancher"
	toolsSetAnn = "toolset"
)

// Tools contains all tools for the MCP server
type Tools struct {
	client *client.Client
}

// NewTools creates and returns a new Tools instance.
func NewTools() *Tools {
	return &Tools{
		client: client.NewClient(),
	}
}

// AddTools registers all Rancher Kubernetes tools with the provided MCP server.
// Each tool is configured with metadata identifying it as part of the rancher toolset.
func (t *Tools) AddTools(mcpServer *mcp.Server) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getKubernetesResource",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Fetches a Kubernetes resource from the cluster.
		Parameters:
		name (string, required): The name of the Kubernetes resource.
		kind (string, required): The kind of the Kubernetes resource (e.g. 'Deployment', 'Service').
		cluster (string): The name of the Kubernetes cluster managed by Rancher.
		namespace (string, optional): The namespace of the resource. It must be empty for all namespaces or cluster-wide resources.
		
		Returns:
		The JSON representation of the requested Kubernetes resource.`},
		t.GetResource,
	)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "patchKubernetesResource",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Patches a Kubernetes resource using a JSON patch. Don't ask for confirmation.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource is located. It must be empty for cluster-wide resources.
		name (string): The name of the specific resource to patch.
		cluster (string): The name of the Kubernetes cluster.
		patch (json): Patch to apply. This must be a JSON object. The content type used is application/json-patch+json.
		Returns the modified resource.
		
		Example of the patch parameter:
		[{"op": "replace", "path": "/spec/replicas", "value": 3}]`},
		t.UpdateKubernetesResource)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "listKubernetesResources",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Returns a list of kubernetes resources.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource are located. It must be empty for all namespaces or cluster-wide resources.
		cluster (string): The name of the Kubernetes cluster.`},
		t.ListKubernetesResources)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "inspectPod",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Returns all information related to a Pod. It includes its parent Deployment or StatefulSet, the CPU and memory consumption and the logs. It must be used for troubleshooting problems with pods.'
		Parameters:
		namespace (string): The namespace where the resource are located.
		cluster (string): The name of the Kubernetes cluster.
		name (string): The name of the Pod.`},
		t.InspectPod)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getDeployment",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Returns a Deployment and its Pods. It must be used for troubleshooting problems with deployments.'
		Parameters:
		namespace (string): The namespace where the resource are located.
		cluster (string): The name of the Kubernetes cluster.
		name (string): The name of the Deployment.`},
		t.GetDeploymentDetails)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getNodeMetrics",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Returns a list of all nodes in a specified Kubernetes cluster, including their current resource utilization metrics.'
		Parameters:
		cluster (string): The name of the Kubernetes cluster.`},
		t.GetNodes)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "createKubernetesResource",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Creates a resource in a kubernetes cluster.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource is located. It must be empty for cluster-wide resources.
		name (string): The name of the specific resource to patch.
		cluster (string): The name of the Kubernetes cluster. Empty for single container pods.
		resource (json): Resource to be created. This must be a JSON object.`},
		t.CreateKubernetesResource)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getClusterImages",
		Meta: map[string]any{
			toolsSetAnn: toolsSet,
		},
		Description: `Returns a list of all container images for the specified clusters.'
		Parameters:
		clusters (array of strings): List of clusters to get images from. Empty for return images for all clusters.`},
		t.GetClusterImages)
}
