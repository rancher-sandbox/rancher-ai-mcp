package rancher

import (
	"context"
	"testing"

	"mcp/pkg/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

var fakeConfigMapForPatch = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-config",
		Namespace: "default",
	},
	Data: map[string]string{
		"key1": "value1",
		"key2": "value2",
	},
}

func patchResourceScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

func TestUpdateKubernetesResource(t *testing.T) {
	fakeUrl := "https://localhost:8080"
	fakeToken := "fakeToken"

	tests := map[string]struct {
		params         UpdateKubernetesResourceParams
		fakeDynClient  *dynamicfake.FakeDynamicClient
		expectedResult string
		expectedError  string
	}{
		"update configmap - add new key": {
			params: UpdateKubernetesResourceParams{
				Name:      "test-config",
				Namespace: "default",
				Kind:      "configmap",
				Cluster:   "local",
				Patch: []JSONPatch{
					{
						Op:    "add",
						Path:  "/data/key3",
						Value: "value3",
					},
				},
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(patchResourceScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			}, fakeConfigMapForPatch),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "v1",
						"data": {"key1": "value1", "key2": "value2", "key3": "value3"},
						"kind": "ConfigMap",
						"metadata": {"name": "test-config", "namespace": "default"}
					}
				],
				"uiContext": [
					{"cluster": "local", "kind": "ConfigMap", "name": "test-config", "namespace": "default", "type": "configmap"}
				]
			}`,
		},
		"update configmap - replace existing key": {
			params: UpdateKubernetesResourceParams{
				Name:      "test-config",
				Namespace: "default",
				Kind:      "configmap",
				Cluster:   "local",
				Patch: []JSONPatch{
					{
						Op:    "replace",
						Path:  "/data/key1",
						Value: "updated-value",
					},
				},
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(patchResourceScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			}, fakeConfigMapForPatch),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "v1",
						"data": {"key1": "updated-value", "key2": "value2"},
						"kind": "ConfigMap",
						"metadata": {"name": "test-config", "namespace": "default"}
					}
				],
				"uiContext": [
					{"cluster": "local", "kind": "ConfigMap", "name": "test-config", "namespace": "default", "type": "configmap"}
				]
			}`,
		},
		"update configmap - remove key": {
			params: UpdateKubernetesResourceParams{
				Name:      "test-config",
				Namespace: "default",
				Kind:      "configmap",
				Cluster:   "local",
				Patch: []JSONPatch{
					{
						Op:   "remove",
						Path: "/data/key2",
					},
				},
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(patchResourceScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			}, fakeConfigMapForPatch),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "v1",
						"data": {"key1": "value1"},
						"kind": "ConfigMap",
						"metadata": {"name": "test-config", "namespace": "default"}
					}
				],
				"uiContext": [
					{"cluster": "local", "kind": "ConfigMap", "name": "test-config", "namespace": "default", "type": "configmap"}
				]
			}`,
		},
		"update configmap - not found": {
			params: UpdateKubernetesResourceParams{
				Name:      "nonexistent-config",
				Namespace: "default",
				Kind:      "configmap",
				Cluster:   "local",
				Patch: []JSONPatch{
					{
						Op:    "replace",
						Path:  "/data/key1",
						Value: "value",
					},
				},
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(patchResourceScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			}),
			expectedError: `configmaps "nonexistent-config" not found`,
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

			result, _, err := tools.UpdateKubernetesResource(context.TODO(), &mcp.CallToolRequest{
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
