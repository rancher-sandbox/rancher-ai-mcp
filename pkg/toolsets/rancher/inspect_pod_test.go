package rancher

import (
	"context"
	"testing"

	"mcp/pkg/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

var fakePodForInspect = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "nginx-pod-abc123",
		Namespace: "default",
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       "nginx-replicaset",
				Controller: ptr.To(true),
			},
		},
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "nginx",
				Image: "nginx:1.21",
			},
			{
				Name:  "sidecar",
				Image: "busybox:latest",
			},
		},
	},
	Status: corev1.PodStatus{
		Phase: corev1.PodRunning,
	},
}

var fakeReplicaSet = &appsv1.ReplicaSet{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "nginx-replicaset",
		Namespace: "default",
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "nginx-deployment",
				Controller: ptr.To(true),
			},
		},
	},
	Spec: appsv1.ReplicaSetSpec{
		Replicas: ptr.To(int32(1)),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "nginx",
			},
		},
	},
}

var fakeDeploymentForInspect = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "nginx-deployment",
		Namespace: "default",
	},
	Spec: appsv1.DeploymentSpec{
		Replicas: ptr.To(int32(1)),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "nginx",
			},
		},
	},
}

func inspectPodScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

func TestInspectPod(t *testing.T) {
	fakeUrl := "https://localhost:8080"
	fakeToken := "fakeToken"

	tests := map[string]struct {
		params        specificResourceParams
		fakeDynClient *dynamicfake.FakeDynamicClient
		expectedError string
	}{
		// TODO add more cases
		"inspect pod - not found": {
			params: specificResourceParams{
				Name:      "nonexistent-pod",
				Namespace: "default",
				Cluster:   "local",
			},
			fakeDynClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(inspectPodScheme(), map[schema.GroupVersionResource]string{
				{Group: "", Version: "v1", Resource: "pods"}: "PodList",
			}),
			expectedError: `pods "nonexistent-pod" not found`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := &client.Client{
				DynClientCreator: func(inConfig *rest.Config) (dynamic.Interface, error) {
					return test.fakeDynClient, nil
				},
				ClientSetCreator: func(inConfig *rest.Config) (*kubernetes.Clientset, error) {
					return nil, nil
				},
			}
			tools := Tools{client: c}

			result, _, err := tools.InspectPod(context.TODO(), &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{Header: map[string][]string{urlHeader: {fakeUrl}, tokenHeader: {fakeToken}}},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
