package rancher

import (
	"context"
	"testing"

	"mcp/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

var fakePod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rancher",
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "rancher-container",
				Image: "rancher:latest",
			},
		},
	},
}

func scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	return scheme
}
func TestGetResource(t *testing.T) {
	fakeUrl := "https://localhost:8080"
	fakeToken := "fakeToken"

	tests := map[string]struct {
		params         resourceParams
		fakeDynClient  *dynamicfake.FakeDynamicClient
		expectedResult string
		expectedError  string
	}{
		"get pod": {
			params:         resourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			fakeDynClient:  dynamicfake.NewSimpleDynamicClient(scheme(), fakePod),
			expectedResult: `{"llm":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"rancher","namespace":"default"},"spec":{"containers":[{"image":"rancher:latest","name":"rancher-container","resources":{}}]},"status":{}}],"uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"rancher","type":"pod"}]}`,
		},
		"get pod - not found": {
			params:        resourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			fakeDynClient: dynamicfake.NewSimpleDynamicClient(scheme()),
			expectedError: `pods "rancher" not found`,
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
