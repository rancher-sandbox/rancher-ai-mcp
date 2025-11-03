package k8s

import (
	"context"
	"fmt"
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"mcp/internal/tools/converter"
)

const skipTLSVerifyEnvVar = "INSECURE_SKIP_TLS"

var clusterIdsCache = sync.Map{}
var clustersDisplayNameToIDCache = sync.Map{}

type resourceInterface interface {
	GetResourceInterface(token string, url string, namespace string, cluster string, gvr schema.GroupVersionResource) (dynamic.ResourceInterface, error)
}

// Client is a struct that provides methods for interacting with Kubernetes clusters.
type Client struct{}

// NewClient creates and returns a new instance of the Client struct.
func NewClient() *Client {
	return &Client{}
}

// CreateClientSet creates a new Kubernetes clientset for the given Token and URL.
func (c *Client) CreateClientSet(token string, url string, cluster string) (kubernetes.Interface, error) {
	clusterID, err := getClusterId(c, token, url, cluster)
	if err != nil {
		return nil, err
	}
	restConfig, err := createRestConfig(token, url, clusterID)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

// GetResourceInterface returns a dynamic resource interface for the given Token, URL, Namespace, and GroupVersionResource.
func (c *Client) GetResourceInterface(token string, url string, namespace string, cluster string, gvr schema.GroupVersionResource) (dynamic.ResourceInterface, error) {
	clusterID, err := getClusterId(c, token, url, cluster)
	if err != nil {
		return nil, err
	}
	restConfig, err := createRestConfig(token, url, clusterID)
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

// createRestConfig creates a new rest.Config for the given Token and URL.
func createRestConfig(token string, url string, clusterID string) (*rest.Config, error) {
	clusterURL := url + "/k8s/clusters/" + clusterID
	kubeconfig := clientcmdapi.NewConfig()
	kubeconfig.Clusters["Cluster"] = &clientcmdapi.Cluster{
		Server:                clusterURL,
		InsecureSkipTLSVerify: os.Getenv(skipTLSVerifyEnvVar) == "true",
	}
	kubeconfig.AuthInfos["mcp"] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	kubeconfig.Contexts["Cluster"] = &clientcmdapi.Context{
		Cluster:  "Cluster",
		AuthInfo: "mcp",
	}
	kubeconfig.CurrentContext = "Cluster"
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

// getClusterId returns the cluster's unique ID given either its cluster ID (metadata.name)
// or its display name (spec.displayName). It uses local caches to avoid redundant lookups.
//
// The lookup order is:
//  1. If the input is "local", return immediately.
//  2. Check in-memory caches for cluster ID or display name.
//  3. Query the cluster resource API by ID.
//  4. If not found, fall back to listing all clusters and matching by display name.
//
// both cluster ID and display name are cached for future lookups.
func getClusterId(c resourceInterface, token string, url string, clusterNameOrID string) (string, error) {
	// handle the special case for the local cluster, it always exists and is known by ID and displayName "local"
	if clusterNameOrID == "local" {
		return "local", nil
	}

	// check if the provided identifier is already known to be a cluster ID
	if _, ok := clusterIdsCache.Load(clusterNameOrID); ok {
		return clusterNameOrID, nil // it is a cluster ID
	}

	// check if the provided identifier matches a display name cached earlier
	if clusterID, exists := clustersDisplayNameToIDCache.Load(clusterNameOrID); exists {
		return clusterID.(string), nil
	}

	// try to fetch the cluster directly by its ID
	clusterInterface, err := c.GetResourceInterface(token, url, "", "local", converter.K8sKindsToGVRs["cluster"])
	if err != nil {
		return "", err
	}

	cluster, err := clusterInterface.Get(context.Background(), clusterNameOrID, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return "", err
		}

		// If not found by ID, try to locate it by display name.
		clusters, err := clusterInterface.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return "", err
		}
		for _, cluster := range clusters.Items {
			clusterID := cluster.GetName()
			clusterIdsCache.Store(clusterID, struct{}{})

			displayName, found, err := unstructured.NestedString(
				cluster.Object,
				"spec",
				"displayName",
			)
			if err != nil {
				return "", err
			}

			if found {
				clustersDisplayNameToIDCache.Store(displayName, clusterID)

				// If the given identifier matches this display name, return its ID.
				if displayName == clusterNameOrID {
					return clusterID, nil
				}
			}
		}

		return "", fmt.Errorf("cluster '%s' not found", clusterNameOrID)
	}

	// clusterNameOrIDInput contains the cluster ID. Store it in the cache.
	clusterID := clusterNameOrID
	clusterIdsCache.Store(clusterID, struct{}{})

	displayName, found, err := unstructured.NestedString(
		cluster.Object,
		"spec",
		"displayName",
	)
	if err != nil {
		return "", err
	}
	if found {
		clustersDisplayNameToIDCache.Store(displayName, clusterID)
	}

	return clusterID, nil
}
