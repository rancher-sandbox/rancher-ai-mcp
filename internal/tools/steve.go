package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"net/http"
	"strings"
)

const steveEndpointVersion = "v1"

// ListRoot represents the structure of a list response from the Steve API.
type ListRoot struct {
	Data []map[string]any `json:"data"`
}

// fetchParams holds the parameters required to fetch a resource from Steve.
type fetchParams struct {
	cluster   string // The cluster ID.
	kind      string // The kind of the Kubernetes resource (e.g., "pod", "deployment").
	namespace string // The namespace of the resource (optional).
	name      string // The name of the resource (optional).
	url       string // The base URL of the Steve API.
	token     string // The authentication token for Steve.
	filter    string // Optional filter string for the request.
}

// steveFetcher is a struct that handles fetching resources from the Steve API.
type steveFetcher struct{}

// newSteveFetcher creates and returns a new instance of steveFetcher.
func newSteveFetcher() *steveFetcher {
	return &steveFetcher{}
}

// fetchK8sResource fetches a single Kubernetes resource from Steve and decodes it into an unstructured.Unstructured object.
func (s *steveFetcher) fetchK8sResource(params fetchParams) (*unstructured.Unstructured, error) {
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

// fetchK8sResources fetches a list of Kubernetes resources from Steve and decodes them into a slice of unstructured.Unstructured objects.
func (s *steveFetcher) fetchK8sResources(params fetchParams) ([]*unstructured.Unstructured, error) {
	resp, err := s.fetchFromSteve(params)
	if err != nil {
		return nil, err
	}

	var root ListRoot
	if err := json.Unmarshal([]byte(resp), &root); err != nil {
		return nil, fmt.Errorf("error parsing response : %w", err)
	}
	var objs []*unstructured.Unstructured
	for _, item := range root.Data {
		objs = append(objs, &unstructured.Unstructured{
			Object: item,
		})
	}

	return objs, nil
}

// fetchFromSteve constructs the request URL and makes the HTTP request to the Steve API.
func (s *steveFetcher) fetchFromSteve(params fetchParams) (string, error) {
	lowerKind := strings.ToLower(params.kind)
	reqURL := fmt.Sprintf("%s/k8s/clusters/%s/%s", params.url, params.cluster, steveEndpointVersion)
	resourcePath := lowerKind
	if gvr, ok := k8sKindsToGVRs[lowerKind]; ok && gvr.Group != "" {
		resourcePath = gvr.Group + "." + lowerKind
	}
	reqURL = reqURL + "/" + resourcePath
	if params.namespace != "" {
		reqURL = reqURL + "/" + params.namespace
	}
	if params.name != "" {
		reqURL = reqURL + "/" + params.name
	}
	if params.filter != "" {
		reqURL = reqURL + "?" + params.filter
	}

	resp, err := doRequest(reqURL, params.token)
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
