package tools

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"mcp/internal/tools/converter"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"mcp/internal/tools/k8s"
	"mcp/internal/tools/mocks"
)

const (
	fakeUrl   = "https://localhost:8080"
	fakeToken = "token-xxx"
)

func podUnstructured() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name": "rancher",
		},
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name":  "rancher-container",
					"image": "rancher:latest",
				},
			},
		},
	}}
}

func TestGetKubernetesResource(t *testing.T) {
	ctlr := gomock.NewController(t)

	tests := map[string]struct {
		params              ResourceParams
		mockResourceFetcher func() ResourceFetcher
		expectedResult      string
		expectedError       string
	}{
		"get pod": {
			params: ResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResourceFetcher: func() ResourceFetcher {
				mock := mocks.NewMockResourceFetcher(ctlr)
				mock.EXPECT().FetchK8sResource(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "pod",
					Namespace: "default",
					Name:      "rancher",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return(podUnstructured(), nil)

				return mock
			},
			expectedResult: `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"rancher\"},\"spec\":{\"containers\":[{\"image\":\"rancher:latest\",\"name\":\"rancher-container\"}]}}]","uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"rancher","type":"pod"}]}`,
		},
		"error fetching pod": {
			params: ResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResourceFetcher: func() ResourceFetcher {
				mock := mocks.NewMockResourceFetcher(ctlr)
				mock.EXPECT().FetchK8sResource(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "pod",
					Namespace: "default",
					Name:      "rancher",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return(nil, fmt.Errorf("unexpected error"))

				return mock
			},
			expectedError: "unexpected error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			tools := Tools{fetcher: test.mockResourceFetcher()}

			result, _, err := tools.GetResource(context.TODO(), &mcp.CallToolRequest{
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

func TestListKubernetesResource(t *testing.T) {
	ctlr := gomock.NewController(t)

	tests := map[string]struct {
		params              ListKubernetesResourcesParams
		mockResourceFetcher func() ResourceFetcher
		expectedResult      string
		expectedError       string
	}{
		"get pod list": {
			params: ListKubernetesResourcesParams{Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResourceFetcher: func() ResourceFetcher {
				mock := mocks.NewMockResourceFetcher(ctlr)
				mock.EXPECT().FetchK8sResources(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "pod",
					Namespace: "default",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return([]*unstructured.Unstructured{podUnstructured(), podUnstructured()}, nil)

				return mock
			},
			expectedResult: `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"rancher\"},\"spec\":{\"containers\":[{\"image\":\"rancher:latest\",\"name\":\"rancher-container\"}]}},{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"rancher\"},\"spec\":{\"containers\":[{\"image\":\"rancher:latest\",\"name\":\"rancher-container\"}]}}]","uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"rancher","type":"pod"},{"namespace":"default","kind":"Pod","cluster":"local","name":"rancher","type":"pod"}]}`,
		},
		"error fetching pod list": {
			params: ListKubernetesResourcesParams{Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResourceFetcher: func() ResourceFetcher {
				mock := mocks.NewMockResourceFetcher(ctlr)
				mock.EXPECT().FetchK8sResources(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "pod",
					Namespace: "default",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return(nil, fmt.Errorf("unexpected error"))

				return mock
			},
			expectedError: "unexpected error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			tools := Tools{fetcher: test.mockResourceFetcher()}

			result, _, err := tools.ListKubernetesResources(context.TODO(), &mcp.CallToolRequest{
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

func TestUpdateKubernetesResource(t *testing.T) {
	ctlr := gomock.NewController(t)
	patchData := []interface{}{
		map[string]interface{}{
			"op":    "replace",
			"path":  "/metadata/labels/foo",
			"value": "bar",
		},
	}

	tests := map[string]struct {
		params            UpdateKubernetesResourceParams
		mockClientCreator func() ClientCreator
		expectedResult    string
		expectedError     string
	}{
		"patch pod": {
			params: UpdateKubernetesResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local", Patch: patchData},
			mockClientCreator: func() ClientCreator {
				mockResourceInterface := mocks.NewMockResourceInterface(ctlr)
				patchBytes, _ := json.Marshal(patchData)
				mockResourceInterface.EXPECT().Patch(context.TODO(), "rancher", types.JSONPatchType, patchBytes, metav1.PatchOptions{}).Return(podUnstructured(), nil)

				mockClientCreator := mocks.NewMockClientCreator(ctlr)
				mockClientCreator.EXPECT().GetResourceInterface(fakeToken, fakeUrl, "default", converter.K8sKindsToGVRs[strings.ToLower("pod")]).Return(mockResourceInterface, nil)

				return mockClientCreator
			},
			expectedResult: `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"rancher\"},\"spec\":{\"containers\":[{\"image\":\"rancher:latest\",\"name\":\"rancher-container\"}]}}]","uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"rancher","type":"pod"}]}`,
		},
		"error patching pod": {
			params: UpdateKubernetesResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local", Patch: patchData},
			mockClientCreator: func() ClientCreator {
				mockResourceInterface := mocks.NewMockResourceInterface(ctlr)
				patchBytes, _ := json.Marshal(patchData)
				mockResourceInterface.EXPECT().Patch(context.TODO(), "rancher", types.JSONPatchType, patchBytes, metav1.PatchOptions{}).Return(nil, fmt.Errorf("unexpected error"))

				mockClientCreator := mocks.NewMockClientCreator(ctlr)
				mockClientCreator.EXPECT().GetResourceInterface(fakeToken, fakeUrl, "default", converter.K8sKindsToGVRs[strings.ToLower("pod")]).Return(mockResourceInterface, nil)

				return mockClientCreator
			},
			expectedError: "unexpected error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			tools := Tools{client: test.mockClientCreator()}

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

func TestCreateKubernetesResource(t *testing.T) {
	ctlr := gomock.NewController(t)

	tests := map[string]struct {
		params            CreateKubernetesResourceParams
		mockClientCreator func() ClientCreator
		expectedResult    string
		expectedError     string
	}{
		"create pod": {
			params: CreateKubernetesResourceParams{
				Name:      "rancher",
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Resource: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "rancher",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "rancher:latest",
								"name":  "rancher-container",
							},
						},
					},
				},
			},
			mockClientCreator: func() ClientCreator {
				mockResourceInterface := mocks.NewMockResourceInterface(ctlr)
				mockResourceInterface.EXPECT().Create(context.TODO(), podUnstructured(), metav1.CreateOptions{}).Return(podUnstructured(), nil)

				mockClientCreator := mocks.NewMockClientCreator(ctlr)
				mockClientCreator.EXPECT().GetResourceInterface(fakeToken, fakeUrl, "default", converter.K8sKindsToGVRs[strings.ToLower("pod")]).Return(mockResourceInterface, nil)

				return mockClientCreator
			},

			expectedResult: `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"rancher\"},\"spec\":{\"containers\":[{\"image\":\"rancher:latest\",\"name\":\"rancher-container\"}]}}]","uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"rancher","type":"pod"}]}`,
		},
		"error creating pod": {
			params: CreateKubernetesResourceParams{
				Name:      "rancher",
				Kind:      "pod",
				Namespace: "default",
				Cluster:   "local",
				Resource: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "rancher",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "rancher:latest",
								"name":  "rancher-container",
							},
						},
					},
				},
			},
			mockClientCreator: func() ClientCreator {
				mockClientCreator := mocks.NewMockClientCreator(ctlr)
				mockClientCreator.EXPECT().GetResourceInterface(fakeToken, fakeUrl, "default", converter.K8sKindsToGVRs[strings.ToLower("pod")]).Return(nil, fmt.Errorf("unexpected error"))

				return mockClientCreator
			},
			expectedError: "unexpected error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			tools := Tools{client: test.mockClientCreator()}

			result, _, err := tools.CreateKubernetesResource(context.TODO(), &mcp.CallToolRequest{
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

func TestInspectPod(t *testing.T) {
	ctlr := gomock.NewController(t)

	tests := map[string]struct {
		params              SpecificResourceParams
		mockClientCreator   func() ClientCreator
		mockResourceFetcher func() ResourceFetcher
		expectedResult      string
		expectedError       string
	}{
		"create pod": {
			params: SpecificResourceParams{
				Name:      "rancher",
				Namespace: "default",
				Cluster:   "local",
			},
			mockClientCreator: func() ClientCreator {
				mock := mocks.NewMockClientCreator(ctlr)
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rancher",
						Namespace: "default",
					},
				}
				mock.EXPECT().CreateClientSet(fakeToken, fakeUrl+"/k8s/clusters/local").Return(fake.NewClientset(pod), nil)

				return mock
			},
			mockResourceFetcher: func() ResourceFetcher {
				mock := mocks.NewMockResourceFetcher(ctlr)
				mock.EXPECT().FetchK8sResource(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "pod",
					Namespace: "default",
					Name:      "rancher",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return(&unstructured.Unstructured{Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "rancher",
						"ownerReferences": []interface{}{
							map[string]interface{}{
								"apiVersion": "apps/v1",
								"kind":       "ReplicaSet",
								"name":       "my-replicaset",
								"uid":        "uid",
							},
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "rancher-container",
								"image": "rancher:latest",
							},
						},
					},
				}}, nil)
				mock.EXPECT().FetchK8sResource(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "replicaset",
					Namespace: "default",
					Name:      "my-replicaset",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return(&unstructured.Unstructured{Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "ReplicaSet",
					"metadata": map[string]interface{}{
						"name": "rancher",
						"ownerReferences": []interface{}{
							map[string]interface{}{
								"apiVersion": "apps/v1",
								"kind":       "Deployment",
								"name":       "my-deployment",
								"uid":        "uid",
							},
						},
					},
				}}, nil)
				mock.EXPECT().FetchK8sResource(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "Deployment",
					Namespace: "default",
					Name:      "my-deployment",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return(&unstructured.Unstructured{Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "rancher",
					},
				}}, nil)
				mock.EXPECT().FetchK8sResource(k8s.FetchParams{
					Cluster:   "local",
					Kind:      "metrics.k8s.io.pods",
					Namespace: "default",
					Name:      "rancher",
					URL:       fakeUrl,
					Token:     fakeToken,
				}).Return(&unstructured.Unstructured{Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "PodMetrics",
					"metadata": map[string]interface{}{
						"name": "rancher",
					},
				}}, nil)

				return mock
			},

			expectedResult: `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"rancher\",\"ownerReferences\":[{\"apiVersion\":\"apps/v1\",\"kind\":\"ReplicaSet\",\"name\":\"my-replicaset\",\"uid\":\"uid\"}]},\"spec\":{\"containers\":[{\"image\":\"rancher:latest\",\"name\":\"rancher-container\"}]}},{\"apiVersion\":\"v1\",\"kind\":\"Deployment\",\"metadata\":{\"name\":\"rancher\"}},{\"apiVersion\":\"v1\",\"kind\":\"PodMetrics\",\"metadata\":{\"name\":\"rancher\"}},{\"pod-logs\":{\"rancher-container\":\"fake logs\"}}]","uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"rancher","type":"pod"},{"namespace":"default","kind":"Deployment","cluster":"local","name":"rancher","type":"apps.deployment"},{"namespace":"default","kind":"PodMetrics","cluster":"local","name":"rancher","type":"podmetrics"},{"namespace":"default","kind":"","cluster":"local","name":""}]}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			tools := Tools{
				client:  test.mockClientCreator(),
				fetcher: test.mockResourceFetcher(),
			}

			result, _, err := tools.InspectPod(context.TODO(), &mcp.CallToolRequest{
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
