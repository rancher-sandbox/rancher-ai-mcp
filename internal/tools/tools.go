package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	v1 "k8s.io/api/rbac/v1"
	"mcp/internal/tools/converter"
	"mcp/internal/tools/k8s"
	"mcp/internal/tools/response"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8srand "k8s.io/apimachinery/pkg/util/rand"
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

type CreateResource struct {
	Name      string `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
	Resource  any    `json:"resource" jsonschema:"the resource to be created"`
}

// CreateKubernetesResourceParams defines the structure for creating a general Kubernetes resource.
type CreateKubernetesResourceParams struct {
	Resources []CreateResource `json:"resources" jsonschema:"resources to be created"`
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

type GrantPermissionsParams struct {
	Namespace string          `json:"namespace" jsonschema:"the namespace of the resource"`
	Cluster   string          `json:"cluster" jsonschema:"the cluster of the resource"`
	User      string          `json:"user" jsonschema:"the user to check permissions for"`
	Group     string          `json:"group" jsonschema:"the user to check permissions for"`
	Grant     bool            `json:"grant" jsonschema:"true if the user is asking to grant permissions"`
	Project   string          `json:"project" jsonschema:"the project where permissions are grant"`
	Rules     []v1.PolicyRule `json:"rules" jsonschema:"kubernetes rules"`
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

	mcpResponse, err := response.CreateMcpResponse([]*unstructured.Unstructured{resource}, params.Cluster)
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

	mcpResponse, err := response.CreateMcpResponse(resources, params.Cluster)
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

	mcpResponse, err := response.CreateMcpResponse([]*unstructured.Unstructured{obj}, params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// CreateKubernetesResource creates a new Kubernetes resource.
func (t *Tools) CreateKubernetesResources(ctx context.Context, toolReq *mcp.CallToolRequest, params CreateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	var objs []*unstructured.Unstructured

	for _, param := range params.Resources {

		var objBytes []byte

		switch v := param.Resource.(type) {
		case string:
			// params.Resource is already a JSON string
			objBytes = []byte(v)
		default:
			// params.Resource is a structured object
			var err error
			objBytes, err = json.Marshal(v)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal resource: %w", err)
			}
		}

		unstructuredObj := &unstructured.Unstructured{}
		if err := json.Unmarshal(objBytes, unstructuredObj); err != nil {
			return nil, nil, fmt.Errorf("failed to create unstructured object: %w", err)
		}

		namespace := param.Namespace
		if namespace == "" {
			namespace = unstructuredObj.GetNamespace()
		}

		resourceInterface, err := t.client.GetResourceInterface(toolReq.Extra.Header.Get(tokenHeader), toolReq.Extra.Header.Get(urlHeader), namespace, param.Cluster, converter.K8sKindsToGVRs[strings.ToLower(param.Kind)])
		if err != nil {
			return nil, nil, err
		}

		obj, err := resourceInterface.Create(ctx, unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create resource %s: %w", param.Name, err)
		}
		objs = append(objs, obj)
	}

	mcpResponse, err := response.CreateMcpResponse(objs, params.Resources[0].Cluster) // TODO separate cluster
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

	mcpResponse, err := response.CreateMcpResponse([]*unstructured.Unstructured{podResource, parentResource, podMetrics, logs}, params.Cluster)
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

	mcpResponse, err := response.CreateMcpResponse(append([]*unstructured.Unstructured{deploymentResource}, pods...), params.Cluster)
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

	mcpResponse, err := response.CreateMcpResponse(append(nodeResource, nodeMetricsResource...), params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

type ToolCallRequest struct {
	Name string `json:"name"`
	Args any    `json:"args"`
	Id   string `json:"id"`
}
type NestedToolCallResponse struct {
	ToolCallRequests ToolCallRequest `json:"toolCallsRequest"`
	AdditionalText   string          `json:"additionalText"`
	ConfirmText      string          `json:"confirmText"`
}

// GrantOrVerifyResourceAccess checks if the user or group has permission to access a resource.
func (t *Tools) GrantOrVerifyResourceAccess(ctx context.Context, toolReq *mcp.CallToolRequest, params GrantPermissionsParams) (*mcp.CallToolResult, any, error) {
	if params.User == "" && params.Group == "" {
		return nil, nil, fmt.Errorf("user or group must be provided")
	}
	clientset, err := t.client.CreateClientSet(toolReq.Extra.Header.Get(tokenHeader), toolReq.Extra.Header.Get(urlHeader), params.Cluster)
	if err != nil {
		return nil, nil, err
	}

	allowed := true
	for _, rule := range params.Rules {
		verbs := rule.Verbs
		if len(verbs) == 0 {
			verbs = []string{""}
		}

		resources := rule.Resources
		if len(resources) == 0 {
			resources = []string{""}
		}

		apiGroups := rule.APIGroups
		if len(apiGroups) == 0 {
			apiGroups = []string{""}
		}

		for _, verb := range verbs {
			for _, resource := range resources {
				for _, group := range apiGroups {
					// TODO not working for Projects!
					// TODO check in downstream clusters
					sar := &authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: params.Namespace,
								Verb:      verb,
								Group:     group,
								Resource:  resource,
							},
						},
					}
					if params.User != "" {
						sar.Spec.User = params.User
					}
					if params.Group != "" {
						sar.Spec.Groups = []string{params.Group}
					}
					resp, err := clientset.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
					if err != nil {
						return nil, nil, err
					}
					if !resp.Status.Allowed {
						allowed = false // TODO add missing permissions!
					}
				}
			}
		}
	}

	var response string
	if allowed {
		response = fmt.Sprintf("the user %s has permission to access the resource", params.User)
	} else {
		rtName := generateRandomName("rt")
		rt := v3.RoleTemplate{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleTemplate",
				APIVersion: "management.cattle.io/v3",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: rtName,
			},
		}
		if params.Project != "" {
			rt.Context = "project"
		} else {
			rt.Context = "cluster"
		}
		var rules []v1.PolicyRule
		for _, rule := range params.Rules {
			verbs := rule.Verbs
			if len(verbs) == 0 {
				verbs = []string{""}
			}

			resources := rule.Resources
			if len(resources) == 0 {
				resources = []string{""}
			}

			apiGroups := rule.APIGroups
			if len(apiGroups) == 0 {
				apiGroups = []string{""}
			}
			rules = append(rules, v1.PolicyRule{
				Verbs:     verbs,
				APIGroups: apiGroups,
				Resources: resources,
			})
		}
		rt.Rules = rules
		rtBytes, err := json.Marshal(rt)
		if err != nil {
			return nil, nil, err
		}
		var bindingResource CreateResource
		if params.Project != "" {
			prtb := v3.ProjectRoleTemplateBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProjectRoleTemplateBinding",
					APIVersion: "management.cattle.io/v3",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      generateRandomName("prtb"),
					Namespace: params.Cluster + "-" + params.Project,
				},
				ProjectName:      params.Cluster + ":" + params.Project,
				RoleTemplateName: rtName,
			}
			if params.Group != "" {
				prtb.GroupName = params.Group
			} else if params.User != "" {
				prtb.UserName = params.User
			}

			prtbBytes, err := json.Marshal(prtb)
			if err != nil {
				return nil, nil, err
			}
			bindingResource = CreateResource{
				Name:      prtb.Name,
				Namespace: params.Cluster + "-" + params.Project,
				Kind:      "ProjectRoleTemplateBinding",
				Cluster:   "local",
				Resource:  string(prtbBytes),
			}
		} else {
			crtb := v3.ClusterRoleTemplateBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleTemplateBinding",
					APIVersion: "management.cattle.io/v3",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      generateRandomName("crtb"),
					Namespace: params.Cluster,
				},
				ClusterName:      params.Cluster,
				RoleTemplateName: rtName,
			}
			if params.Group != "" {
				crtb.GroupName = params.Group
			} else if params.User != "" {
				crtb.UserName = params.User
			}

			crtbBytes, err := json.Marshal(crtb)
			if err != nil {
				return nil, nil, err
			}
			bindingResource = CreateResource{
				Name:      crtb.Name,
				Namespace: params.Cluster,
				Kind:      "ClusterRoleTemplateBinding",
				Cluster:   "local",
				Resource:  string(crtbBytes),
			}
		}

		call := &NestedToolCallResponse{
			ToolCallRequests: ToolCallRequest{
				Name: "createKubernetesResources",
				Args: CreateKubernetesResourceParams{
					Resources: []CreateResource{
						{
							Name:      rt.Name,
							Namespace: "",
							Kind:      "RoleTemplate",
							Cluster:   "local",
							Resource:  string(rtBytes),
						},
						bindingResource,
					},
				},
				Id: generateToolID(),
			},
			AdditionalText: "You can create RT and CTRB from the UI",
			ConfirmText:    "I can create the reouscer. Would you like me to create the necessary resources?",
		}
		bytes, err := json.Marshal(call)
		if err != nil {
			return nil, nil, err
		}
		response = string(bytes)

	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: response}},
	}, nil, nil
}
func generateToolID() string {
	// Generate 4 random bytes → 8 hex characters
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	return fmt.Sprintf("tool-%s", hex.EncodeToString(bytes))
}

func generateRandomName(prefix string) string {
	// rand.String(5) generates a random 5-character string.
	return fmt.Sprintf("%s-%s", prefix, k8srand.String(10))
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
