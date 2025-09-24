package main

import (
	"log"
	"mcp/internal/tools"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "pod finder", Version: "v1.0.0"}, nil)
	tools := tools.NewTools()
	mcp.AddTool(server, &mcp.Tool{
		Name: "getKubernetesResource",
		Description: `Fetches a Kubernetes resource from the cluster. Use this tool to retrieve the YAML or JSON representation of any Kubernetes object.
Parameters:
name (string, required): The name of the Kubernetes resource.
kind (string, required): The kind of the Kubernetes resource (e.g. 'Deployment', 'Service').
cluster (string): The name of the Kubernetes cluster managed by Rancher.
namespace (string, optional): The namespace of the resource. This parameter is required for all namespaced resources (e.g., Pods, Deployments). It should be an empty string for cluster-scoped resources (e.g., 'Node', 'ClusterRole').

Returns:
The JSON representation of the requested Kubernetes resource.

Examples:
To get the my-pod in the default namespace in the local cluster:
getKubernetesResource(name='my-pod', kind='Pod', namespace='default', cluster='local')

To get the node-1 node (a cluster-scoped resource) in the local cluster:
getKubernetesResource(name='node-1', kind='Node', namespace=''), cluster='local'`},
		tools.GetResource)
	mcp.AddTool(server, &mcp.Tool{
		Name: "patchKubernetesResource",
		Description: `Patches a Kubernetes resource using a JSON patch. The JSON patch must be provided as a string. Don't ask for confirmation.'
Parameters:
kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
namespace (string): The namespace where the resource is located. It must be empty for cluster-wide resources.
name (string): The name of the specific resource to patch.
cluster (string): The name of the Kubernetes cluster.
patch (json): Patch to apply. This must be a JSON object. The content type used is application/json-patch+json.
Returns the modified resource.

Example of the patch parameter:
[{\"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": 3}]`},
		tools.UpdateKubernetesResource)
	mcp.AddTool(server, &mcp.Tool{
		Name: "listKubernetesResources",
		Description: `Returns a list of kubernetes resources.'
Parameters:
kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
namespace (string): The namespace where the resource are located. It must be empty for cluster-wide resources.
cluster (string): The name of the Kubernetes cluster.`},
		tools.ListKubernetesResources)
	mcp.AddTool(server, &mcp.Tool{
		Name: "getPod",
		Description: `Description: Returns a Pod, its parent Deployment or StatefulSet and the CPU and memory consumption. It must be used for troubleshooting problems with pods.'
Parameters:
namespace (string): The namespace where the resource are located.
cluster (string): The name of the Kubernetes cluster.
name (string): The name of the Pod.`},
		tools.GetPodDetails)
	mcp.AddTool(server, &mcp.Tool{
		Name: "getDeployment",
		Description: `Description: Returns a Deployment and its Pods. It must be used for troubleshooting problems with deployments.'
Parameters:
namespace (string): The namespace where the resource are located.
cluster (string): The name of the Kubernetes cluster.
name (string): The name of the Deployment.`},
		tools.GetDeploymentDetails)
	mcp.AddTool(server, &mcp.Tool{
		Name: "getNodeMetrics",
		Description: `Returns a list of all nodes in a specified Kubernetes cluster, including their current resource utilization metrics.'
Parameters:
cluster (string): The name of the Kubernetes cluster.`},
		tools.GetNodes)
	/*	mcp.AddTool(server, &mcp.Tool{
				Name: "createKubernetesResource",
				Description: `Returns a list of all nodes in a specified Kubernetes cluster, including their current resource utilization metrics.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource is located. It must be empty for cluster-wide resources.
		name (string): The name of the specific resource to patch.
		cluster (string): The name of the Kubernetes cluster. Empty for single container pods.
		resource (json): Resource to be created. This must be a JSON object.`},
				tools.CreateKubernetesResource)*/
	mcp.AddTool(server, &mcp.Tool{
		Name: "getPodLogs",
		Description: `Returns logs from a pod.'
Parameters:
namespace (string): The namespace where the pod is located.
name (string): The name of the pod.
container (string, optional): The name of the container. Leave empty if not specified.
cluster (string): The name of the Kubernetes cluster.`},
		tools.GetPodLogs)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{})

	log.Println("Listening on :9092")
	log.Fatal(http.ListenAndServe(":9092", handler))
}
