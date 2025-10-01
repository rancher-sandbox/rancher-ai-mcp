package k8s

import (
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"mcp/internal/tools/converter"
	"net/http"
	"strings"
)

const steveEndpointVersion = "v1"

// ListRoot represents the structure of a list response from the Steve API.
type ListRoot struct {
	Data []map[string]any `json:"data"`
}

// FetchParams holds the parameters required to fetch a resource from Steve.
type FetchParams struct {
	Cluster   string // The Cluster ID.
	Kind      string // The Kind of the Kubernetes resource (e.g., "pod", "deployment").
	Namespace string // The Namespace of the resource (optional).
	Name      string // The Name of the resource (optional).
	URL       string // The base URL of the Steve API.
	Token     string // The authentication Token for Steve.
	Filter    string // Optional Filter string for the request.
}

// SteveFetcher is a struct that handles fetching resources from the Steve API.
type SteveFetcher struct{}

// NewSteveFetcher creates and returns a new instance of SteveFetcher.
func NewSteveFetcher() *SteveFetcher {
	return &SteveFetcher{}
}

// FetchK8sResource fetches a single Kubernetes resource from Steve and decodes it into an unstructured.Unstructured object.
func (s *SteveFetcher) FetchK8sResource(params FetchParams) (*unstructured.Unstructured, error) {
	resp, err := s.fetchFromSteve(params)
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(resp), 4096)
	if err := decoder.Decode(obj); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return obj, nil
}

// FetchK8sResources fetches a list of Kubernetes resources from Steve and decodes them into a slice of unstructured.Unstructured objects.
func (s *SteveFetcher) FetchK8sResources(params FetchParams) ([]*unstructured.Unstructured, error) {
	resp, err := s.fetchFromSteve(params)
	if err != nil {
		return nil, err
	}

	var root ListRoot
	if err := json.Unmarshal([]byte(resp), &root); err != nil {
		return nil, fmt.Errorf("failed to decode response : %w", err)
	}
	var objs []*unstructured.Unstructured
	for _, item := range root.Data {
		objs = append(objs, &unstructured.Unstructured{
			Object: item,
		})
	}

	return objs, nil
}

// FetchFromSteve constructs the request URL and makes the HTTP request to the Steve API.
func (s *SteveFetcher) fetchFromSteve(params FetchParams) (string, error) {
	lowerKind := strings.ToLower(params.Kind)
	reqURL := fmt.Sprintf("%s/k8s/clusters/%s/%s", params.URL, params.Cluster, steveEndpointVersion)
	resourcePath := lowerKind
	if gvr, ok := converter.K8sKindsToGVRs[lowerKind]; ok && gvr.Group != "" {
		resourcePath = gvr.Group + "." + lowerKind
	}
	reqURL = reqURL + "/" + resourcePath
	if params.Namespace != "" {
		reqURL = reqURL + "/" + params.Namespace
	}
	if params.Name != "" {
		reqURL = reqURL + "/" + params.Name
	}
	if params.Filter != "" {
		reqURL = reqURL + "?" + params.Filter
	}

	resp, err := doRequest(reqURL, params.Token)
	if err != nil {
		return "", fmt.Errorf("failed API request to %s: %w", reqURL, err)
	}

	return resp, nil
}

// doRequest performs an HTTP GET request to Steve.
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
		return "", fmt.Errorf("response error: %s", string(body))
	}

	return string(body), nil
}
