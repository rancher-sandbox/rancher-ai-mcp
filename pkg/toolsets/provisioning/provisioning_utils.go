package provisioning

import (
	"context"
	"fmt"
	"strings"

	"mcp/pkg/client"
	"mcp/pkg/converter"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	provisioningV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	CAPIMachineDeploymentKind = "MachineDeployment"
	CAPIMachineSetKind        = "MachineSet"
	CAPIMachineKind           = "Machine"

	LocalCluster                     = "local"
	DefaultClusterResourcesNamespace = "fleet-default"
)

type getCAPIMachineResourcesParams struct {
	namespace     string
	targetCluster string
	machineName   string
}

func (t *Tools) getCAPIMachineResourcesByName(ctx context.Context, toolReq *mcp.CallToolRequest, log *zap.Logger, params getCAPIMachineResourcesParams) (*unstructured.Unstructured, *unstructured.Unstructured, *unstructured.Unstructured, error) {
	if params.namespace == "" {
		params.namespace = DefaultClusterResourcesNamespace
	}

	capiMachine, err := t.client.GetResourceAtAnyAPIVersion(ctx, client.GetParams{
		Cluster:   LocalCluster,
		Kind:      converter.CAPIMachineResourceKind,
		Namespace: params.namespace,
		Name:      params.machineName,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil, nil, apierrors.NewNotFound(schema.GroupResource{
				Group:    converter.CAPIGroup,
				Resource: CAPIMachineKind,
			}, params.machineName)
		}
		return nil, nil, nil, fmt.Errorf("failed to get machine: %w", err)
	}
	log.Info("found CAPI machine", zap.String("machine", params.machineName))

	var capiMachineSet, capiMachineDeployment *unstructured.Unstructured
	foundSetOwner := false
	for _, ownerRef := range capiMachine.GetOwnerReferences() {
		if ownerRef.Kind != CAPIMachineSetKind {
			continue
		}
		foundSetOwner = true
		capiMachineSet, err = t.client.GetResourceAtAnyAPIVersion(ctx, client.GetParams{
			Cluster:   LocalCluster,
			Kind:      converter.CAPIMachineSetResourceKind,
			Namespace: params.namespace,
			Name:      ownerRef.Name,
			URL:       toolReq.Extra.Header.Get(urlHeader),
			Token:     toolReq.Extra.Header.Get(tokenHeader),
		})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return capiMachine, nil, nil, nil
			}
			return nil, nil, nil, fmt.Errorf("failed to get machine set: %w", err)
		}
	}
	if !foundSetOwner || capiMachineSet == nil {
		return capiMachine, nil, nil, nil
	}
	log.Info("found CAPI machine set", zap.String("machine", params.machineName))

	foundDeploymentOwner := false
	for _, ownerRef := range capiMachineSet.GetOwnerReferences() {
		if ownerRef.Kind != CAPIMachineDeploymentKind {
			continue
		}
		foundDeploymentOwner = true
		capiMachineDeployment, err = t.client.GetResourceAtAnyAPIVersion(ctx, client.GetParams{
			Cluster:   LocalCluster,
			Kind:      converter.CAPIMachineDeploymentResourceKind,
			Namespace: params.namespace,
			Name:      ownerRef.Name,
			URL:       toolReq.Extra.Header.Get(urlHeader),
			Token:     toolReq.Extra.Header.Get(tokenHeader),
		})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return capiMachine, capiMachineSet, nil, nil
			}
			return nil, nil, nil, fmt.Errorf("failed to get machine deployment: %w", err)
		}
	}
	if !foundDeploymentOwner {
		return capiMachine, capiMachineSet, nil, nil
	}
	log.Info("found CAPI machine deployment", zap.String("machine", params.machineName))

	return capiMachineSet, capiMachineSet, capiMachineDeployment, nil
}

// getAllCAPIMachineResources retrieves the cluster API machines, machine sets, and machine deployments for a given provisioning cluster.
func (t *Tools) getAllCAPIMachineResources(ctx context.Context, toolReq *mcp.CallToolRequest, log *zap.Logger, params getCAPIMachineResourcesParams) ([]*unstructured.Unstructured, []*unstructured.Unstructured, []*unstructured.Unstructured, error) {
	if params.namespace == "" {
		params.namespace = DefaultClusterResourcesNamespace
	}

	clusterSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			"cluster.x-k8s.io/cluster-name": params.targetCluster,
		},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create machine selector for cluster machines")
	}

	var capiMachines, capiMachineSets, capiMachineDeployments []*unstructured.Unstructured
	deployments, err := t.client.GetResourcesAtAnyAPIVersion(ctx, client.ListParams{
		Cluster:       LocalCluster,
		Kind:          converter.CAPIMachineDeploymentResourceKind,
		Namespace:     params.namespace,
		LabelSelector: clusterSelector.String(),
		URL:           toolReq.Extra.Header.Get(urlHeader),
		Token:         toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, nil, nil, fmt.Errorf("failed to list machine deployments: %w", err)
	}
	if err == nil {
		capiMachineDeployments = deployments
	}

	machineSets, err := t.client.GetResourcesAtAnyAPIVersion(ctx, client.ListParams{
		Cluster:       LocalCluster,
		Kind:          converter.CAPIMachineSetResourceKind,
		Namespace:     params.namespace,
		LabelSelector: clusterSelector.String(),
		URL:           toolReq.Extra.Header.Get(urlHeader),
		Token:         toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, nil, nil, fmt.Errorf("failed to list machine sets: %w", err)
	}
	if err == nil {
		capiMachineSets = machineSets
	}

	machines, err := t.client.GetResourcesAtAnyAPIVersion(ctx, client.ListParams{
		Cluster:       LocalCluster,
		Kind:          converter.CAPIMachineResourceKind,
		Namespace:     params.namespace,
		LabelSelector: clusterSelector.String(),
		URL:           toolReq.Extra.Header.Get(urlHeader),
		Token:         toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, nil, nil, fmt.Errorf("failed to list machines: %w", err)
	}
	if err == nil {
		capiMachines = machines
	}

	return capiMachines, capiMachineSets, capiMachineDeployments, nil
}

func (t *Tools) getProvisioningCluster(ctx context.Context, toolReq *mcp.CallToolRequest, log *zap.Logger, ns, clusterName string) (*unstructured.Unstructured, provisioningV1.Cluster, error) {
	provisioningClusterResource, err := t.client.GetResource(ctx, client.GetParams{
		Cluster:   LocalCluster,
		Kind:      converter.ProvisioningClusterResourceKind,
		Namespace: ns,
		Name:      clusterName,
		URL:       toolReq.Extra.Header.Get(urlHeader),
		Token:     toolReq.Extra.Header.Get(tokenHeader),
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, provisioningV1.Cluster{}, apierrors.NewNotFound(schema.GroupResource{
				Group:    converter.ProvisioningGroup,
				Resource: "cluster",
			}, clusterName)
		}
		return nil, provisioningV1.Cluster{}, err
	}

	provCluster := provisioningV1.Cluster{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(provisioningClusterResource.Object, &provCluster)
	if err != nil {
		return nil, provCluster, err
	}

	return provisioningClusterResource, provCluster, nil
}

func (t *Tools) getMachinePoolConfigs(ctx context.Context, toolReq *mcp.CallToolRequest, log *zap.Logger, provCluster provisioningV1.Cluster) ([]*unstructured.Unstructured, error) {
	if provCluster.Spec.RKEConfig == nil || provCluster.Spec.RKEConfig.MachinePools == nil || len(provCluster.Spec.RKEConfig.MachinePools) == 0 {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    "rke-machine-config.cattle.io",
			Resource: "",
		}, provCluster.Name)
	}

	var resources []*unstructured.Unstructured
	pools := provCluster.Spec.RKEConfig.MachinePools
	for _, pool := range pools {
		config, err := t.client.GetResourceByGVR(ctx, client.GetParams{
			Cluster:   LocalCluster,
			Namespace: DefaultClusterResourcesNamespace,
			Name:      pool.NodeConfig.Name,
			URL:       toolReq.Extra.Header.Get(urlHeader),
			Token:     toolReq.Extra.Header.Get(tokenHeader),
		}, schema.GroupVersionResource{
			Group:    "rke-machine-config.cattle.io",
			Version:  "v1",
			Resource: fmt.Sprintf("%ss", strings.ToLower(pool.NodeConfig.GroupVersionKind().Kind)),
		})
		if apierrors.IsNotFound(err) {
			log.Debug("machine config not found for pool, skipping", zap.String("pool", pool.Name))
			continue
		}
		if err != nil {
			log.Error("failed to get machine config from pool", zap.Error(err))
			return nil, err
		}
		resources = append(resources, config)
	}
	return resources, nil
}
