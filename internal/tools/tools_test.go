package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestGetKubernetesResource(t *testing.T) {
	podJSON := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"rancher"},"spec":{"containers":[{"name":"rancher-container","image":"rancher:latest"}]}}`
	podJSONWithManagedFields := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"rancher","managedFields":{"apiVersion": "v1","fieldsType":"FieldsV1"}},"spec":{"containers":[{"name":"rancher-container","image":"rancher:latest"}]}}`
	deploymentJSON := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"nginx-deployment"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"name":"nginx","image":"nginx:1.14.2","ports":[{"containerPort":80}]}]}}}}`

	tests := map[string]struct {
		params         GetKubernetesResourceParams
		mockResponse   string
		expectedPath   string
		expectedResult string
		expectedError  string
	}{
		"pod": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResponse:   podJSON,
			expectedPath:   "/k8s/clusters/local/v1/pod/default/rancher",
			expectedResult: podJSON,
		},
		"pod with managed fields": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResponse:   podJSONWithManagedFields,
			expectedPath:   "/k8s/clusters/local/v1/pod/default/rancher",
			expectedResult: podJSON,
		},
		"deployment": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "deployment", Namespace: "default", Cluster: "local"},
			mockResponse:   deploymentJSON,
			expectedPath:   "/k8s/clusters/local/v1/apps.deployment/default/rancher",
			expectedResult: deploymentJSON,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, test.expectedPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(test.mockResponse))
			}))
			defer mockServer.Close()

			result, _, err := GetResource(nil, &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url": {mockServer.URL},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestListKubernetesResource(t *testing.T) {
	podsJSON := `["pod1", "pod2"]`
	deploymentsJSON := `["deployment1", "deployment2"]`

	tests := map[string]struct {
		params         ListKubernetesResourcesParams
		mockResponse   string
		expectedPath   string
		expectedResult string
		expectedError  string
	}{
		"pod": {
			params:         ListKubernetesResourcesParams{Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResponse:   podsJSON,
			expectedPath:   "/k8s/clusters/local/v1/pod/default",
			expectedResult: podsJSON,
		},
		"deployment": {
			params:         ListKubernetesResourcesParams{Kind: "deployment", Namespace: "default", Cluster: "local"},
			mockResponse:   deploymentsJSON,
			expectedPath:   "/k8s/clusters/local/v1/apps.deployment/default",
			expectedResult: deploymentsJSON,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, test.expectedPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(test.mockResponse))
			}))
			defer mockServer.Close()

			result, _, err := ListKubernetesResources(nil, &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url": {mockServer.URL},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestGetNodes(t *testing.T) {
	nodes := `{"type":"collection","resourceType":"node","count":1,"data":[{"id":"k3d-test-server-0","type":"node","apiVersion":"v1","kind":"Node"}]}`
	nodesMetrics := `{"type":"collection","resourceType":"metrics.k8s.io.nodemetrics","count":1,"data":[{"id":"k3d-test-server-0","type":"metrics.k8s.io.nodemetrics","apiVersion":"metrics.k8s.io/v1beta1","kind":"NodeMetrics","name":"k3d-test-server-0","relationships":null,"state":{"error":false,"message":"Resourceiscurrent","name":"active","transitioning":false}},"timestamp":"2025-09-18T15:56:28Z","usage":{"cpu":"215886808n","memory":"2794176Ki"},"window":"20.028s"}]}`
	tests := map[string]struct {
		params                   GetNodesParams
		mockNodesResponse        string
		mockNodesMetricsResponse string
		expectedNodesPath        string
		expectedMetricsPath      string
		expectedResult           string
		expectedError            string
	}{
		"get nodes": {
			params:                   GetNodesParams{Cluster: "local"},
			mockNodesResponse:        nodes,
			mockNodesMetricsResponse: nodesMetrics,
			expectedNodesPath:        "/k8s/clusters/local/v1/nodes",
			expectedMetricsPath:      "/k8s/clusters/local/v1/metrics.k8s.io.nodes",
			expectedResult:           nodes + nodesMetrics,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case test.expectedNodesPath:
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(test.mockNodesResponse))
				case test.expectedMetricsPath:
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(test.mockNodesMetricsResponse))
				default:
					assert.Fail(t, fmt.Sprintf("unexpected path: %s", r.URL.Path))
				}
			}))
			defer mockServer.Close()

			result, _, err := GetNodes(nil, &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url": {mockServer.URL},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}
