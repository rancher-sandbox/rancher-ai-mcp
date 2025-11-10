package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rancher/dynamiclistener"
	"github.com/rancher/dynamiclistener/server"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/rest"
	"mcp/internal/tools"
)

const (
	skipTLSVerifyEnvVar = "INSECURE_SKIP_TLS"
	tlsName             = "rancher-mcp-server.cattle-ai-agent-system.svc"
	certNamespace       = "cattle-ai-agent-system"
	certName            = "cattle-mcp-tls"
	caName              = "cattle-mcp-ca"
)

func init() {
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	} else {
		config := zap.NewProductionConfig()
		// remove the "caller" key from the log output
		config.EncoderConfig.CallerKey = zapcore.OmitKey
		zap.ReplaceGlobals(zap.Must(config.Build()))
	}
}

func main() {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "pod finder", Version: "v1.0.0"}, nil)
	tools := tools.NewTools()
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getKubernetesResource",
		Description: `Fetches a Kubernetes resource from the cluster.
		Parameters:
		name (string, required): The name of the Kubernetes resource.
		kind (string, required): The kind of the Kubernetes resource (e.g. 'Deployment', 'Service').
		cluster (string): The name of the Kubernetes cluster managed by Rancher.
		namespace (string, optional): The namespace of the resource. It must be empty for all namespaces or cluster-wide resources.
		
		Returns:
		The JSON representation of the requested Kubernetes resource.`},
		tools.GetResource)
	mcp.AddTool(mcpServer, &mcp.Tool{
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
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "listKubernetesResources",
		Description: `Returns a list of kubernetes resources.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource are located. It must be empty for all namespaces or cluster-wide resources.
		cluster (string): The name of the Kubernetes cluster.`},
		tools.ListKubernetesResources)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "inspectPod",
		Description: `Returns all information related to a Pod. It includes its parent Deployment or StatefulSet, the CPU and memory consumption and the logs. It must be used for troubleshooting problems with pods.'
		Parameters:
		namespace (string): The namespace where the resource are located.
		cluster (string): The name of the Kubernetes cluster.
		name (string): The name of the Pod.`},
		tools.InspectPod)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getDeployment",
		Description: `Returns a Deployment and its Pods. It must be used for troubleshooting problems with deployments.'
		Parameters:
		namespace (string): The namespace where the resource are located.
		cluster (string): The name of the Kubernetes cluster.
		name (string): The name of the Deployment.`},
		tools.GetDeploymentDetails)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getNodeMetrics",
		Description: `Returns a list of all nodes in a specified Kubernetes cluster, including their current resource utilization metrics.'
		Parameters:
		cluster (string): The name of the Kubernetes cluster.`},
		tools.GetNodes)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "createKubernetesResource",
		Description: `Creates a resource in a kubernetes cluster.'
		Parameters:
		kind (string): The type of Kubernetes resource to patch (e.g., Pod, Deployment, Service).
		namespace (string): The namespace where the resource is located. It must be empty for cluster-wide resources.
		name (string): The name of the specific resource to patch.
		cluster (string): The name of the Kubernetes cluster. Empty for single container pods.
		resource (json): Resource to be created. This must be a JSON object.`},
		tools.CreateKubernetesResource)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getClusterImages",
		Description: `Returns a list of all container images for the specified clusters.'
		Parameters:
		clusters (array of strings): List of clusters to get images from. Empty for return images for all clusters.`},
		tools.GetClusterImages)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{})

	if os.Getenv(skipTLSVerifyEnvVar) == "true" {
		zap.L().Info("MCP Server started!")
		log.Fatal(http.ListenAndServe(":9092", handler))
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalf("error creating in-cluster config: %v", err)
		}
		factory, err := core.NewFactoryFromConfig(config)
		if err != nil {
			log.Fatalf("error creating factory: %v", err)
		}

		ctx := context.Background()
		err = server.ListenAndServe(ctx, 9092, 0, handler, &server.ListenOpts{
			Secrets:       factory.Core().V1().Secret(),
			CertNamespace: certNamespace,
			CertName:      certName,
			CAName:        caName,
			TLSListenerConfig: dynamiclistener.Config{
				SANs: []string{
					tlsName,
				},
				FilterCN: dynamiclistener.OnlyAllow(tlsName),
			},
		})
		if err != nil {
			log.Fatalf("error creating tls server: %v", err)
		}
		zap.L().Info("MCP Server with TLS started!")
		<-ctx.Done()
	}
}
