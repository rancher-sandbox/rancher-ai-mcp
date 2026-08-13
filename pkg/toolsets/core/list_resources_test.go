package core

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rancher/rancher-ai-mcp/internal/middleware"
	"github.com/rancher/rancher-ai-mcp/pkg/client"
	"github.com/rancher/rancher-ai-mcp/pkg/client/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

var fakePod1 = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod-1",
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "nginx",
				Image: "nginx:latest",
			},
		},
	},
	Status: corev1.PodStatus{
		Phase: corev1.PodRunning,
	},
}

var fakePod2 = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod-2",
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "redis",
				Image: "redis:latest",
			},
		},
	},
	Status: corev1.PodStatus{
		Phase: corev1.PodRunning,
	},
}

var fakePod3 = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod-3",
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "busybox",
				Image: "busybox:latest",
			},
		},
	},
	Status: corev1.PodStatus{
		Phase: corev1.PodPending,
	},
}

// pods used to verify sort ordering. They are intentionally declared out of
// namespace/name order.
var fakePodBravo = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod-1",
		Namespace: "bravo",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{{Name: "nginx", Image: "nginx:latest"}},
	},
	Status: corev1.PodStatus{Phase: corev1.PodRunning},
}

var fakePodAlphaB = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod-2",
		Namespace: "alpha",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{{Name: "redis", Image: "redis:latest"}},
	},
	Status: corev1.PodStatus{Phase: corev1.PodRunning},
}

var fakePodAlphaA = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod-1",
		Namespace: "alpha",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{{Name: "busybox", Image: "busybox:latest"}},
	},
	Status: corev1.PodStatus{Phase: corev1.PodRunning},
}

func listResourcesScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

func TestListKubernetesResources(t *testing.T) {
	fakeUrl := "https://localhost:8080"
	fakeToken := "fakeToken"

	tests := map[string]struct {
		params        listKubernetesResourcesParams
		fakeDynClient *dynamicfake.FakeDynamicClient
		// used in the CallToolRequest
		requestURL string
		// used in the creation of the Tools.
		rancherURL     string
		expectedResult string
		expectedError  string
	}{
		"list pods in namespace": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": [
					{
						"metadata": {"name": "pod-1", "namespace": "default"},
						"spec": {"containers": [{"image": "nginx:latest", "name": "nginx", "resources": {}}]},
						"status": {"phase": "Running"}
					},
					{
						"metadata": {"name": "pod-2", "namespace": "default"},
						"spec": {"containers": [{"image": "redis:latest", "name": "redis", "resources": {}}]},
						"status": {"phase": "Running"}
					}
				]
			}`,
		},
		"list pods - empty namespace": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "kube-system",
				Cluster:   "local",
			},
			requestURL: fakeUrl,
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}),
			expectedResult: `{"llm": "no resources found"}`,
		},
		"list pods - when tool is configured with URL": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2),
			rancherURL: fakeUrl,
			expectedResult: `{
				"llm": [
					{
						"metadata": {"name": "pod-1", "namespace": "default"},
						"spec": {"containers": [{"image": "nginx:latest", "name": "nginx", "resources": {}}]},
						"status": {"phase": "Running"}
					},
					{
						"metadata": {"name": "pod-2", "namespace": "default"},
						"spec": {"containers": [{"image": "redis:latest", "name": "redis", "resources": {}}]},
						"status": {"phase": "Running"}
					}
				]
			}`,
		},
		"list pods - with explicit limit": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Limit:     1,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": {
					"resources": [
						{
							"metadata": {"name": "pod-1", "namespace": "default"},
							"spec": {"containers": [{"image": "nginx:latest", "name": "nginx", "resources": {}}]},
							"status": {"phase": "Running"}
						}
					],
					"note": "Returned 1 resources (offset 0, limit 1) out of 2 total. Use a namespace or label selector to narrow results, or increase the limit. To get the next page, set offset=1."
				}
			}`,
		},
		"list pods - with offset paging": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Limit:     1,
				Offset:    1,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": {
					"resources": [
						{
							"metadata": {"name": "pod-2", "namespace": "default"},
							"spec": {"containers": [{"image": "redis:latest", "name": "redis", "resources": {}}]},
							"status": {"phase": "Running"}
						}
					],
					"note": "Returned 1 resources (offset 1, limit 1) out of 2 total. Use a namespace or label selector to narrow results, or increase the limit."
				}
			}`,
		},
		"list pods - with jsonpath filter": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				JSONPath:  `@.status.phase=="Pending"`,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2, fakePod3),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": [
					{
						"metadata": {"name": "pod-3", "namespace": "default"},
						"spec": {"containers": [{"image": "busybox:latest", "name": "busybox", "resources": {}}]},
						"status": {"phase": "Pending"}
					}
				]
			}`,
		},
		"list pods - jsonpath filter combined with paging": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				JSONPath:  `@.status.phase=="Running"`,
				Limit:     1,
				Offset:    1,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2, fakePod3),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": {
					"resources": [
						{
							"metadata": {"name": "pod-2", "namespace": "default"},
							"spec": {"containers": [{"image": "redis:latest", "name": "redis", "resources": {}}]},
							"status": {"phase": "Running"}
						}
					],
					"note": "Returned 1 resources (offset 1, limit 1) out of 2 total matching the JSONPath filter. Use a namespace or label selector to narrow results, or increase the limit."
				}
			}`,
		},
		"list pods - offset beyond total": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Offset:    10,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": {
					"resources": "no resources found",
					"note": "Returned 0 resources (offset 10, limit 100) out of 2 total. Use a namespace or label selector to narrow results, or increase the limit."
				}
			}`,
		},
		"list pods - invalid jsonpath": {
			params: listKubernetesResourcesParams{
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				JSONPath:  `@.status.phase==`,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePod1, fakePod2),
			requestURL:    fakeUrl,
			expectedError: "invalid jsonPath filter",
		},
		"list pods - sorted by namespace then name": {
			params: listKubernetesResourcesParams{
				Kind:    "pod",
				Cluster: "local",
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePodBravo, fakePodAlphaB, fakePodAlphaA),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": [
					{
						"metadata": {"name": "pod-1", "namespace": "alpha"},
						"spec": {"containers": [{"image": "busybox:latest", "name": "busybox", "resources": {}}]},
						"status": {"phase": "Running"}
					},
					{
						"metadata": {"name": "pod-2", "namespace": "alpha"},
						"spec": {"containers": [{"image": "redis:latest", "name": "redis", "resources": {}}]},
						"status": {"phase": "Running"}
					},
					{
						"metadata": {"name": "pod-1", "namespace": "bravo"},
						"spec": {"containers": [{"image": "nginx:latest", "name": "nginx", "resources": {}}]},
						"status": {"phase": "Running"}
					}
				]
			}`,
		},
		"list pods - sorted ordering respected with paging": {
			params: listKubernetesResourcesParams{
				Kind:    "pod",
				Cluster: "local",
				Limit:   1,
				Offset:  1,
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(listResourcesScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}, fakePodBravo, fakePodAlphaB, fakePodAlphaA),
			requestURL: fakeUrl,
			expectedResult: `{
				"llm": {
					"resources": [
						{
							"metadata": {"name": "pod-2", "namespace": "alpha"},
							"spec": {"containers": [{"image": "redis:latest", "name": "redis", "resources": {}}]},
							"status": {"phase": "Running"}
						}
					],
					"note": "Returned 1 resources (offset 1, limit 1) out of 3 total. Use a namespace or label selector to narrow results, or increase the limit. To get the next page, set offset=2."
				}
			}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &client.Client{
				DynClientCreator: func(inConfig *rest.Config) (dynamic.Interface, error) {
					return tt.fakeDynClient, nil
				},
			}
			tools := NewTools(test.WrapClient(c, fakeToken), false)
			req := test.NewCallToolRequest(tt.requestURL)

			result, _, err := tools.listKubernetesResources(middleware.WithToken(t.Context(), fakeToken), req, tt.params)

			if tt.expectedError != "" {
				assert.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Len(t, result.Content, 1, "expected a single content entry with valid JSON")
				assert.JSONEq(t, tt.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func makeUnstructuredPod(name, namespace, phase string, labels map[string]string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"status": map[string]interface{}{
			"phase": phase,
		},
	}
	if len(labels) > 0 {
		labelMap := make(map[string]interface{}, len(labels))
		for k, v := range labels {
			labelMap[k] = v
		}
		obj["metadata"].(map[string]interface{})["labels"] = labelMap
	}
	return &unstructured.Unstructured{Object: obj}
}

func makeContainerStatusPod(name string, restartCount int64, waitingReason string) *unstructured.Unstructured {
	cs := map[string]interface{}{"name": "app", "restartCount": restartCount}
	if waitingReason != "" {
		cs["state"] = map[string]interface{}{
			"waiting": map[string]interface{}{"reason": waitingReason},
		}
	}
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{"name": name, "namespace": "default"},
			"status":   map[string]interface{}{"containerStatuses": []interface{}{cs}},
		},
	}
}

func TestFilterByJSONPath(t *testing.T) {
	running1 := makeUnstructuredPod("pod-1", "default", "Running", map[string]string{"app": "nginx"})
	running2 := makeUnstructuredPod("pod-2", "default", "Running", map[string]string{"app": "redis"})
	pending := makeUnstructuredPod("pod-3", "default", "Pending", map[string]string{"app": "busybox"})
	failed := makeUnstructuredPod("pod-4", "default", "Failed", nil)

	tests := map[string]struct {
		objs          []*unstructured.Unstructured
		expr          string
		expectedNames []string
		expectError   string
	}{
		"matches single phase": {
			objs:          []*unstructured.Unstructured{running1, running2, pending},
			expr:          `@.status.phase=="Running"`,
			expectedNames: []string{"pod-1", "pod-2"},
		},
		"matches no resources": {
			objs:          []*unstructured.Unstructured{running1, running2},
			expr:          `@.status.phase=="Failed"`,
			expectedNames: []string{},
		},
		"matches all resources": {
			objs:          []*unstructured.Unstructured{running1, running2},
			expr:          `@.status.phase=="Running"`,
			expectedNames: []string{"pod-1", "pod-2"},
		},
		"logical OR matches two different phases": {
			objs:          []*unstructured.Unstructured{running1, pending, failed},
			expr:          `@.status.phase=="Running" || @.status.phase=="Pending"`,
			expectedNames: []string{"pod-1", "pod-3"},
		},
		"logical OR where only one branch matches": {
			objs:          []*unstructured.Unstructured{running1, running2, pending},
			expr:          `@.status.phase=="Failed" || @.status.phase=="Pending"`,
			expectedNames: []string{"pod-3"},
		},
		"logical OR matching all": {
			objs:          []*unstructured.Unstructured{running1, pending, failed},
			expr:          `@.status.phase=="Running" || @.status.phase=="Pending" || @.status.phase=="Failed"`,
			expectedNames: []string{"pod-1", "pod-3", "pod-4"},
		},
		"matches by label": {
			objs:          []*unstructured.Unstructured{running1, running2, pending},
			expr:          `@.metadata.labels.app=="nginx"`,
			expectedNames: []string{"pod-1"},
		},
		"missing field treated as non-match": {
			objs:          []*unstructured.Unstructured{running1, failed},
			expr:          `@.metadata.labels.app=="nginx"`,
			expectedNames: []string{"pod-1"},
		},
		"empty input returns empty slice": {
			objs:          []*unstructured.Unstructured{},
			expr:          `@.status.phase=="Running"`,
			expectedNames: []string{},
		},
		"invalid expression returns error": {
			objs:        []*unstructured.Unstructured{running1},
			expr:        `@.status.phase==`,
			expectError: "invalid jsonPath filter",
		},
		"nested predicate OR - restartCount or CrashLoopBackOff": {
			objs: []*unstructured.Unstructured{
				makeContainerStatusPod("pod-high-restart", int64(5), ""),
				makeContainerStatusPod("pod-crashloop", int64(0), "CrashLoopBackOff"),
				makeContainerStatusPod("pod-both", int64(3), "CrashLoopBackOff"),
				makeContainerStatusPod("pod-healthy", int64(1), ""),
			},
			expr:          `@.status.containerStatuses[?(@.restartCount>2 || @.state.waiting.reason=='CrashLoopBackOff')]`,
			expectedNames: []string{"pod-high-restart", "pod-crashloop", "pod-both"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := filterByJSONPath(tt.objs, tt.expr)
			if tt.expectError != "" {
				assert.ErrorContains(t, err, tt.expectError)
				return
			}
			require.NoError(t, err)
			names := make([]string, len(got))
			for i, obj := range got {
				names[i] = obj.GetName()
			}
			assert.ElementsMatch(t, tt.expectedNames, names)
		})
	}
}
