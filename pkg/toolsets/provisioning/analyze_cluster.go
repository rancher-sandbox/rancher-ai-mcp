package provisioning

import (
	"context"
	"fmt"
	"mcp/pkg/client"
	"mcp/pkg/converter"
	"mcp/pkg/response"
	"mcp/pkg/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type InspectClusterParams struct {
	Cluster   string `json:"cluster" jsonschema:"the name of the provisioning cluster"`
	Namespace string `json:"namespace" jsonschema:"the namespace of the resource, defaults to fleet-local if not set"`
}

// AnalyzeCluster returns a set of kubernetes resources that can be used to inspect the cluster for debugging and summary purposes.
func (t *Tools) AnalyzeCluster(ctx context.Context, toolReq *mcp.CallToolRequest, params InspectClusterParams) (*mcp.CallToolResult, any, error) {
	ns := params.Namespace
	if ns == "" {
		ns = "fleet-default"
		if params.Cluster == LocalCluster {
			ns = "fleet-local"
		}
	}

	log := utils.NewChildLogger(toolReq, map[string]string{
		"cluster":   params.Cluster,
		"namespace": ns,
	})

	log.Info("Analyzing cluster")

	provClusterResource, provCluster, err := t.getProvisioningCluster(ctx, toolReq, log, ns, params.Cluster)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error("failed to get provisioning cluster", zap.Error(err))
		return nil, nil, err
	}

	if apierrors.IsNotFound(err) {
		// the only cluster type without a provisioning cluster object is rke1, which is no longer supported.
		log.Warn("could not find provisioning cluster, likely an unsupported cluster type for tool")
		return nil, nil, fmt.Errorf("provisioning cluster %s not found in namespace %s", params.Cluster, ns)
	}

	var resources []*unstructured.Unstructured
	resources = append(resources, provClusterResource)

	// get the management cluster, its status may be relevant.
	// NB: Unlike the v1.Cluster object we can't directly import the v3.Cluster
	// since it pulls in a lot of indirect dependencies (operators for aks, eks, gke, etc.)
	managementClusterResource, err := t.client.GetResource(ctx, client.GetParams{
		Cluster: LocalCluster,
		Kind:    converter.ManagementClusterResourceKind,
		// Unlike provisioning clusters, management cluster objects are cluster scoped.
		Namespace: "",
		Name:      provCluster.Status.ClusterName,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error("failed to get management cluster", zap.Error(err))
		return nil, nil, err
	}
	if err == nil {
		resources = append(resources, managementClusterResource)
		log.Info("found management cluster", zap.String("managementCluster", provCluster.Status.ClusterName))
	}

	// get the CAPI cluster
	capiClusterResource, err := t.client.GetResourceAtAnyAPIVersion(ctx, client.GetParams{
		Cluster:   LocalCluster,
		Kind:      converter.CAPIClusterResourceKind,
		Namespace: "fleet-default",
		Name:      provCluster.Name,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error("failed to get CAPI cluster", zap.Error(err))
		return nil, nil, err
	}
	if err == nil {
		log.Info("found CAPI cluster")
		resources = append(resources, capiClusterResource)
	}

	// get all machine configs for node driver clusters.
	machineConfigs, err := t.getMachinePoolConfigs(ctx, toolReq, log, provCluster)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error("failed to get machine pool configs", zap.Error(err))
		return nil, nil, err
	}
	if err == nil && len(machineConfigs) > 0 {
		log.Info("found machine pool configs", zap.Int("count", len(machineConfigs)))
		resources = append(resources, machineConfigs...)
	}

	// get all the CAPI machine resources
	machines, machineSets, machineDeployments, err := t.getAllCAPIMachineResources(ctx, toolReq, log, getCAPIMachineResourcesParams{
		namespace:     "fleet-default",
		targetCluster: params.Cluster,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error("failed to lookup CAPI machines", zap.Error(err))
		return nil, nil, err
	}
	if err == nil {
		log.Info("found CAPI machines",
			zap.Int("machines", len(machines)),
			zap.Int("machineSets", len(machineSets)),
			zap.Int("machineDeployments", len(machineDeployments)))
	}

	if machines != nil && len(machines) > 0 {
		resources = append(resources, machines...)
	}
	if machineSets != nil && len(machineSets) > 0 {
		resources = append(resources, machineSets...)
	}
	if machineDeployments != nil && len(machineDeployments) > 0 {
		resources = append(resources, machineDeployments...)
	}

	mcpResponse, err := response.CreateMcpResponse(resources, LocalCluster)
	if err != nil {
		log.Error("failed to create mcp response", zap.Error(err))
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}
