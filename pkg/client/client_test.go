package client

/*
import (
	"reflect"
	"sync"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"mcp/internal/tools/converter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

const (
	fakeUrl   = "https://localhost:8080"
	fakeToken = "token-xxx"
)

// helper to create a fake cluster object for tests.
func newFakeCluster(id, displayName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "management.cattle.io/v3",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name": id,
			},
			"spec": map[string]interface{}{
				"displayName": displayName,
			},
		},
	}
}

func TestGetClusterId(t *testing.T) {
	ctlr := gomock.NewController(t)

	// GVR for a Rancher Cluster
	clusterGVR := converter.K8sKindsToGVRs["cluster"]

	const (
		clusterID = "c-m-12345"
		clusterDN = "my-display-name"
	)

	// Define test cases
	tests := map[string]struct {
		clusterNameOrIDInput                 string
		mockClient                           func() resourceInterface
		expectedClusterIdsCache              map[string]interface{}
		expectedClustersDisplayNameToIDCache map[string]interface{}
		expectedID                           string
		expectErr                            string
	}{
		"should return clusterID if input is a clusterID": {
			clusterNameOrIDInput: clusterID,
			mockClient: func() resourceInterface {
				mock := mocks.NewMockK8sClient(ctlr)
				fakeClient := dynamicfake.NewSimpleDynamicClient(scheme(), newFakeCluster(clusterID, clusterDN)).Resource(clusterGVR)
				mock.EXPECT().GetResourceInterface(fakeToken, fakeUrl, "", "local", clusterGVR).Return(fakeClient, nil)

				return mock

			},
			expectedClusterIdsCache:              map[string]interface{}{clusterID: struct{}{}},
			expectedClustersDisplayNameToIDCache: map[string]interface{}{clusterDN: clusterID},
			expectedID:                           clusterID,
		},

		"should return clusterID if input is a cluster displayName": {
			clusterNameOrIDInput: clusterDN,
			mockClient: func() resourceInterface {
				mock := mocks.NewMockK8sClient(ctlr)
				fakeClient := dynamicfake.NewSimpleDynamicClient(scheme(), newFakeCluster(clusterID, clusterDN)).Resource(clusterGVR)
				mock.EXPECT().GetResourceInterface(fakeToken, fakeUrl, "", "local", clusterGVR).Return(fakeClient, nil)

				return mock

			},
			expectedClusterIdsCache:              map[string]interface{}{clusterID: struct{}{}},
			expectedClustersDisplayNameToIDCache: map[string]interface{}{clusterDN: clusterID},
			expectedID:                           clusterID,
		},

		"local": {
			clusterNameOrIDInput: "local",
			mockClient: func() resourceInterface {
				return mocks.NewMockK8sClient(ctlr)
			},
			expectedClusterIdsCache:              map[string]interface{}{},
			expectedClustersDisplayNameToIDCache: map[string]interface{}{},
			expectedID:                           "local",
		},

		"cluster not found": {
			clusterNameOrIDInput: clusterDN,
			mockClient: func() resourceInterface {
				mock := mocks.NewMockK8sClient(ctlr)
				fakeClient := dynamicfake.NewSimpleDynamicClient(scheme(), newFakeCluster(clusterID, "another cluster")).Resource(clusterGVR)
				mock.EXPECT().GetResourceInterface(fakeToken, fakeUrl, "", "local", clusterGVR).Return(fakeClient, nil)

				return mock

			},
			expectedClusterIdsCache:              map[string]interface{}{clusterID: struct{}{}},
			expectedClustersDisplayNameToIDCache: map[string]interface{}{"another cluster": clusterID},
			expectErr:                            "cluster 'my-display-name' not found",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clusterIdsCache = sync.Map{}
			clustersDisplayNameToIDCache = sync.Map{}

			clusterID, err := getClusterId(test.mockClient(), fakeToken, fakeUrl, test.clusterNameOrIDInput)

			if test.expectErr != "" {
				require.ErrorContains(t, err, test.expectErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedID, clusterID)
			assert.True(t, compareMapWithSyncMap(test.expectedClusterIdsCache, &clusterIdsCache))
			assert.True(t, compareMapWithSyncMap(test.expectedClustersDisplayNameToIDCache, &clustersDisplayNameToIDCache))
		})
	}
}

func compareMapWithSyncMap(standardMap map[string]interface{}, syncMap *sync.Map) bool {
	// extract contents of the sync.Map into a temporary map and count elements
	syncMapContents := make(map[string]interface{})
	syncMapCount := 0

	// Use Range for thread-safe iteration
	syncMap.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		syncMapContents[keyStr] = value
		syncMapCount++
		return true // continue iteration
	})

	standardMapCount := len(standardMap)

	if standardMapCount != syncMapCount {
		return false
	}

	return reflect.DeepEqual(standardMap, syncMapContents)
}

func scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)

	return scheme
}
*/
