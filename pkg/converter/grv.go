package converter

import "k8s.io/apimachinery/pkg/runtime/schema"

var K8sKindsToGVRs = map[string]schema.GroupVersionResource{
	// --- CORE Kubernetes Resources (Group: "") ---
	"pod":                   {Group: "", Version: "v1", Resource: "pods"},
	"service":               {Group: "", Version: "v1", Resource: "services"},
	"configmap":             {Group: "", Version: "v1", Resource: "configmaps"},
	"secret":                {Group: "", Version: "v1", Resource: "secrets"},
	"event":                 {Group: "", Version: "v1", Resource: "events"},
	"namespace":             {Group: "", Version: "v1", Resource: "namespaces"},
	"node":                  {Group: "", Version: "v1", Resource: "nodes"},
	"serviceaccount":        {Group: "", Version: "v1", Resource: "serviceaccounts"},
	"persistentvolume":      {Group: "", Version: "v1", Resource: "persistentvolumes"},
	"persistentvolumeclaim": {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
	"resourcequota":         {Group: "", Version: "v1", Resource: "resourcequotas"},
	"limitrange":            {Group: "", Version: "v1", Resource: "limitranges"},

	// --- Apps Resources (Group: "apps") ---
	"deployment":  {Group: "apps", Version: "v1", Resource: "deployments"},
	"statefulset": {Group: "apps", Version: "v1", Resource: "statefulsets"},
	"daemonset":   {Group: "apps", Version: "v1", Resource: "daemonsets"},
	"replicaset":  {Group: "apps", Version: "v1", Resource: "replicasets"},

	// --- Batch Resources (Group: "batch") ---
	"job":     {Group: "batch", Version: "v1", Resource: "jobs"},
	"cronjob": {Group: "batch", Version: "v1", Resource: "cronjobs"},

	// --- Networking Resources (Group: "networking.k8s.io") ---
	"ingress":       {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	"networkpolicy": {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
	"ingressclass":  {Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"},

	// --- Autoscaling Resources (Group: "autoscaling") ---
	"horizontalpodautoscaler": {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
	"vpa":                     {Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}, // Note: VPA is separate group

	// --- RBAC Resources (Group: "rbac.authorization.k8s.io") ---
	"role":               {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
	"rolebinding":        {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
	"clusterrole":        {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
	"clusterrolebinding": {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},

	// --- Storage Resources (Group: "storage.k8s.io") ---
	"storageclass":     {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
	"volumeattachment": {Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"},

	// --- Custom Resource Definitions (Group: "apiextensions.k8s.io") ---
	"crd": {Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},

	// --- Discovery/Endpoint Resources (Group: "discovery.k8s.io") ---
	"endpointslices": {Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"},

	// --- Policy Resources (Group: "policy") ---
	"poddisruptionbudget": {Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"},

	// --- METRICS Resources (Group: "metrics.k8s.io") ---
	"node.metrics.k8s.io": {Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"},
	"pod.metrics.k8s.io":  {Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"},

	// --- RANCHER CORE Resources (Group: "management.cattle.io") ---
	"cluster":                    {Group: "management.cattle.io", Version: "v3", Resource: "clusters"},
	"project":                    {Group: "management.cattle.io", Version: "v3", Resource: "projects"},
	"user":                       {Group: "management.cattle.io", Version: "v3", Resource: "users"},
	"roletemplate":               {Group: "management.cattle.io", Version: "v3", Resource: "roletemplates"},
	"globalrole":                 {Group: "management.cattle.io", Version: "v3", Resource: "globalroles"},
	"globalrolebinding":          {Group: "management.cattle.io", Version: "v3", Resource: "globalrolebindings"},
	"clusterroletemplatebinding": {Group: "management.cattle.io", Version: "v3", Resource: "clusterroletemplatebindings"},
	"projectroletemplatebinding": {Group: "management.cattle.io", Version: "v3", Resource: "projectroletemplatebindings"},
	"nodetemplate":               {Group: "management.cattle.io", Version: "v3", Resource: "nodetemplates"},
	"nodedriver":                 {Group: "management.cattle.io", Version: "v3", Resource: "nodedrivers"},

	// --- RANCHER FLEET Resources (Group: "fleet.cattle.io") ---
	"bundle":           {Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "bundles"},
	"gitrepo":          {Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "gitrepos"},
	"bundledeployment": {Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "bundledeployments"},
	"clustergroup":     {Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "clustergroups"},
	"fleetcluster":     {Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "clusters"}, // Renamed to avoid collision with management.cattle.io/v3/clusters

	// --- RANCHER CATTLE Resources (Group: "cattle.io") ---
	"setting": {Group: "management.cattle.io", Version: "v3", Resource: "settings"},
}
