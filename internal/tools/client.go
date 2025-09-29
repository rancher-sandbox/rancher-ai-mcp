package tools

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// client is a struct that provides methods for interacting with Kubernetes clusters.
type client struct{}

// newClient creates and returns a new instance of the client struct.
func newClient() *client {
	return &client{}
}

// createClientSet creates a new Kubernetes clientset for the given token and URL.
func (c *client) createClientSet(token string, url string) (kubernetes.Interface, error) {
	restConfig, err := createRestConfig(token, url)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

// getResourceInterface returns a dynamic resource interface for the given token, URL, namespace, and GroupVersionResource.
func (c *client) getResourceInterface(token string, url string, namespace string, gvr schema.GroupVersionResource) (dynamic.ResourceInterface, error) {
	restConfig, err := createRestConfig(token, url)
	if err != nil {
		return nil, err
	}
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	var resourceInterface dynamic.ResourceInterface = dynClient.Resource(gvr)
	if namespace != "" {
		resourceInterface = dynClient.Resource(gvr).Namespace(namespace)
	}

	return resourceInterface, nil
}

// createRestConfig creates a new rest.Config for the given token and URL.
func createRestConfig(token string, url string) (*rest.Config, error) {
	kubeconfig := clientcmdapi.NewConfig()
	kubeconfig.Clusters["cluster"] = &clientcmdapi.Cluster{
		Server: url,
	}
	kubeconfig.AuthInfos["mcp"] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	kubeconfig.Contexts["cluster"] = &clientcmdapi.Context{
		Cluster:  "cluster",
		AuthInfo: "mcp",
	}
	kubeconfig.CurrentContext = "cluster"
	restConfig, err := clientcmd.NewNonInteractiveClientConfig(
		*kubeconfig,
		kubeconfig.CurrentContext,
		&clientcmd.ConfigOverrides{},
		nil,
	).ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}
