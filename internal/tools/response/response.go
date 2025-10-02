package response

import (
	"fmt"
	"mcp/internal/tools/converter"
	"strings"

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
	// Name is a string containing the name of the resource.
	Name string `json:"name" jsonschema:"the name of k8s resource"`
	// Type is a string representing the resource type in steve
	Type string `json:"type,omitempty"`
}

// MCPResponse represents the response returned by the MCP server
type MCPResponse struct {
	// LLM response to be sent to the LLM
	LLM string `json:"llm"`
	// UIContext contains a list of resources so the UI can generate links to them
	UIContext []UIContext `json:"uiContext,omitempty"`
}

// CreateMcpResponse constructs an MCPResponse object. It takes a slice of unstructured Kubernetes objects, namespace, kind, cluster,
// and optional additional information strings. It marshals the response into a JSON string.
func CreateMcpResponse(objs []*unstructured.Unstructured, namespace string, cluster string) (string, error) {
	var uiContext []UIContext
	for _, obj := range objs {
		// Remove managedFields from each object to reduce payload size and remove irrelevant data for the LLM.
		removeManagedFieldsIfPresent(obj)
		lowerKind := strings.ToLower(obj.GetKind())
		steveType := lowerKind
		if gvr, ok := converter.K8sKindsToGVRs[lowerKind]; ok && gvr.Group != "" {
			steveType = gvr.Group + "." + lowerKind
		}
		uiContext = append(uiContext, UIContext{
			Namespace: namespace,
			Kind:      obj.GetKind(),
			Cluster:   cluster,
			Name:      obj.GetName(),
			Type:      steveType,
		})
	}

	llmResponse := "no resources found"
	if len(objs) > 0 {
		llmResponseBytes, err := json.Marshal(objs)
		if err != nil {
			return "", err
		}
		llmResponse = string(llmResponseBytes)
	}

	resp := MCPResponse{
		LLM:       llmResponse,
		UIContext: uiContext,
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
