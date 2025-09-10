package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"regexp"
	"strings"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	steveEndpoint = "v1"
	tokenHeader   = "R_token"
	urlHeader     = "R_url"
	firstLen      = 49
	certDelim     = "\\\n      "
)

var (
	splitRegexp = regexp.MustCompile(`\S{1,76}`)
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

type UpdateKubernetesResourceParams struct {
	Name      string        `json:"name" jsonschema:"the name of k8s resource"`
	Namespace string        `json:"namespace" jsonschema:"the namespace of the resource"`
	Kind      string        `json:"kind" jsonschema:"the kind of the resource"`
	Patch     []interface{} `json:"patch" jsonschema:"the patch of the request"`
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

func UpdateNamespacedKubernetesResource(ctx context.Context, toolReq *mcp.CallToolRequest, params UpdateKubernetesResourceParams) (*mcp.CallToolResult, any, error) {
	rancherURL := toolReq.Extra.Header.Get(urlHeader)
	clusterURL := rancherURL + "/k8s/clusters/local"
	dynClient, err := createDynamicClient(toolReq.Extra.Header.Get(tokenHeader), clusterURL)
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	patchBytes, err := json.Marshal(params.Patch)
	if err != nil {
		return nil, nil, err
	}
	obj, err := dynClient.Resource(gvr).Namespace(params.Namespace).Patch(ctx, params.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, nil, err
	}
	fmt.Println(obj)

	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
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

func formatCertString(certData string) string {
	buf := &bytes.Buffer{}
	if len(certData) > firstLen {
		buf.WriteString(certData[:firstLen])
		certData = certData[firstLen:]
	} else {
		return certData
	}

	for _, part := range splitRegexp.FindAllStringSubmatch(certData, -1) {
		buf.WriteString(certDelim)
		buf.WriteString(part[0])
	}

	return buf.String()
}

func caCertString() string {
	certData := `-----BEGIN CERTIFICATE-----
  MIIBvDCCAWOgAwIBAgIBADAKBggqhkjOPQQDAjBGMRwwGgYDVQQKExNkeW5hbWlj
  bGlzdGVuZXItb3JnMSYwJAYDVQQDDB1keW5hbWljbGlzdGVuZXItY2FAMTc1NzMy
  NTYwMTAeFw0yNTA5MDgxMDAwMDFaFw0zNTA5MDYxMDAwMDFaMEYxHDAaBgNVBAoT
  E2R5bmFtaWNsaXN0ZW5lci1vcmcxJjAkBgNVBAMMHWR5bmFtaWNsaXN0ZW5lci1j
  YUAxNzU3MzI1NjAxMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEl7ynOU4cwm2N
  9/MQB5To5PKx+ZE+LeKUc0ftt/KvVJa4NIMlNqsNPEO1GQ3UCGprF9dEnK1a34JB
  fatwpnZA26NCMEAwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wHQYD
  VR0OBBYEFB3an3o9DMrBJtHD6oiBVFVvdOIuMAoGCCqGSM49BAMCA0cAMEQCIFoi
  XFAZ+YHTwNR6FN+t9CMK9jkBFDLOAvDsZJD+GrjBAiAfUMOwdx7Z3eX2hJjlQPej
  X5jf0OdLHZqOTUAV/zipFw==
  -----END CERTIFICATE-----`
	if certData == "" {
		return ""
	}
	certData = base64.StdEncoding.EncodeToString([]byte(certData))
	return formatCertString(certData)
}

func createDynamicClient(token string, url string) (*dynamic.DynamicClient, error) {
	kubeconfig := clientcmdapi.NewConfig()

	kubeconfig.Clusters["cluster"] = &clientcmdapi.Cluster{
		Server: url,
		//CertificateAuthorityData: []byte(caCertString()),
		InsecureSkipTLSVerify: true,
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

/*
func getCAFromSettings() string {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		log.Fatalf("failed to get kubeconfig: %v", err)
	}

	// Create a dynamic client
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("failed to create dynamic client: %v", err)
	}

	// Define the GroupVersionResource for Rancher settings
	gvr := schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "settings", // plural of the CRD
	}

	// Fetch the resource named "aaa"
	setting, err := dynClient.Resource(gvr).Get(context.TODO(), "aaa", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to get setting: %v", err)
	}

	// Print the unstructured object
	fmt.Printf("Got setting: %v\n", setting)

	return ""

}
*/
