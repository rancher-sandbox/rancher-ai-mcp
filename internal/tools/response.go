package tools

import (
	"fmt"

	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// UIContext holds the contextual information for a Kubernetes resource. Used to build links by the UI.
type UIContext struct {
	// Namespace is the Kubernetes namespace where the resources are located.
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource"`
	// Kind is the type of the Kubernetes resource (e.g., "Pod", "Deployment").
	Kind string `json:"kind" jsonschema:"the kind of the resource"`
	// Cluster identifies the Rancher cluster where the resources reside.
	Cluster string `json:"cluster" jsonschema:"the cluster of the resource"`
	// Names is a slice of strings containing the names of the resources
	Names []string `json:"names" jsonschema:"the name of k8s resource"`
}

// MCPResponse represents the response returned by the MCP server
type MCPResponse struct {
	// LLM response to be sent to the LLM
	LLM string `json:"llm"`
	// UIContext contains a list of resources so the UI can generate links to them
	UIContext UIContext `json:"uiContext,omitempty"`
}

// createMcpResponse constructs an MCPResponse object. It takes a slice of unstructured Kubernetes objects, namespace, kind, cluster,
// and optional additional information strings. It marshals the response into a JSON string.
func createMcpResponse(objs []*unstructured.Unstructured, namespace string, kind string, cluster string, additionalInfo ...string) (string, error) {
	var names []string
	for _, obj := range objs {
		// Remove managedFields from each object to reduce payload size and remove irrelevant data for the LLM.
		removeManagedFieldsIfPresent(obj)
		if kind == obj.GetKind() {
			names = append(names, obj.GetName())
		}
	}

	llmResponse, err := json.Marshal(objs)
	if err != nil {
		return "", err
	}

	stringToSendToLLM := string(llmResponse)
	// Append any additional information provided to the LLM string.
	for _, str := range additionalInfo {
		stringToSendToLLM = stringToSendToLLM + "\n" + str
	}

	resp := MCPResponse{
		LLM: stringToSendToLLM,
		UIContext: UIContext{
			Namespace: namespace,
			Kind:      kind,
			Cluster:   cluster,
			Names:     names,
		},
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(bytes), nil
}

func removeManagedFieldsIfPresent(obj *unstructured.Unstructured) {
	metadata, ok := obj.Object["metadata"].(map[string]interface{})
	if !ok {
		// nothing to do
		return
	}
	delete(metadata, "managedFields")
}
