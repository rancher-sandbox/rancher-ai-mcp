package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

const (
	tokenHeader      = "R_token"
	urlHeader        = "R_url"
	podLogsTailLines = 50
)

// TODO add missing resources
var k8sKindsToGVRs = map[string]schema.GroupVersionResource{
	"pod":                     {Group: "", Version: "v1", Resource: "pods"},
	"service":                 {Group: "", Version: "v1", Resource: "services"},
	"configmap":               {Group: "", Version: "v1", Resource: "configmaps"},
	"secret":                  {Group: "", Version: "v1", Resource: "secrets"},
	"deployment":              {Group: "apps", Version: "v1", Resource: "deployments"},
	"statefulset":             {Group: "apps", Version: "v1", Resource: "statefulsets"},
	"daemonset":               {Group: "apps", Version: "v1", Resource: "daemonsets"},
	"replicaset":              {Group: "apps", Version: "v1", Resource: "replicasets"},
	"ingress":                 {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	"networkpolicy":           {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
	"horizontalpodautoscaler": {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
	"serviceaccount":          {Group: "", Version: "v1", Resource: "serviceaccounts"},
	"role":                    {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
	"rolebinding":             {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
	"clusterrole":             {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
	"clusterrolebinding":      {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
	"persistentvolume":        {Group: "", Version: "v1", Resource: "persistentvolumes"},
	"persistentvolumeclaim":   {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
	"project":                 {Group: "management.cattle.io", Version: "v3", Resource: "projects"},
	"cluster":                 {Group: "management.cattle.io", Version: "v3", Resource: "clusters"},
	"user":                    {Group: "management.cattle.io", Version: "v3", Resource: "users"},
	"bundle":                  {Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "bundles"},
	"gitrepo":                 {Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "gitrepos"},
}

// ResourceParams uniquely identifies a specific named resource within a cluster.
type ResourceParams struct {
	Name      string `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
}

// GetNodesParams specifies the parameters needed to retrieve node metrics.
type GetNodesParams struct {
	Cluster string `json:"cluster" jsonschema:"the cluster of the resource"`
}

// UpdateKubernetesResourceParams defines the structure for updating a general Kubernetes resource.
// It includes fields required to uniquely identify a resource within a cluster.
type UpdateKubernetesResourceParams struct {
	Name      string        `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string        `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string        `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string        `json:"cluster" jsonschema:"the cluster of the resource"`
	Patch     []interface{} `json:"patch" jsonschema:"the patch of the request"`
}

// CreateKubernetesResourceParams defines the structure for creating a general Kubernetes resource.
type CreateKubernetesResourceParams struct {
	Name      string `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
	Resource  any    `json:"patch" jsonschema:"the patch of the request"`
}

// ListKubernetesResourcesParams specifies the parameters needed to list kubernetes resources.
type ListKubernetesResourcesParams struct {
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
}

// SpecificResourceParams uniquely identifies a resource with a known kind within a cluster.
type SpecificResourceParams struct {
	Name      string `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
}

// ContainerLogs holds logs for multiple containers.
type ContainerLogs struct {
	Logs map[string]string `json:"logs"`
}

// clientCreator defines an interface for creating Kubernetes clients.
type clientCreator interface {
	getResourceInterface(token string, url string, namespace string, gvr schema.GroupVersionResource) (dynamic.ResourceInterface, error)
	createClientSet(token string, url string) (kubernetes.Interface, error)
}

// resourceFetcher defines an interface for fetching Kubernetes resources.
type resourceFetcher interface {
	fetchK8sResource(params fetchParams) (*unstructured.Unstructured, error)
	fetchK8sResources(params fetchParams) ([]*unstructured.Unstructured, error)
}

// Tools contains all tools for the MCP server
type Tools struct {
	fetcher resourceFetcher
	client  clientCreator
}

// NewTools creates and returns a new Tools instance.
func NewTools() *Tools {
	return &Tools{
		fetcher: newSteveFetcher(),
		client:  newClient(),
	}
}

// GetResource retrieves a specific Kubernetes resource based on the provided parameters.
func (t *Tools) GetResource(_ context.Context, toolReq *mcp.CallToolRequest, params ResourceParams) (*mcp.CallToolResult, any, error) {
	resource, err := t.fetcher.fetchK8sResource(fetchParams{
		cluster:   params.Cluster,
		kind:      params.Kind,
		namespace: params.Namespace,
		name:      params.Name,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	mcpResponse, err := createMcpResponse([]*unstructured.Unstructured{resource}, params.Namespace, params.Kind, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// ListKubernetesResources lists Kubernetes resources of a specific kind and namespace.
func (t *Tools) ListKubernetesResources(_ context.Context, toolReq *mcp.CallToolRequest, params ListKubernetesResourcesParams) (*mcp.CallToolResult, any, error) {
	resources, err := t.fetcher.fetchK8sResources(fetchParams{
		cluster:   params.Cluster,
		kind:      params.Kind,
		namespace: params.Namespace,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	mcpResponse, err := createMcpResponse(resources, params.Namespace, params.Kind, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// UpdateKubernetesResource updates a specific Kubernetes resource using a JSON patch.
func (t *Tools) UpdateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params UpdateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	resourceInterface, err := t.client.getResourceInterface(toolReq.Extra.Header.Get(tokenHeader), toolReq.Extra.Header.Get(urlHeader), params.Namespace, k8sKindsToGVRs[strings.ToLower(params.Kind)])
	if err != nil {
		return nil, nil, err
	}

	patchBytes, err := json.Marshal(params.Patch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal patch: %w", err)
	}

	obj, err := resourceInterface.Patch(ctx, params.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to patch resource %s: %w", params.Name, err)
	}

	mcpResponse, err := createMcpResponse([]*unstructured.Unstructured{obj}, params.Namespace, params.Kind, params.Cluster) //string(respWithoutManagedFields)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// CreateKubernetesResource creates a new Kubernetes resource.
func (t *Tools) CreateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params CreateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	resourceInterface, err := t.client.getResourceInterface(toolReq.Extra.Header.Get(tokenHeader), toolReq.Extra.Header.Get(urlHeader), params.Namespace, k8sKindsToGVRs[strings.ToLower(params.Kind)])
	if err != nil {
		return nil, nil, err
	}

	objBytes, err := json.Marshal(params.Resource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal resource: %w", err)
	}

	unstructuredObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(objBytes, unstructuredObj); err != nil {
		return nil, nil, fmt.Errorf("failed to create unstructured object: %w", err)
	}

	obj, err := resourceInterface.Create(ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource %s: %w", params.Name, err)
	}

	mcpResponse, err := createMcpResponse([]*unstructured.Unstructured{obj}, params.Namespace, params.Kind, params.Cluster) //string(respWithoutManagedFields)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// InspectPod retrieves detailed information about a specific pod, its owner, metrics, and logs.
func (t *Tools) InspectPod(ctx context.Context, toolReq *mcp.CallToolRequest, params SpecificResourceParams) (*mcp.CallToolResult, any, error) {
	podResource, err := t.fetcher.fetchK8sResource(fetchParams{
		cluster:   params.Cluster,
		kind:      "pod",
		namespace: params.Namespace,
		name:      params.Name,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	var pod corev1.Pod
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(podResource.Object, &pod); err != nil {
		return nil, nil, fmt.Errorf("failed to convert unstructured object to Pod: %w", err)
	}

	// find the parent of the pod
	var replicaSetName string
	for _, or := range pod.OwnerReferences {
		if or.Kind == "ReplicaSet" {
			replicaSetName = or.Name
			break
		}
	}
	replicaSetResource, err := t.fetcher.fetchK8sResource(fetchParams{
		cluster:   params.Cluster,
		kind:      "replicaset",
		namespace: params.Namespace,
		name:      replicaSetName,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	var replicaSet appsv1.ReplicaSet
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(replicaSetResource.Object, &replicaSetResource); err != nil {
		return nil, nil, fmt.Errorf("failed to convert unstructured object to Pod: %w", err)
	}

	var parentName, parentKind string
	for _, or := range replicaSet.OwnerReferences {
		if or.Kind == "Deployment" {
			parentName = or.Name
			parentKind = or.Kind
			break
		}
		if or.Kind == "StatefulSet" {
			parentName = or.Name
			parentKind = or.Kind
			break
		}
		if or.Kind == "DaemonSet" {
			parentName = or.Name
			parentKind = or.Kind
			break
		}
	}
	parentResource, err := t.fetcher.fetchK8sResource(fetchParams{
		cluster:   params.Cluster,
		kind:      parentKind,
		namespace: params.Namespace,
		name:      parentName,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	podMetrics, err := t.fetcher.fetchK8sResource(fetchParams{
		cluster:   params.Cluster,
		kind:      "metrics.k8s.io.pods",
		namespace: params.Namespace,
		name:      params.Name,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	logs, err := t.getPodLogs(ctx, toolReq.Extra.Header.Get(urlHeader), params.Cluster, toolReq.Extra.Header.Get(tokenHeader), pod)
	if err != nil {
		return nil, nil, err
	}
	logsText := "Logs for all containers: \n" + logs

	mcpResponse, err := createMcpResponse([]*unstructured.Unstructured{podResource, parentResource, podMetrics}, params.Namespace, "pod", params.Cluster, logsText)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// GetDeploymentDetails retrieves details about a deployment and its associated pods.
func (t *Tools) GetDeploymentDetails(_ context.Context, toolReq *mcp.CallToolRequest, params SpecificResourceParams) (*mcp.CallToolResult, any, error) {
	deploymentResource, err := t.fetcher.fetchK8sResource(fetchParams{
		cluster:   params.Cluster,
		kind:      "deployment",
		namespace: params.Namespace,
		name:      params.Name,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	var deployment appsv1.Deployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentResource.Object, &deployment); err != nil {
		return nil, nil, fmt.Errorf("failed to convert unstructured object to Pod: %w", err)
	}

	// find all pods for this deployment
	filter := ""
	for k, v := range deployment.Spec.Selector.MatchLabels {
		filter = filter + "filter=metadata.labels." + k + "=" + v + "&"
	}
	filter = filter[:len(filter)-1]
	pods, err := t.fetcher.fetchK8sResources(fetchParams{
		cluster:   params.Cluster,
		kind:      "deployment",
		namespace: params.Namespace,
		name:      params.Name,
		url:       toolReq.Extra.Header.Get(urlHeader),
		token:     toolReq.Extra.Header.Get(tokenHeader),
		filter:    filter,
	})
	if err != nil {
		return nil, nil, err
	}

	mcpResponse, err := createMcpResponse(append([]*unstructured.Unstructured{deploymentResource}, pods...), params.Namespace, "pod", params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// GetNodes retrieves information and metrics for all nodes in a given cluster.
func (t *Tools) GetNodes(_ context.Context, toolReq *mcp.CallToolRequest, params GetNodesParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/nodes"
	nodeResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/metrics.k8s.io.nodes"
	metricsResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: nodeResp + metricsResp}},
	}, nil, nil
}

func (t *Tools) getPodLogs(ctx context.Context, url string, cluster string, token string, pod corev1.Pod) (string, error) {
	clusterURL := url + "/k8s/clusters/" + cluster
	clientset, err := t.client.createClientSet(token, clusterURL)
	if err != nil {
		return "", fmt.Errorf("failed to create clientset: %w", err)
	}
	logs := ContainerLogs{
		Logs: make(map[string]string),
	}
	for _, container := range pod.Spec.Containers {
		podLogOptions := corev1.PodLogOptions{
			TailLines: ptr.To[int64](podLogsTailLines),
			Container: container.Name,
		}
		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
		podLogs, err := req.Stream(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to open log stream: %v", err)
		}
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			return "", fmt.Errorf("failed to copy log stream to buffer: %v", err)
		}
		logs.Logs[container.Name] = buf.String()
		if err := podLogs.Close(); err != nil {
			return "", fmt.Errorf("failed to close pod logs stream: %v", err)
		}
	}
	jsonData, err := json.Marshal(logs)
	if err != nil {
		return "", fmt.Errorf("error marshalling pod logs: %w", err)
	}

	return string(jsonData), nil
}
