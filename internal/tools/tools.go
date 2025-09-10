package tools

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"strings"

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

type GetKubernetesResourceParams struct {
	Name      string `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string `json:"cluster" jsonschema:"the cluster of the resource"`
}

type UpdateKubernetesResourceParams struct {
	Name      string        `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string        `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string        `json:"kind" jsonschema:"the kind of the resource"`
	Cluster   string        `json:"cluster" jsonschema:"the cluster of the resource"`
	Patch     []interface{} `json:"patch" jsonschema:"the patch of the request"`
}

func GetResource(_ context.Context, toolReq *mcp.CallToolRequest, params GetKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
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

	resp, statusCode, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}
	if statusCode != http.StatusOK {
		return nil, nil, errors.New(resp)
	}
	// TODO check if empty!
	// TODO remove links too!
	respWithoutManagedFields, err := removeManagedFields([]byte(resp))
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(respWithoutManagedFields)}},
	}, nil, nil
}

func UpdateKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params UpdateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	clusterURL := rancherURL + "/k8s/clusters/" + params.Cluster
	dynClient, err := createDynamicClient(toolReq.Extra.Header.Get(tokenHeader), clusterURL)
	if err != nil {
		return nil, nil, err
	}
	patchBytes, err := json.Marshal(params.Patch)
	if err != nil {
		return nil, nil, err
	}
	var obj *unstructured.Unstructured
	kind := strings.ToLower(params.Kind)
	if params.Namespace != "" {
		obj, err = dynClient.Resource(k8sKindsToGVRs[kind]).Namespace(params.Namespace).Patch(ctx, params.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	} else {
		obj, err = dynClient.Resource(k8sKindsToGVRs[kind]).Patch(ctx, params.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	}
	if err != nil {
		return nil, nil, err
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, nil, err
	}
	respWithoutManagedFields, err := removeManagedFields(objBytes)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(respWithoutManagedFields)}},
	}, nil, nil
}

func removeManagedFields(obj []byte) ([]byte, error) {
	var result map[string]interface{}
	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		return nil, err
	}
	delete(metadata, "managedFields")
	respWithoutManagedFields, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return respWithoutManagedFields, nil
}

/*
func GetNamespacedKubernetesResourceList(_ context.Context, toolReq *mcp.CallToolRequest, params GetKubernetesResourceListParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	kind := strings.ToLower(params.Kind)
	var reqUrl string
	if k8sKindsToGVRs[kind].Group == "" {
		reqUrl = fmt.Sprintf("%s/%s/%s", rancherURL, steveEndpoint, kind)
	} else {
		reqUrl = fmt.Sprintf("%s/%s/%s.%s", rancherURL, steveEndpoint, k8sKindsToGVRs[kind].Group, kind)
	}
	resp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}*/

func doRequest(url string, token string) (string, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Cookie", "R_SESS="+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	return string(body), resp.StatusCode, nil
}

func createDynamicClient(token string, url string) (*dynamic.DynamicClient, error) {
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
