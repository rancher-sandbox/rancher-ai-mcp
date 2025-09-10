package main

import (
	"log"
	"mcp/internal/tools"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "pod finder", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name: "getKubernetesResource",
		Description: `Description:
Fetches a Kubernetes resource from the cluster. Use this tool to retrieve the YAML or JSON representation of any Kubernetes object.
Parameters:
name (string, required): The name of the Kubernetes resource.
kind (string, required): The kind of the Kubernetes resource (e.g., 'Pod', 'Deployment', 'Service').
cluster (string): The name of the Kubernetes cluster managed by Rancher.
namespace (string, optional): The namespace of the resource. This parameter is required for all namespaced resources (e.g., Pods, Deployments). It should be an empty string for cluster-scoped resources (e.g., 'Node', 'ClusterRole').

Returns:
The JSON representation of the requested Kubernetes resource.

Usage Notes:
Use this tool when a user asks for information about a specific Kubernetes resource.
Always provide the name and kind parameters.
If the kind is a namespaced resource, you must provide the namespace.
If the kind is a cluster-scoped resource, leave the namespace parameter as an empty string.

Examples:
To get the my-pod in the default namespace in the local cluster:
getKubernetesResource(name='my-pod', kind='Pod', namespace='default', cluster='local')

To get the node-1 node (a cluster-scoped resource) in the local cluster:
getKubernetesResource(name='node-1', kind='Node', namespace=''), cluster='local'`},
		tools.GetResource)
	mcp.AddTool(server, &mcp.Tool{
		Name: "patchKubernetesResource",
		Description: `Description: Patches a namespaced Kubernetes resource using a JSON patch. The JSON patch must be provided as a string. Don't ask for confirmation.'
Parameters:
kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
namespace (string): The namespace where the resource is located.
name (string): The name of the specific resource to patch.
cluster (string): The name of the Kubernetes cluster managed by Rancher.
patch (json): Patch to apply. This must be a JSON object. The content type used is application/json-patch+json.
Returns the modified resource.

Example of the patch parameter:
[{\"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": 3}]`},
		tools.UpdateKubernetesResource)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{})

	log.Println("Listening on :9092")
	log.Fatal(http.ListenAndServe(":9092", handler))
}
