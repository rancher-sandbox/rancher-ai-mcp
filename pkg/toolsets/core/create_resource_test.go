package core

import (
	"context"
	"testing"

	"mcp/pkg/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

func createResourceScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

func TestCreateKubernetesResource(t *testing.T) {
	fakeUrl := "https://localhost:8080"
	fakeToken := "fakeToken"

	configMapResource := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "test-config",
			"namespace": "default",
		},
		"data": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	tests := map[string]struct {
		params         createKubernetesResourceParams
		fakeDynClient  *dynamicfake.FakeDynamicClient
		expectedResult string
		expectedError  string
	}{
		"create configmap": {
			params: createKubernetesResourceParams{
				Name:      "test-config",
				Namespace: "default",
				Kind:      "configmap",
				Cluster:   "local",
				Resource:  configMapResource,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(createResourceScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			}),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "v1",
						"data": {"key1": "value1", "key2": "value2"},
						"kind": "ConfigMap",
						"metadata": {"name": "test-config", "namespace": "default"}
					}
				],
				"uiContext": [
					{"namespace": "default", "kind": "ConfigMap", "cluster": "local", "name": "test-config", "type": "configmap"}
				]
			}`,
		},
		"create configmap - marshal error": {
			params: createKubernetesResourceParams{
				Name:      "test-config",
				Namespace: "default",
				Kind:      "configmap",
				Cluster:   "local",
				Resource:  make(chan int),
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClient(createResourceScheme()),
			expectedError: `failed to marshal resource`,
		},
		"create configmap - invalid": {
			params: createKubernetesResourceParams{
				Name:      "test-config",
				Namespace: "default",
				Kind:      "configmap",
				Cluster:   "local",
				Resource:  "invalid-resource-type",
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(createResourceScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			}),
			expectedError: "failed to create unstructured object",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := &client.Client{
				DynClientCreator: func(inConfig *rest.Config) (dynamic.Interface, error) {
					return test.fakeDynClient, nil
				},
			}
			tools := Tools{client: c}

			result, _, err := tools.createKubernetesResource(context.TODO(), &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{Header: map[string][]string{urlHeader: {fakeUrl}, tokenHeader: {fakeToken}}},
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
