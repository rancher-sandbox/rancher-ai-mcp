package main

import (
	"log"
	"mcp/internal/tools"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "pod finder", Version: "v1.0.0"}, nil)
	/*mcp.AddTool(server, &mcp.Tool{
	Name:        "getKubernetesResource",
	Description: "Returns non-namespaced kubernetes resource. Resources supported: User, Cluster, ClusterRole, ClusterRoleBinding, PersistentVolume"},
	tools.GetNonNamespacedKubernetesResource)*/
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getKubernetesNamespacedResource",
		Description: "Returns namespaced kubernetes resource. Resources supported: Pod, Service, ConfigMap, Secret, Deployment, StatefulSet, DaemonSet, ReplicaSet, Ingress, NetworkPolicy, HorizontalPodAutoscaler, ServiceAccount, Role, RoleBinding, PersistentVolumeClaim, Project, Bundle, GitRepo"},
		tools.GetNamespacedKubernetesResource)
	/*mcp.AddTool(server, &mcp.Tool{
	Name:        "getKubernetesNamespacedResourceList",
	Description: "Returns a list of kubernetes resources inside a namespace. Resources supported: Pod, Service, ConfigMap, Secret, Deployment, StatefulSet, DaemonSet, ReplicaSet, Ingress, NetworkPolicy, HorizontalPodAutoscaler, ServiceAccount, Role, RoleBinding, PersistentVolumeClaim, Project, Bundle, GitRepo"},
	tools.GetNamespacedKubernetesResourceList)*/
	mcp.AddTool(server, &mcp.Tool{
		Name: "patchKubernetesNamespacedResource",
		Description: `Description: Patches a namespaced Kubernetes resource using a JSON patch. The JSON patch must be provided as a string.
Parameters:
resource_type (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
namespace (string): The namespace where the resource is located.
resource_name (string): The name of the specific resource to patch.
patch (json): Patch to apply. This must be a JSON object. The content type used is application/json-patch+json.
Returns the modified resource.

Example of the patch parameter:
[{\"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": 3}]`},
		tools.UpdateNamespacedKubernetesResource)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{})

	log.Println("Listening on :9092")
	log.Fatal(http.ListenAndServe(":9092", handler))
}
