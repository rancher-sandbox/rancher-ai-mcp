package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCreateMCPResponse(t *testing.T) {
	tests := map[string]struct {
		objs           []*unstructured.Unstructured
		namespace      string
		cluster        string
		additionalInfo []string
		expected       string
		expectError    bool
	}{
		"single pod": {
			objs: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"name":      "test-pod",
							"namespace": "default",
						},
					},
				},
			},
			namespace:      "default",
			cluster:        "local",
			additionalInfo: []string{},
			expected:       `{"llm":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod","namespace":"default"}}],"uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"test-pod","type":"pod"}]}`,
			expectError:    false,
		},
		"single deployment": {
			objs: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name":      "test-deployment",
							"namespace": "default",
						},
					},
				},
			},
			namespace:      "default",
			cluster:        "local",
			additionalInfo: []string{},
			expected:       `{"llm":[{"apiVersion":"v1","kind":"Deployment","metadata":{"name":"test-deployment","namespace":"default"}}],"uiContext":[{"namespace":"default","kind":"Deployment","cluster":"local","name":"test-deployment","type":"apps.deployment"}]}`,
			expectError:    false,
		},
		"single pod with managedFields": {
			objs: []*unstructured.Unstructured{
				{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]any{
							"name":      "test-pod",
							"namespace": "default",
							"managedFields": map[string]any{
								"apiVersion": "v1",
								"fieldsType": "FieldsV1",
							},
						},
					},
				},
			},
			namespace:      "default",
			cluster:        "local",
			additionalInfo: []string{},
			expected:       `{"llm":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod","namespace":"default"}}],"uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"test-pod","type":"pod"}]}`,
			expectError:    false,
		},
		"multiple pods": {
			objs: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"name":      "test-pod-1",
							"namespace": "default",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"name":      "test-pod-2",
							"namespace": "default",
						},
					},
				},
			},
			namespace:      "default",
			cluster:        "local",
			additionalInfo: []string{},
			expected:       `{"llm":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod-1","namespace":"default"}},{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod-2","namespace":"default"}}],"uiContext":[{"namespace":"default","kind":"Pod","cluster":"local","name":"test-pod-1","type":"pod"},{"namespace":"default","kind":"Pod","cluster":"local","name":"test-pod-2","type":"pod"}]}`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := CreateMcpResponse(test.objs, test.cluster)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, test.expected, resp)
			}
		})
	}
}
