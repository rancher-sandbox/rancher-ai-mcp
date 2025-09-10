package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetNamespacedKubernetesResource(t *testing.T) {
	podJSON := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"rancher"},"spec":{"containers":[{"name":"rancher-container","image":"rancher:latest"}]}}`
	deploymentJSON := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"nginx-deployment"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"name":"nginx","image":"nginx:1.14.2","ports":[{"containerPort":80}]}]}}}}`

	tests := map[string]struct {
		params         GetKubernetesResourceParams
		mockResponse   string
		expectedPath   string
		expectedResult string
		expectedError  string
	}{
		"pod": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "pod", Namespace: "default"},
			mockResponse:   podJSON,
			expectedPath:   "/v1/pod/default/rancher",
			expectedResult: podJSON,
		},
		"deployment": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "deployment", Namespace: "default"},
			mockResponse:   deploymentJSON,
			expectedPath:   "/v1/apps.deployment/default/rancher",
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
				assert.Equal(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestGetNonNamespacedKubernetesResource(t *testing.T) {
	userJSON := `{"apiVersion":"management.cattle.io/v3","description":"","displayName":"Testuser","kind":"User","metadata":{"principalIds":["system://local","local://u-b4qkhsnliz"]}`

	tests := map[string]struct {
		params         GetNonNamespacedResourceParams
		mockResponse   string
		expectedPath   string
		expectedResult string
		expectedError  string
	}{
		"user": {
			params:         GetNonNamespacedResourceParams{Name: "user-abc", Kind: "user"},
			mockResponse:   userJSON,
			expectedPath:   "/v1/management.cattle.io.user/user-abc",
			expectedResult: userJSON,
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

			result, _, err := GetNonNamespacedKubernetesResource(nil, &mcp.CallToolRequest{
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
