package provisioning

import (
	"context"
	"testing"

	"mcp/pkg/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var fakeCAPIMachine = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "cluster.x-k8s.io/v1beta1",
		"kind":       "Machine",
		"metadata": map[string]interface{}{
			"name":      "test-cluster-machine-1",
			"namespace": "fleet-default",
			"labels": map[string]interface{}{
				"cluster.x-k8s.io/cluster-name": "test-cluster",
			},
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "cluster.x-k8s.io/v1beta1",
					"kind":       "MachineSet",
					"name":       "test-cluster-machineset-1",
					"controller": true,
				},
			},
		},
		"spec": map[string]interface{}{
			"clusterName": "test-cluster",
			"bootstrap": map[string]interface{}{
				"configRef": map[string]interface{}{
					"kind": "RKEBootstrap",
					"name": "test-cluster-machine-1",
				},
			},
		},
		"status": map[string]interface{}{
			"phase": "Running",
		},
	},
}

var fakeCAPIMachine2 = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "cluster.x-k8s.io/v1beta1",
		"kind":       "Machine",
		"metadata": map[string]interface{}{
			"name":      "test-cluster-machine-2",
			"namespace": "fleet-default",
			"labels": map[string]interface{}{
				"cluster.x-k8s.io/cluster-name": "test-cluster",
			},
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "cluster.x-k8s.io/v1beta1",
					"kind":       "MachineSet",
					"name":       "test-cluster-machineset-1",
					"controller": true,
				},
			},
		},
		"spec": map[string]interface{}{
			"clusterName": "test-cluster",
		},
		"status": map[string]interface{}{
			"phase": "Provisioning",
		},
	},
}

var fakeCAPIMachineSet = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "cluster.x-k8s.io/v1beta1",
		"kind":       "MachineSet",
		"metadata": map[string]interface{}{
			"name":      "test-cluster-machineset-1",
			"namespace": "fleet-default",
			"labels": map[string]interface{}{
				"cluster.x-k8s.io/cluster-name": "test-cluster",
			},
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "cluster.x-k8s.io/v1beta1",
					"kind":       "MachineDeployment",
					"name":       "test-cluster-md-0",
					"controller": true,
				},
			},
		},
		"spec": map[string]interface{}{
			"replicas": int64(2),
		},
		"status": map[string]interface{}{
			"replicas":      int64(2),
			"readyReplicas": int64(1),
		},
	},
}

var fakeCAPIMachineDeployment = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "cluster.x-k8s.io/v1beta1",
		"kind":       "MachineDeployment",
		"metadata": map[string]interface{}{
			"name":      "test-cluster-md-0",
			"namespace": "fleet-default",
			"labels": map[string]interface{}{
				"cluster.x-k8s.io/cluster-name": "test-cluster",
			},
		},
		"spec": map[string]interface{}{
			"replicas": int64(2),
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"cluster.x-k8s.io/cluster-name": "test-cluster",
				},
			},
		},
		"status": map[string]interface{}{
			"replicas":      int64(2),
			"readyReplicas": int64(1),
		},
	},
}

func TestAnalyzeClusterMachines(t *testing.T) {
	fakeUrl := "https://localhost:8080"
	fakeToken := "fakeToken"

	tests := map[string]struct {
		params         InspectClusterMachinesParams
		fakeClientset  kubernetes.Interface
		fakeDynClient  *dynamicfake.FakeDynamicClient
		expectedResult interface{} // can be string (JSON)
		expectedError  string
	}{
		"analyze cluster with machines, machine sets, and deployments": {
			params: InspectClusterMachinesParams{
				Cluster:   "test-cluster",
				Namespace: "fleet-default",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				fakeCAPIMachine, fakeCAPIMachine2, fakeCAPIMachineSet, fakeCAPIMachineDeployment),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "Machine",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machine-1",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineSet",
									"name": "test-cluster-machineset-1"
								}
							]
						},
						"spec": {
							"bootstrap": {
								"configRef": {
									"kind": "RKEBootstrap",
									"name": "test-cluster-machine-1"
								}
							},
							"clusterName": "test-cluster"
						},
						"status": {
							"phase": "Running"
						}
					},
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "Machine",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machine-2",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineSet",
									"name": "test-cluster-machineset-1"
								}
							]
						},
						"spec": {
							"clusterName": "test-cluster"
						},
						"status": {
							"phase": "Provisioning"
						}
					},
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "MachineSet",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machineset-1",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineDeployment",
									"name": "test-cluster-md-0"
								}
							]
						},
						"spec": {
							"replicas": 2
						},
						"status": {
							"readyReplicas": 1,
							"replicas": 2
						}
					},
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "MachineDeployment",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-md-0",
							"namespace": "fleet-default"
						},
						"spec": {
							"replicas": 2,
							"selector": {
								"matchLabels": {
									"cluster.x-k8s.io/cluster-name": "test-cluster"
								}
							}
						},
						"status": {
							"readyReplicas": 1,
							"replicas": 2
						}
					}
				],
				"uiContext": [
					{
						"cluster": "local",
						"kind": "Machine",
						"name": "test-cluster-machine-1",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machine"
					},
					{
						"cluster": "local",
						"kind": "Machine",
						"name": "test-cluster-machine-2",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machine"
					},
					{
						"cluster": "local",
						"kind": "MachineSet",
						"name": "test-cluster-machineset-1",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machineset"
					},
					{
						"cluster": "local",
						"kind": "MachineDeployment",
						"name": "test-cluster-md-0",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machinedeployment"
					}
				]
			}`,
		},
		"analyze cluster with no machines": {
			params: InspectClusterMachinesParams{
				Cluster:   "empty-cluster",
				Namespace: "fleet-default",
			},
			fakeClientset:  newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient:  dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds()),
			expectedResult: `{"llm":"no resources found"}`,
		},
		"analyze cluster with default namespace": {
			params: InspectClusterMachinesParams{
				Cluster:   "test-cluster",
				Namespace: "",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				fakeCAPIMachine, fakeCAPIMachineSet, fakeCAPIMachineDeployment),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "Machine",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machine-1",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineSet",
									"name": "test-cluster-machineset-1"
								}
							]
						},
						"spec": {
							"bootstrap": {
								"configRef": {
									"kind": "RKEBootstrap",
									"name": "test-cluster-machine-1"
								}
							},
							"clusterName": "test-cluster"
						},
						"status": {
							"phase": "Running"
						}
					},
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "MachineSet",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machineset-1",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineDeployment",
									"name": "test-cluster-md-0"
								}
							]
						},
						"spec": {
							"replicas": 2
						},
						"status": {
							"readyReplicas": 1,
							"replicas": 2
						}
					},
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "MachineDeployment",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-md-0",
							"namespace": "fleet-default"
						},
						"spec": {
							"replicas": 2,
							"selector": {
								"matchLabels": {
									"cluster.x-k8s.io/cluster-name": "test-cluster"
								}
							}
						},
						"status": {
							"readyReplicas": 1,
							"replicas": 2
						}
					}
				],
				"uiContext": [
					{
						"cluster": "local",
						"kind": "Machine",
						"name": "test-cluster-machine-1",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machine"
					},
					{
						"cluster": "local",
						"kind": "MachineSet",
						"name": "test-cluster-machineset-1",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machineset"
					},
					{
						"cluster": "local",
						"kind": "MachineDeployment",
						"name": "test-cluster-md-0",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machinedeployment"
					}
				]
			}`,
		},
		"analyze cluster with only machines (no sets or deployments, custom cluster)": {
			params: InspectClusterMachinesParams{
				Cluster:   "test-cluster",
				Namespace: "fleet-default",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				fakeCAPIMachine),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "Machine",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machine-1",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineSet",
									"name": "test-cluster-machineset-1"
								}
							]
						},
						"spec": {
							"bootstrap": {
								"configRef": {
									"kind": "RKEBootstrap",
									"name": "test-cluster-machine-1"
								}
							},
							"clusterName": "test-cluster"
						},
						"status": {
							"phase": "Running"
						}
					}
				],
				"uiContext": [
					{
						"cluster": "local",
						"kind": "Machine",
						"name": "test-cluster-machine-1",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machine"
					}
				]
			}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := &client.Client{
				ClientSetCreator: func(inConfig *rest.Config) (kubernetes.Interface, error) {
					return test.fakeClientset, nil
				},
				DynClientCreator: func(inConfig *rest.Config) (dynamic.Interface, error) {
					return test.fakeDynClient, nil
				},
			}
			tools := Tools{client: c}

			result, _, err := tools.AnalyzeClusterMachines(context.TODO(), &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Name: "analyze-cluster-machines",
				},
				Extra: &mcp.RequestExtra{Header: map[string][]string{urlHeader: {fakeUrl}, tokenHeader: {fakeToken}}},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				text, ok := result.Content[0].(*mcp.TextContent)
				assert.Truef(t, ok, "expected type *mcp.TextContent")
				expectedJSON, ok := test.expectedResult.(string)
				assert.Truef(t, ok, "expected expectedResult to be a JSON string")
				assert.JSONEq(t, expectedJSON, text.Text)
			}
		})
	}
}
