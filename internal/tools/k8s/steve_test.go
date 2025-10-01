package k8s

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchK8sResource(t *testing.T) {
	tests := map[string]struct {
		params             FetchParams
		mockResponse       string
		mockResponseHeader int
		expectedPath       string
		expectedResult     *unstructured.Unstructured
		expectedError      string
	}{
		"pod": {
			params: FetchParams{
				Name:      "rancher",
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Token:     "Token",
			},
			mockResponse:       `{"apiVersion":"v1","kind":"Pod","metadata":{"Name":"rancher"},"spec":{"containers":[{"Name":"rancher-container","image":"rancher:latest"}]}}`,
			mockResponseHeader: http.StatusOK,
			expectedPath:       "/k8s/clusters/local/v1/pod/default/rancher",
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"Name": "rancher",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"Name":  "rancher-container",
								"image": "rancher:latest",
							},
						},
					},
				},
			},
		},
		"invalid response": {
			params: FetchParams{
				Name:      "rancher",
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Token:     "Token",
			},
			mockResponse:       `invalid response`,
			mockResponseHeader: http.StatusOK,
			expectedPath:       "/k8s/clusters/local/v1/pod/default/rancher",
			expectedError:      "failed to decode response",
		},
		"error fetching": {
			params: FetchParams{
				Name:      "rancher",
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Token:     "Token",
			},
			mockResponse:       `error`,
			mockResponseHeader: http.StatusInternalServerError,
			expectedPath:       "/k8s/clusters/local/v1/pod/default/rancher",
			expectedError:      "failed API request",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, test.expectedPath, r.URL.Path)
				w.WriteHeader(test.mockResponseHeader)
				w.Write([]byte(test.mockResponse))
			}))
			defer mockServer.Close()
			test.params.URL = mockServer.URL
			f := &SteveFetcher{}

			result, err := f.FetchK8sResource(test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}

func TestFetchK8sResources(t *testing.T) {
	tests := map[string]struct {
		params             FetchParams
		mockResponse       string
		mockResponseHeader int
		expectedPath       string
		expectedResult     []*unstructured.Unstructured
		expectedError      string
	}{
		"pod": {
			params: FetchParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Token:     "Token",
			},
			mockResponse:       `{"data":[{"apiVersion":"v1","kind":"Pod","metadata":{"Name":"rancher"},"spec":{"containers":[{"Name":"rancher-container","image":"rancher:latest"}]}}, {"apiVersion":"v1","kind":"Pod","metadata":{"Name":"rancher2"},"spec":{"containers":[{"Name":"rancher-container2","image":"rancher:latest2"}]}}]}`,
			mockResponseHeader: http.StatusOK,
			expectedPath:       "/k8s/clusters/local/v1/pod/default",
			expectedResult: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"Name": "rancher",
						},
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"Name":  "rancher-container",
									"image": "rancher:latest",
								},
							},
						},
					},
				},
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"Name": "rancher2",
						},
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"Name":  "rancher-container2",
									"image": "rancher:latest2",
								},
							},
						},
					},
				},
			},
		},
		"invalid response": {
			params: FetchParams{
				Name:      "rancher",
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Token:     "Token",
			},
			mockResponse:       `invalid response`,
			mockResponseHeader: http.StatusOK,
			expectedPath:       "/k8s/clusters/local/v1/pod/default/rancher",
			expectedError:      "failed to decode response",
		},
		"error fetching": {
			params: FetchParams{
				Name:      "rancher",
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Token:     "Token",
			},
			mockResponse:       `error`,
			mockResponseHeader: http.StatusInternalServerError,
			expectedPath:       "/k8s/clusters/local/v1/pod/default/rancher",
			expectedError:      "failed API request",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, test.expectedPath, r.URL.Path)
				w.WriteHeader(test.mockResponseHeader)
				w.Write([]byte(test.mockResponse))
			}))
			defer mockServer.Close()
			test.params.URL = mockServer.URL
			f := &SteveFetcher{}

			result, err := f.FetchK8sResources(test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}
