package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mcp/internal/tools/converter"
	"mcp/internal/tools/k8s"
	"mcp/internal/tools/response"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
	Resource  any    `json:"resource" jsonschema:"the resource to be created"`
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
	Logs map[string]any `json:"logs"`
}

// K8sClient defines an interface for a Kubernetes client.
type K8sClient interface {
	GetResourceInterface(token string, url string, namespace string, cluster string, gvr schema.GroupVersionResource) (dynamic.ResourceInterface, error)
	CreateClientSet(token string, url string, cluster string) (kubernetes.Interface, error)
	GetResource(ctx context.Context, params k8s.GetParams) (*unstructured.Unstructured, error)
	GetResources(ctx context.Context, params k8s.ListParams) ([]*unstructured.Unstructured, error)
}

// Tools contains all tools for the MCP server
type Tools struct {
	client K8sClient
}

// NewTools creates and returns a new Tools instance.
func NewTools() *Tools {
	return &Tools{
		client: k8s.NewClient(),
	}
}

// GetResource retrieves a specific Kubernetes resource based on the provided parameters.
func (t *Tools) GetResource(ctx context.Context, toolReq *mcp.CallToolRequest, params ResourceParams) (*mcp.CallToolResult, any, error) {
	resource, err := t.client.GetResource(ctx, k8s.GetParams{
		Cluster:   params.Cluster,
		Kind:      params.Kind,
		Namespace: params.Namespace,
		Name:      params.Name,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	mcpResponse, err := response.CreateMcpResponse([]*unstructured.Unstructured{resource}, params.Namespace, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// ListKubernetesResources lists Kubernetes resources of a specific kind and namespace.
func (t *Tools) ListKubernetesResources(ctx context.Context, toolReq *mcp.CallToolRequest, params ListKubernetesResourcesParams) (*mcp.CallToolResult, any, error) {
	resources, err := t.client.GetResources(ctx, k8s.ListParams{
		Cluster:   params.Cluster,
		Kind:      params.Kind,
		Namespace: params.Namespace,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	mcpResponse, err := response.CreateMcpResponse(resources, params.Namespace, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// UpdateKubernetesResource updates a specific Kubernetes resource using a JSON patch.
func (t *Tools) UpdateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params UpdateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	resourceInterface, err := t.client.GetResourceInterface(toolReq.Extra.Header.Get(tokenHeader), toolReq.Extra.Header.Get(urlHeader), params.Namespace, params.Cluster, converter.K8sKindsToGVRs[strings.ToLower(params.Kind)])
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

	mcpResponse, err := response.CreateMcpResponse([]*unstructured.Unstructured{obj}, params.Namespace, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// CreateKubernetesResource creates a new Kubernetes resource.
func (t *Tools) CreateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params CreateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	resourceInterface, err := t.client.GetResourceInterface(toolReq.Extra.Header.Get(tokenHeader), toolReq.Extra.Header.Get(urlHeader), params.Namespace, params.Cluster, converter.K8sKindsToGVRs[strings.ToLower(params.Kind)])
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

	mcpResponse, err := response.CreateMcpResponse([]*unstructured.Unstructured{obj}, params.Namespace, params.Cluster) //string(respWithoutManagedFields)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// InspectPod retrieves detailed information about a specific pod, its owner, metrics, and logs.
func (t *Tools) InspectPod(ctx context.Context, toolReq *mcp.CallToolRequest, params SpecificResourceParams) (*mcp.CallToolResult, any, error) {
	podResource, err := t.client.GetResource(ctx, k8s.GetParams{
		Cluster:   params.Cluster,
		Kind:      "pod",
		Namespace: params.Namespace,
		Name:      params.Name,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
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
	replicaSetResource, err := t.client.GetResource(ctx, k8s.GetParams{
		Cluster:   params.Cluster,
		Kind:      "replicaset",
		Namespace: params.Namespace,
		Name:      replicaSetName,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	var replicaSet appsv1.ReplicaSet
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(replicaSetResource.Object, &replicaSet); err != nil {
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
	parentResource, err := t.client.GetResource(ctx, k8s.GetParams{
		Cluster:   params.Cluster,
		Kind:      parentKind,
		Namespace: params.Namespace,
		Name:      parentName,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	// ignore error as Metrics Server might not be installed in the cluster
	podMetrics, _ := t.client.GetResource(ctx, k8s.GetParams{
		Cluster:   params.Cluster,
		Kind:      "pod.metrics.k8s.io",
		Namespace: params.Namespace,
		Name:      params.Name,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})

	logs, err := t.getPodLogs(ctx, toolReq.Extra.Header.Get(urlHeader), params.Cluster, toolReq.Extra.Header.Get(tokenHeader), pod)
	if err != nil {
		return nil, nil, err
	}

	mcpResponse, err := response.CreateMcpResponse([]*unstructured.Unstructured{podResource, parentResource, podMetrics, logs}, params.Namespace, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// GetDeploymentDetails retrieves details about a deployment and its associated pods.
func (t *Tools) GetDeploymentDetails(ctx context.Context, toolReq *mcp.CallToolRequest, params SpecificResourceParams) (*mcp.CallToolResult, any, error) {
	deploymentResource, err := t.client.GetResource(ctx, k8s.GetParams{
		Cluster:   params.Cluster,
		Kind:      "deployment",
		Namespace: params.Namespace,
		Name:      params.Name,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	var deployment appsv1.Deployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentResource.Object, &deployment); err != nil {
		return nil, nil, fmt.Errorf("failed to convert unstructured object to Pod: %w", err)
	}

	// find all pods for this deployment
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert label selector: %w", err)
	}
	pods, err := t.client.GetResources(ctx, k8s.ListParams{
		Cluster:       params.Cluster,
		Kind:          "pod",
		Namespace:     params.Namespace,
		Name:          params.Name,
		URL:           toolReq.Extra.Header.Get(urlHeader),
		Token:         toolReq.Extra.Header.Get(tokenHeader),
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pods: %w", err)
	}

	mcpResponse, err := response.CreateMcpResponse(append([]*unstructured.Unstructured{deploymentResource}, pods...), params.Namespace, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// GetNodes retrieves information and metrics for all nodes in a given cluster.
func (t *Tools) GetNodes(ctx context.Context, toolReq *mcp.CallToolRequest, params GetNodesParams) (*mcp.CallToolResult, any, error) {
	nodeResource, err := t.client.GetResources(ctx, k8s.ListParams{
		Cluster: params.Cluster,
		Kind:    "node",
		URL:     toolReq.Extra.Header.Get(urlHeader),
		Token:   toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		return nil, nil, err
	}

	// ignore error as Metrics Server might not be installed in the cluster
	nodeMetricsResource, _ := t.client.GetResources(ctx, k8s.ListParams{
		Cluster: params.Cluster,
		Kind:    "node.metrics.k8s.io",
		URL:     toolReq.Extra.Header.Get(urlHeader),
		Token:   toolReq.Extra.Header.Get(tokenHeader),
	})

	mcpResponse, err := response.CreateMcpResponse(append(nodeResource, nodeMetricsResource...), "", params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

func (t *Tools) getPodLogs(ctx context.Context, url string, cluster string, token string, pod corev1.Pod) (*unstructured.Unstructured, error) {
	clientset, err := t.client.CreateClientSet(token, url, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}
	logs := ContainerLogs{
		Logs: make(map[string]any),
	}
	for _, container := range pod.Spec.Containers {
		podLogOptions := corev1.PodLogOptions{
			TailLines: ptr.To[int64](podLogsTailLines),
			Container: container.Name,
		}
		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
		podLogs, err := req.Stream(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to open log stream: %v", err)
		}
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			return nil, fmt.Errorf("failed to copy log stream to buffer: %v", err)
		}
		logs.Logs[container.Name] = buf.String()
		if err := podLogs.Close(); err != nil {
			return nil, fmt.Errorf("failed to close pod logs stream: %v", err)
		}
	}

	return &unstructured.Unstructured{Object: map[string]interface{}{"pod-logs": logs.Logs}}, nil
}
