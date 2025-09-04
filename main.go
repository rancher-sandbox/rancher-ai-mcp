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
		Name:        "getKubernetesResource",
		Description: "Returns non-namespaced kubernetes resource. Resources supported: User, Cluster, ClusterRole, ClusterRoleBinding, PersistentVolume"},
		tools.GetNonNamespacedKubernetesResource)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getKubernetesNamespacedResource",
		Description: "Returns namespaced kubernetes resource. Resources supported: Pod, Service, ConfigMap, Secret, Deployment, StatefulSet, DaemonSet, ReplicaSet, Ingress, NetworkPolicy, HorizontalPodAutoscaler, ServiceAccount, Role, RoleBinding, PersistentVolumeClaim, Project, Bundle, GitRepo"},
		tools.GetNamespacedKubernetesResource)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getKubernetesNamespacedResourceList",
		Description: "Returns a list of kubernetes resources inside a namespace. Resources supported: Pod, Service, ConfigMap, Secret, Deployment, StatefulSet, DaemonSet, ReplicaSet, Ingress, NetworkPolicy, HorizontalPodAutoscaler, ServiceAccount, Role, RoleBinding, PersistentVolumeClaim, Project, Bundle, GitRepo"},
		tools.GetNamespacedKubernetesResourceList)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{})

	log.Println("Listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", handler))
}
