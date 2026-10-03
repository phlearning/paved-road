// Package kube reads the platform state from the cluster.
package kube

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ServiceLabel is set by the ApplicationSet on every Application it creates.
const ServiceLabel = "paved-road.io/service"

const argoNamespace = "argocd"

var (
	applicationGVR   = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}
	clusterIssuerGVR = schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "clusterissuers"}
)

// Client bundles the typed and dynamic clients. Argo CD and cert-manager
// objects are read through the dynamic client to avoid their heavy modules.
type Client struct {
	Core    kubernetes.Interface
	Dynamic dynamic.Interface
}

// New builds a client from a kubeconfig path ("" uses client-go defaults).
func New(kubeconfig string) (*Client, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeconfig
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	core, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{Core: core, Dynamic: dyn}, nil
}

// Application is the subset of an Argo CD Application platformctl shows.
type Application struct {
	Service  string
	Sync     string
	Health   string
	Revision string
	Message  string
}

// Applications lists the Applications created by the golden path.
func (c *Client) Applications(ctx context.Context) ([]Application, error) {
	list, err := c.Dynamic.Resource(applicationGVR).Namespace(argoNamespace).
		List(ctx, metav1.ListOptions{LabelSelector: ServiceLabel})
	if err != nil {
		return nil, fmt.Errorf("list Argo CD applications: %w", err)
	}
	apps := make([]Application, 0, len(list.Items))
	for _, item := range list.Items {
		apps = append(apps, toApplication(item))
	}
	slices.SortFunc(apps, func(a, b Application) int { return strings.Compare(a.Service, b.Service) })
	return apps, nil
}

func toApplication(u unstructured.Unstructured) Application {
	str := func(fields ...string) string {
		v, _, _ := unstructured.NestedString(u.Object, fields...)
		return v
	}
	app := Application{
		Service:  u.GetLabels()[ServiceLabel],
		Sync:     str("status", "sync", "status"),
		Health:   str("status", "health", "status"),
		Revision: str("status", "sync", "revision"),
		Message:  str("status", "operationState", "message"),
	}
	if len(app.Revision) > 7 {
		app.Revision = app.Revision[:7]
	}
	return app
}

// ClusterIssuerReady reports whether a cert-manager ClusterIssuer is Ready.
func (c *Client) ClusterIssuerReady(ctx context.Context, name string) (bool, error) {
	issuer, err := c.Dynamic.Resource(clusterIssuerGVR).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	conditions, _, _ := unstructured.NestedSlice(issuer.Object, "status", "conditions")
	for _, raw := range conditions {
		cond, _ := raw.(map[string]any)
		if cond["type"] == "Ready" {
			return cond["status"] == "True", nil
		}
	}
	return false, nil
}
