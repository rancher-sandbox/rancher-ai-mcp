package provisioning

import (
	"context"
	"testing"

	"mcp/pkg/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestGetClusterMachine(t *testing.T) {
	tests := map[string]struct {
		params         GetClusterMachineParams
		fakeClientset  kubernetes.Interface
		fakeDynClient  *dynamicfake.FakeDynamicClient
		expectedResult string
		expectedError  string
	}{
		"get specific machine by name": {
			params: GetClusterMachineParams{
				Cluster:     "test-cluster",
				MachineName: "test-cluster-machine-1",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				newCAPIMachineWithBootstrap("test-cluster-machine-1", "fleet-default", "test-cluster", "Running", "test-cluster-machineset-1", "RKEBootstrap", "test-cluster-machine-1"),
				newCAPIMachine("test-cluster-machine-2", "fleet-default", "test-cluster", "Running", "test-cluster-machineset-1"),
			),
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
		"get machine that doesn't exist returns empty": {
			params: GetClusterMachineParams{
				Cluster:     "test-cluster",
				MachineName: "nonexistent-machine",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				newCAPIMachine("test-cluster-machine-1", "fleet-default", "test-cluster", "Running", "test-cluster-machineset-1"),
			),
			expectedResult: `{"llm":"no resources found"}`,
		},
		"get machines from cluster with no machines": {
			params: GetClusterMachineParams{
				Cluster:     "empty-cluster",
				MachineName: "",
			},
			fakeClientset:  newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient:  dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds()),
			expectedResult: `{"llm":"no resources found"}`,
		},
		"get machine without owner references": {
			params: GetClusterMachineParams{
				Cluster:     "standalone-cluster",
				MachineName: "standalone-machine",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				newCAPIMachine("standalone-machine", "fleet-default", "standalone-cluster", "", ""),
			),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "Machine",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "standalone-cluster"
							},
							"name": "standalone-machine",
							"namespace": "fleet-default"
						},
						"spec": {
							"clusterName": "standalone-cluster"
						}
					}
				],
				"uiContext": [
					{
						"cluster": "local",
						"kind": "Machine",
						"name": "standalone-machine",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machine"
					}
				]
			}`,
		},
		"get machine with machine set": {
			params: GetClusterMachineParams{
				Cluster:     "test-cluster",
				MachineName: "test-cluster-machine-3",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				newCAPIMachine("test-cluster-machine-3", "fleet-default", "test-cluster", "Running", "test-cluster-machineset-2"),
				newCAPIMachineSet("test-cluster-machineset-2", "fleet-default", "test-cluster", 3, 3, ""),
			),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "Machine",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machine-3",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineSet",
									"name": "test-cluster-machineset-2"
								}
							]
						},
						"spec": {
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
							"name": "test-cluster-machineset-2",
							"namespace": "fleet-default"
						},
						"spec": {
							"replicas": 3
						},
						"status": {
							"readyReplicas": 3,
							"replicas": 3
						}
					}
				],
				"uiContext": [
					{
						"cluster": "local",
						"kind": "Machine",
						"name": "test-cluster-machine-3",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machine"
					},
					{
						"cluster": "local",
						"kind": "MachineSet",
						"name": "test-cluster-machineset-2",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machineset"
					}
				]
			}`,
		},
		"get machine with machine set and machine deployment": {
			params: GetClusterMachineParams{
				Cluster:     "test-cluster",
				MachineName: "test-cluster-machine-4",
			},
			fakeClientset: newFakeClientsetWithCAPIDiscovery(),
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(capiMachineScheme(), capiCustomListKinds(),
				newCAPIMachine("test-cluster-machine-4", "fleet-default", "test-cluster", "Running", "test-cluster-machineset-3"),
				newCAPIMachineSet("test-cluster-machineset-3", "fleet-default", "test-cluster", 5, 5, "test-cluster-machinedeployment-1"),
				newCAPIMachineDeployment("test-cluster-machinedeployment-1", "fleet-default", "test-cluster", 5, 5),
			),
			expectedResult: `{
				"llm": [
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "Machine",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machine-4",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineSet",
									"name": "test-cluster-machineset-3"
								}
							]
						},
						"spec": {
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
							"name": "test-cluster-machineset-3",
							"namespace": "fleet-default",
							"ownerReferences": [
								{
									"apiVersion": "cluster.x-k8s.io/v1beta1",
									"controller": true,
									"kind": "MachineDeployment",
									"name": "test-cluster-machinedeployment-1"
								}
							]
						},
						"spec": {
							"replicas": 5
						},
						"status": {
							"readyReplicas": 5,
							"replicas": 5
						}
					},
					{
						"apiVersion": "cluster.x-k8s.io/v1beta1",
						"kind": "MachineDeployment",
						"metadata": {
							"labels": {
								"cluster.x-k8s.io/cluster-name": "test-cluster"
							},
							"name": "test-cluster-machinedeployment-1",
							"namespace": "fleet-default"
						},
						"spec": {
							"replicas": 5,
							"selector": {
								"matchLabels": {
									"cluster.x-k8s.io/cluster-name": "test-cluster"
								}
							}
						},
						"status": {
							"readyReplicas": 5,
							"replicas": 5
						}
					}
				],
				"uiContext": [
					{
						"cluster": "local",
						"kind": "Machine",
						"name": "test-cluster-machine-4",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machine"
					},
					{
						"cluster": "local",
						"kind": "MachineSet",
						"name": "test-cluster-machineset-3",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machineset"
					},
					{
						"cluster": "local",
						"kind": "MachineDeployment",
						"name": "test-cluster-machinedeployment-1",
						"namespace": "fleet-default",
						"type": "cluster.x-k8s.io.machinedeployment"
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

			result, _, err := tools.GetClusterMachine(context.TODO(), &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Name: "get-cluster-machine",
				},
				Extra: &mcp.RequestExtra{Header: map[string][]string{urlHeader: {testURL}, tokenHeader: {testToken}}},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)

				text, ok := result.Content[0].(*mcp.TextContent)
				assert.Truef(t, ok, "expected type *mcp.TextContent")

				assert.Truef(t, ok, "expected expectedResult to be a JSON string")
				assert.JSONEq(t, test.expectedResult, text.Text)
			}
		})
	}
}
