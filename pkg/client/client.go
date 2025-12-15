package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"mcp/pkg/converter"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const skipTLSVerifyEnvVar = "INSECURE_SKIP_TLS"

var clusterIdsCache = sync.Map{}
var clustersDisplayNameToIDCache = sync.Map{}

type resourceInterface interface {
	GetResourceInterface(token string, url string, namespace string, cluster string, gvr schema.GroupVersionResource) (dynamic.ResourceInterface, error)
}

// K8sClient defines an interface for a Kubernetes client.
type K8sClient interface {
	GetResourceInterface(token string, url string, namespace string, cluster string, gvr schema.GroupVersionResource) (dynamic.ResourceInterface, error)
	CreateClientSet(token string, url string, cluster string) (kubernetes.Interface, error)
	GetResource(ctx context.Context, params GetParams) (*unstructured.Unstructured, error)
	GetResources(ctx context.Context, params ListParams) ([]*unstructured.Unstructured, error)
}

// Client is a struct that provides methods for interacting with Kubernetes clusters.
type Client struct {
	DynClientCreator func(*rest.Config) (dynamic.Interface, error)
	ClientSetCreator func(*rest.Config) (kubernetes.Interface, error)
}

// GetParams holds the parameters required to get a resource from k8s.
type GetParams struct {
	Cluster   string // The Cluster ID.
	Kind      string // The Kind of the Kubernetes resource (e.g., "pod", "deployment").
	Namespace string // The Namespace of the resource (optional).
	Name      string // The Name of the resource (optional).
	URL       string // The base URL of the Rancher server.
	Token     string // The authentication Token for Steve.
}

// ListParams holds the parameters required to list resources from k8s.
type ListParams struct {
	Cluster       string // The Cluster ID.
	Kind          string // The Kind of the Kubernetes resource (e.g., "pod", "deployment").
	Namespace     string // The Namespace of the resource (optional).
	Name          string // The Name of the resource (optional).
	URL           string // The base URL of the Rancher server.
	Token         string // The authentication Token for Steve.
	LabelSelector string // Optional LabelSelector string for the request.
}

// NewClient creates and returns a new instance of the Client struct.
func NewClient() *Client {
	return &Client{
		DynClientCreator: func(cfg *rest.Config) (dynamic.Interface, error) {
			return dynamic.NewForConfig(cfg)
		},
		ClientSetCreator: func(cfg *rest.Config) (kubernetes.Interface, error) {
			return kubernetes.NewForConfig(cfg)
		},
	}
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

	return c.ClientSetCreator(restConfig)
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
	dynClient, err := c.DynClientCreator(restConfig)
	if err != nil {
		return nil, err
	}
	var resourceInterface dynamic.ResourceInterface = dynClient.Resource(gvr)
	if namespace != "" {
		resourceInterface = dynClient.Resource(gvr).Namespace(namespace)
	}

	return resourceInterface, nil
}

func (c *Client) GetResource(ctx context.Context, params GetParams) (*unstructured.Unstructured, error) {
	resourceInterface, err := c.GetResourceInterface(params.Token, params.URL, params.Namespace, params.Cluster, converter.K8sKindsToGVRs[strings.ToLower(params.Kind)])
	if err != nil {
		return nil, err
	}

	obj, err := resourceInterface.Get(ctx, params.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, err
}

func (c *Client) GetResources(ctx context.Context, params ListParams) ([]*unstructured.Unstructured, error) {
	resourceInterface, err := c.GetResourceInterface(params.Token, params.URL, params.Namespace, params.Cluster, converter.K8sKindsToGVRs[strings.ToLower(params.Kind)])
	if err != nil {
		return nil, err
	}

	opts := metav1.ListOptions{}
	if params.LabelSelector != "" {
		opts.LabelSelector = params.LabelSelector
	}
	list, err := resourceInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	objs := make([]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		objs[i] = &list.Items[i]
	}

	return objs, err
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
