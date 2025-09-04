package tools

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"net/http"
	"strings"
)

const (
	steveEndpoint = "v1"
	tokenHeader   = "R_token"
	urlHeader     = "R_url"
)

// TODO add missing resources
var k8sKindsToGroups = map[string]string{
	"pod":                     "",
	"service":                 "",
	"configmap":               "",
	"secret":                  "",
	"deployment":              "apps",
	"statefulset":             "apps",
	"daemonset":               "apps",
	"replicaset":              "apps",
	"ingress":                 "networking.k8s.io",
	"networkpolicy":           "networking.k8s.io",
	"horizontalpodautoscaler": "autoscaling",
	"serviceaccount":          "",
	"role":                    "rbac.authorization.k8s.io",
	"rolebinding":             "rbac.authorization.k8s.io",
	"clusterrole":             "rbac.authorization.k8s.io",
	"clusterrolebinding":      "rbac.authorization.k8s.io",
	"persistentvolume":        "",
	"persistentvolumeclaim":   "",
	"project":                 "management.cattle.io",
	"cluster":                 "management.cattle.io",
	"user":                    "management.cattle.io",
	"bundle":                  "fleet.cattle.io",
	"gitrepo":                 "fleet.cattle.io",
}

type GetKubernetesResourceParams struct {
	Name      string `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
}

type GetNonNamespacedResourceParams struct {
	Name string `json:"name" jsonschema:"the name of k8s resource"`
	Kind string `json:"kind" jsonschema:"the kind of the resource"`
}

type GetKubernetesResourceListParams struct {
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string `json:"kind" jsonschema:"the kind of the resource"`
}

// TODO check if it'd be better to have only one tool for both namespaced and non-namespaced resources
func GetNamespacedKubernetesResource(_ context.Context, toolReq *mcp.CallToolRequest, params GetKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	kind := strings.ToLower(params.Kind)
	var reqUrl string
	if k8sKindsToGroups[kind] == "" {
		reqUrl = fmt.Sprintf("%s/%s/%s/%s/%s", rancherURL, steveEndpoint, kind, params.Namespace, params.Name)
	} else {
		reqUrl = fmt.Sprintf("%s/%s/%s.%s/%s/%s", rancherURL, steveEndpoint, k8sKindsToGroups[kind], kind, params.Namespace, params.Name)
	}
	resp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}

func GetNonNamespacedKubernetesResource(_ context.Context, toolReq *mcp.CallToolRequest, params GetNonNamespacedResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	kind := strings.ToLower(params.Kind)
	var reqUrl string
	if k8sKindsToGroups[kind] == "" {
		reqUrl = fmt.Sprintf("%s/%s/%s/%s", rancherURL, steveEndpoint, kind, params.Name)
	} else {
		reqUrl = fmt.Sprintf("%s/%s/%s.%s/%s", rancherURL, steveEndpoint, k8sKindsToGroups[kind], kind, params.Name)
	}
	resp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}

func GetNamespacedKubernetesResourceList(_ context.Context, toolReq *mcp.CallToolRequest, params GetKubernetesResourceListParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	kind := strings.ToLower(params.Kind)
	var reqUrl string
	if k8sKindsToGroups[kind] == "" {
		reqUrl = fmt.Sprintf("%s/%s/%s", rancherURL, steveEndpoint, kind)
	} else {
		reqUrl = fmt.Sprintf("%s/%s/%s.%s", rancherURL, steveEndpoint, k8sKindsToGroups[kind], kind)
	}
	resp, err := doRequest(reqUrl, toolReq.Extra.Header.Get(tokenHeader))
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}

func doRequest(url string, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", "R_SESS="+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
