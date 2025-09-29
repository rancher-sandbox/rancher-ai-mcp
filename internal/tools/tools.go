package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
)

const (
	steveEndpointVersion = "v1"
	tokenHeader          = "R_token"
	urlHeader            = "R_url"
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
type UIContext struct {
	Namespace string   `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string   `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string   `json:"cluster" jsonschema:"the cluster of the resource"`
	Name      []string `json:"name" jsonschema:"the name of k8s resource"`
}

// MCPResponse represents the response returned by the MCP server
type MCPResponse struct {
	// LLM response to be sent to the LLM
	LLM string `json:"llm"`
	// UIContext contains a list of resources so the UI can generate links to them
	UIContext UIContext `json:"uiContext,omitempty"`
}

type ListRoot struct {
	Data []Item `json:"data"`
}

type Item struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
}

func createMcpResponse(llmText string, namespace string, kind string, cluster string, name []string) (string, error) {
	resp := MCPResponse{
		LLM: llmText,
		UIContext: UIContext{
			Namespace: namespace,
			Kind:      kind,
			Cluster:   cluster,
			Name:      name,
		},
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(bytes), nil
}

type Tools struct {
	createDynamicClientFunc func(token string, url string) (dynamic.Interface, error)
	createClientSetFunc     func(token string, url string) (kubernetes.Interface, error)
}

func NewTools() *Tools {
	return &Tools{
		createDynamicClientFunc: createDynamicClient,
		createClientSetFunc:     createClientSet,
	}
}

func (t *Tools) GetResource(_ context.Context, toolReq *mcp.CallToolRequest, params ResourceParams) (*mcp.CallToolResult, any, error) {
	resource, err := t.fetchK8sResource(toolReq, params.Cluster, params.Kind, params.Namespace, params.Name)
	if err != nil {
		return nil, nil, err
	}
	mcpResponse, err := createMcpResponse(resource, params.Namespace, params.Kind, params.Cluster, []string{params.Name})
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// TODO remove manageFields!
func (t *Tools) ListKubernetesResources(_ context.Context, toolReq *mcp.CallToolRequest, params ListKubernetesResourcesParams) (*mcp.CallToolResult, any, error) {
	resources, err := t.fetchK8sResource(toolReq, params.Cluster, params.Kind, params.Namespace, "")
	if err != nil {
		return nil, nil, err
	}
	// extract names from list
	var root ListRoot
	if err := json.Unmarshal([]byte(resources), &root); err != nil {
		return nil, nil, fmt.Errorf("error parsing response : %w", err)
	}
	var names []string
	for _, item := range root.Data {
		names = append(names, item.Metadata.Name)
	}
	mcpResponse, err := createMcpResponse(resources, params.Namespace, params.Kind, params.Cluster, names)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

func (t *Tools) UpdateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params UpdateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	dynClient, err := t.getDynamicClientForCluster(toolReq, params.Cluster)
	if err != nil {
		return nil, nil, err
	}
	patchBytes, err := json.Marshal(params.Patch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal patch: %w", err)
	}
	gvr := k8sKindsToGVRs[strings.ToLower(params.Kind)]
	var resourceInterface dynamic.ResourceInterface = dynClient.Resource(gvr)
	if params.Namespace != "" {
		resourceInterface = dynClient.Resource(gvr).Namespace(params.Namespace)
	}

	obj, err := resourceInterface.Patch(ctx, params.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to patch resource %s: %w", params.Name, err)
	}
	result, err := t.formatK8sObjectResponse(obj, ResourceParams{
		Name:      params.Name,
		Namespace: params.Namespace,
		Kind:      params.Kind,
		Cluster:   params.Cluster,
	})

	return result, nil, err
}

func (t *Tools) CreateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params CreateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	dynClient, err := t.getDynamicClientForCluster(toolReq, params.Cluster)
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
	gvr := k8sKindsToGVRs[strings.ToLower(params.Kind)]
	var resourceInterface dynamic.ResourceInterface = dynClient.Resource(gvr)
	if params.Namespace != "" {
		resourceInterface = dynClient.Resource(gvr).Namespace(params.Namespace)
	}

	obj, err := resourceInterface.Create(ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource %s: %w", params.Name, err)
	}
	result, err := t.formatK8sObjectResponse(obj, ResourceParams{
		Name:      params.Name,
		Namespace: params.Namespace,
		Kind:      params.Kind,
		Cluster:   params.Cluster,
	})

	return result, nil, err
}

func (t *Tools) InspectPod(ctx context.Context, toolReq *mcp.CallToolRequest, params ResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/pods/" + params.Namespace + "/" + params.Name
	podResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	var pod corev1.Pod
	if err := json.Unmarshal([]byte(podResp), &pod); err != nil {
		return nil, nil, fmt.Errorf("error parsing pod response : %w", err)
	}
	var replicaSetName string
	for _, or := range pod.OwnerReferences {
		if or.Kind == "ReplicaSet" {
			replicaSetName = or.Name
			break
		}
	}
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/apps.replicasets/" + params.Namespace + "/" + replicaSetName
	replicaSetResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	var replicaSet appsv1.ReplicaSet
	if err := json.Unmarshal([]byte(replicaSetResp), &replicaSet); err != nil {
		return nil, nil, fmt.Errorf("error parsing replicaset response : %w", err)
	}
	// TODO check for replicaset or demonset
	var deploymentName string
	for _, or := range replicaSet.OwnerReferences {
		if or.Kind == "Deployment" {
			deploymentName = or.Name
			break
		}
	}
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/apps.deployments/" + params.Namespace + "/" + deploymentName
	deploymentResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}

	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/metrics.k8s.io.pods/" + params.Namespace + "/" + params.Name
	metricResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	logs, err := t.getPodLogs(ctx, rancherURL, params.Cluster, toolReq.Extra.Header.Get(tokenHeader), pod)
	if err != nil {
		return nil, nil, err
	}

	response := "Pod = " + podResp + "\nDeployment = " + deploymentResp + "\nMetrics = " + metricResp + "\nLogs = " + logs

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: response}},
	}, nil, nil
}

func (t *Tools) GetDeploymentDetails(_ context.Context, toolReq *mcp.CallToolRequest, params ResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/apps.deployments/" + params.Namespace + "/" + params.Name
	deploymentResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	var deployment appsv1.Deployment
	if err := json.Unmarshal([]byte(deploymentResp), &deployment); err != nil {
		return nil, nil, fmt.Errorf("error parsing pod response : %w", err)
	}
	filter := ""
	for k, v := range deployment.Spec.Selector.MatchLabels {
		filter = filter + "filter=metadata.labels." + k + "=" + v + "&"
	}
	filter = filter[:len(filter)-1]
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpointVersion + "/pods/" + params.Namespace + "?" + filter
	podResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: deploymentResp + podResp}},
	}, nil, nil
}

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

type ContainerLogs struct {
	Logs map[string]string `json:"logs"`
}

// fetchK8sResource is a helper method that constructs the request URL and fetches data
// from the Steve API endpoint. If a resource name is provided, it fetches a single
// resource; otherwise, it fetches a list.
func (t *Tools) fetchK8sResource(toolReq *mcp.CallToolRequest, cluster, kind, namespace, name string) (string, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	token := toolReq.Extra.Header.Get(tokenHeader)
	lowerKind := strings.ToLower(kind)
	reqURL := fmt.Sprintf("%s/k8s/clusters/%s/%s", rancherURL, cluster, steveEndpointVersion)
	resourcePath := lowerKind
	if gvr, ok := k8sKindsToGVRs[lowerKind]; ok && gvr.Group != "" {
		resourcePath = gvr.Group + "." + lowerKind
	}
	reqURL = reqURL + "/" + resourcePath
	if namespace != "" {
		reqURL = reqURL + "/" + namespace
	}
	if name != "" {
		reqURL = reqURL + "/" + name
	}
	resp, err := doRequest(reqURL, token)
	if err != nil {
		return "", fmt.Errorf("failed API request to %s: %w", reqURL, err)
	}

	return resp, nil
}

// formatK8sObjectResponse takes a Kubernetes object, marshals it, removes managed fields,
// and wraps it in the final mcp.CallToolResult.
func (t *Tools) formatK8sObjectResponse(obj *unstructured.Unstructured, resource ResourceParams) (*mcp.CallToolResult, error) {
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response object: %w", err)
	}
	respWithoutManagedFields, err := removeManagedFieldsIfPresent(objBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to remove managedFields: %w", err)
	}
	mcpResponse, err := createMcpResponse(string(respWithoutManagedFields), resource.Namespace, resource.Kind, resource.Cluster, []string{resource.Name})
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil
}

// getDynamicClientForCluster creates and returns a dynamic client for a specific cluster.
func (t *Tools) getDynamicClientForCluster(toolReq *mcp.CallToolRequest, cluster string) (dynamic.Interface, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	clusterURL := rancherURL + "/k8s/clusters/" + cluster
	token := toolReq.Extra.Header.Get(tokenHeader)

	dynClient, err := t.createDynamicClientFunc(token, clusterURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}
	return dynClient, nil
}

// TODO modify unit test if we decided to follow this approach once manual evaluation is completed
func (t *Tools) getPodLogs(ctx context.Context, url string, cluster string, token string, pod corev1.Pod) (string, error) {
	clusterURL := url + "/k8s/clusters/" + cluster
	clientset, err := t.createClientSetFunc(token, clusterURL)
	if err != nil {
		return "", fmt.Errorf("failed to create clientset: %w", err)
	}

	logs := ContainerLogs{
		Logs: make(map[string]string),
	}

	for _, container := range pod.Spec.Containers {
		podLogOptions := corev1.PodLogOptions{
			TailLines: ptr.To[int64](50),
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
		return "", fmt.Errorf("error marshalling pod logs:", err)
	}

	return string(jsonData), nil
}

func doRequest(url string, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Cookie", "R_SESS="+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error doing request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}

	respWithoutManagedFields, err := removeManagedFieldsIfPresent(body)
	if err != nil {
		return "", fmt.Errorf("error removing managedFields: %w", err)
	}

	return string(respWithoutManagedFields), nil
}

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

func createClientSet(token string, url string) (kubernetes.Interface, error) {
	restConfig, err := createRestConfig(token, url)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

func createDynamicClient(token string, url string) (dynamic.Interface, error) {
	restConfig, err := createRestConfig(token, url)
	if err != nil {
		return nil, err
	}
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return dynClient, nil
}

func removeManagedFieldsIfPresent(obj []byte) ([]byte, error) {
	var result map[string]interface{}
	err := json.Unmarshal(obj, &result)
	if err != nil {
		// nothing to do
		return obj, nil
	}
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		// nothing to do
		return obj, nil
	}
	delete(metadata, "managedFields")
	respWithoutManagedFields, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return respWithoutManagedFields, nil
}
