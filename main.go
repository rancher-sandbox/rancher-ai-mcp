package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"mcp/internal/tools"
)

func init() {
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	} else {
		zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	}
}

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
		namespace (string, optional): The namespace of the resource. It must be empty for all namespaces or cluster-wide resources.
		
		Returns:
		The JSON representation of the requested Kubernetes resource.`},
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
		[{"op": "replace", "path": "/spec/replicas", "value": 3}]`},
		tools.UpdateKubernetesResource)
	mcp.AddTool(server, &mcp.Tool{
		Name: "listKubernetesResources",
		Description: `Returns a list of kubernetes resources.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource are located. It must be empty for all namespaces or cluster-wide resources.
		cluster (string): The name of the Kubernetes cluster.`},
		tools.ListKubernetesResources)
	mcp.AddTool(server, &mcp.Tool{
		Name: "inspectPod",
		Description: `Description: Returns all information related to a Pod. It includes its parent Deployment or StatefulSet, the CPU and memory consumption and the logs. It must be used for troubleshooting problems with pods.'
		Parameters:
		namespace (string): The namespace where the resource are located.
		cluster (string): The name of the Kubernetes cluster.
		name (string): The name of the Pod.`},
		tools.InspectPod)
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
	mcp.AddTool(server, &mcp.Tool{
		Name: "createKubernetesResource",
		Description: `Returns a list of all nodes in a specified Kubernetes cluster, including their current resource utilization metrics.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource is located. It must be empty for cluster-wide resources.
		name (string): The name of the specific resource to patch.
		cluster (string): The name of the Kubernetes cluster. Empty for single container pods.
		resource (json): Resource to be created. This must be a JSON object.`},
		tools.CreateKubernetesResource)
	mcp.AddTool(server, &mcp.Tool{
		Name: "getClusterImages",
		Description: `Returns a list of all container images for the specified clusters.'
		Parameters:
		clusters (array of strings): List of clusters to get images from. Empty for return images for all clusters.`},
		tools.GetClusterImages)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{})

	zap.L().Info("MCP Server started!")
	log.Fatal(http.ListenAndServe(":9092", handler))
}
