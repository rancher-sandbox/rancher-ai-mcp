package tools

import (
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
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	steveEndpoint = "v1"
	tokenHeader   = "R_token"
	urlHeader     = "R_url"
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

// GetKubernetesResourceParams defines the structure for requesting a general Kubernetes resource.
// It includes fields required to uniquely identify a resource within a cluster.
type GetKubernetesResourceParams struct {
	Name      string `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
}

// GetPodDetailsParams specifies the parameters needed to retrieve details about a specific pod.
type GetPodDetailsParams struct {
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

// ListKubernetesResourcesParams specifies the parameters needed to list kubernetes resources.
type ListKubernetesResourcesParams struct {
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
}

type Tools struct {
	createDynamicClientFunc func(token string, url string) (dynamic.Interface, error)
}

func NewTools() *Tools {
	return &Tools{
		createDynamicClientFunc: createDynamicClient,
	}
}

func (t *Tools) GetResource(_ context.Context, toolReq *mcp.CallToolRequest, params GetKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	kind := strings.ToLower(params.Kind)
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint
	if k8sKindsToGVRs[kind].Group != "" {
		reqUrl += "/" + k8sKindsToGVRs[kind].Group + "." + kind
	} else {
		reqUrl += "/" + kind
	}
	if params.Namespace != "" {
		reqUrl += "/" + params.Namespace
	}
	reqUrl += "/" + params.Name
	resp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get resouce %s in namesapce %s: %w", params.Name, params.Namespace, err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}

func (t *Tools) UpdateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params UpdateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	clusterURL := rancherURL + "/k8s/clusters/" + params.Cluster
	dynClient, err := t.createDynamicClientFunc(toolReq.Extra.Header.Get(tokenHeader), clusterURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}
	patchBytes, err := json.Marshal(params.Patch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create patch: %w", err)
	}
	var obj *unstructured.Unstructured
	kind := strings.ToLower(params.Kind)
	if params.Namespace != "" {
		obj, err = dynClient.Resource(k8sKindsToGVRs[kind]).Namespace(params.Namespace).Patch(ctx, params.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	} else {
		obj, err = dynClient.Resource(k8sKindsToGVRs[kind]).Patch(ctx, params.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update resource %s in namespace %s: %w", params.Name, params.Namespace, err)
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	respWithoutManagedFields, err := removeManagedFieldsIfPresent(objBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to remove managedFields: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(respWithoutManagedFields)}},
	}, nil, nil
}

func (t *Tools) ListKubernetesResources(_ context.Context, toolReq *mcp.CallToolRequest, params ListKubernetesResourcesParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	kind := strings.ToLower(params.Kind)
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint
	if k8sKindsToGVRs[kind].Group != "" {
		reqUrl += "/" + k8sKindsToGVRs[kind].Group + "." + kind
	} else {
		reqUrl += "/" + kind
	}
	if params.Namespace != "" {
		reqUrl += "/" + params.Namespace
	}
	resp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}

func (t *Tools) GetPodDetails(_ context.Context, toolReq *mcp.CallToolRequest, params GetPodDetailsParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/pods/" + params.Namespace + "/" + params.Name
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
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/apps.replicasets/" + params.Namespace + "/" + replicaSetName
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
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/apps.deployments/" + params.Namespace + "/" + deploymentName
	deploymentResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}

	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/metrics.k8s.io.pods/" + params.Namespace + "/" + params.Name
	metricResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: podResp + deploymentResp + metricResp}},
	}, nil, nil
}

func (t *Tools) GetDeploymentDetails(_ context.Context, toolReq *mcp.CallToolRequest, params GetPodDetailsParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/apps.deployments/" + params.Namespace + "/" + params.Name
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
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/pods/" + params.Namespace + "?" + filter
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
	reqUrl := rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/nodes"
	nodeResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	reqUrl = rancherURL + "/k8s/clusters/" + params.Cluster + "/" + steveEndpoint + "/metrics.k8s.io.nodes"
	metricsResp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: nodeResp + metricsResp}},
	}, nil, nil

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

func createDynamicClient(token string, url string) (dynamic.Interface, error) {
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
