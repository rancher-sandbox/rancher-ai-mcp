package response

import (
	"encoding/json"
	"fmt"
	"strings"

	"mcp/pkg/converter"

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
	LLM any `json:"llm"`
	// UIContext contains a list of resources so the UI can generate links to them
	UIContext []UIContext `json:"uiContext,omitempty"`
}

// CreateMcpResponse constructs an MCPResponse object. It takes a slice of unstructured Kubernetes objects, namespace, kind, cluster,
// and optional additional information strings. It marshals the response into a JSON string.
func CreateMcpResponse(objs []*unstructured.Unstructured, cluster string) (string, error) {
	var uiContext []UIContext
	for _, obj := range objs {
		unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")
		unstructured.RemoveNestedField(obj.Object, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")

		gvk := obj.GetObjectKind().GroupVersionKind()
		lowerKind := strings.ToLower(gvk.Kind)
		if lowerKind == "" {
			continue
		}

		// use prefixes to differentiate duplicate kinds from different API groups
		// (e.g. cluster.x-k8s.io.cluster vs provisioning.cattle.io.cluster)
		lookupKind := lowerKind
		steveType := lowerKind
		switch gvk.Group {
		case converter.CAPIGroup:
			lookupKind = converter.CAPIKindPrefix + lookupKind
		case converter.ProvisioningGroup:
			lookupKind = converter.ProvisioningKindPrefix + lookupKind
		case converter.ManagementGroup:
			lookupKind = converter.ManagementKindPrefix + lookupKind
		case converter.MachineConfigGroup:
			// machine configs are dynamically generated from node drivers
			// using their name, so we can't maintain a mapping for all of them.
			// fortunately, its highly unlikely there will be a conflict across groups
			// so we just use the group directly.
			steveType = gvk.Group + "." + lowerKind
		}

		if gvr, ok := converter.K8sKindsToGVRs[lookupKind]; ok && gvr.Group != "" {
			steveType = gvr.Group + "." + lowerKind
		}

		uiContext = append(uiContext, UIContext{
			Namespace: obj.GetNamespace(),
			Kind:      obj.GetKind(),
			Cluster:   cluster,
			Name:      obj.GetName(),
			Type:      steveType,
		})
	}

	resp := MCPResponse{
		UIContext: uiContext,
	}
	if len(objs) > 0 {
		resp.LLM = objs
	} else {
		resp.LLM = "no resources found"
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(bytes), nil
}
