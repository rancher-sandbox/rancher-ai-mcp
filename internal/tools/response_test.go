package tools

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestCreateMCPResponse(t *testing.T) {
	tests := map[string]struct {
		name           string
		objs           []*unstructured.Unstructured
		namespace      string
		kind           string
		cluster        string
		additionalInfo []string
		expected       string
		expectError    bool
	}{
		"single pod": {
			name: "should create response for a single pod",
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
			kind:           "Pod",
			cluster:        "local",
			additionalInfo: []string{},
			expected:       `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"test-pod\",\"namespace\":\"default\"}}]","uiContext":{"namespace":"default","kind":"Pod","cluster":"local","names":["test-pod"]}}`,
			expectError:    false,
		},
		"single pod with managedFields": {
			name: "should create response for a single pod",
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
			kind:           "Pod",
			cluster:        "local",
			additionalInfo: []string{},
			expected:       `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"test-pod\",\"namespace\":\"default\"}}]","uiContext":{"namespace":"default","kind":"Pod","cluster":"local","names":["test-pod"]}}`,
			expectError:    false,
		},
		"multiple pods": {
			name: "should create response for multiple pods",
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
			kind:           "Pod",
			cluster:        "local",
			additionalInfo: []string{},
			expected:       `{"llm":"[{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"test-pod-1\",\"namespace\":\"default\"}},{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"test-pod-2\",\"namespace\":\"default\"}}]","uiContext":{"namespace":"default","kind":"Pod","cluster":"local","names":["test-pod-1","test-pod-2"]}}`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := createMcpResponse(test.objs, test.namespace, test.kind, test.cluster)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, test.expected, resp)
			}
		})
	}
}
