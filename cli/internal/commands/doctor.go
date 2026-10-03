package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/phlearning/paved-road/cli/internal/kube"
)

// Deployments every platform profile runs, as namespace/name.
var platformDeployments = []string{
	"kube-system/traefik",
	"kube-system/sealed-secrets-controller",
	"cert-manager/cert-manager",
	"argocd/argo-cd-argocd-server",
	"argocd/argo-cd-argocd-repo-server",
	"monitoring/kube-prometheus-stack-grafana",
	"monitoring/kube-prometheus-stack-operator",
}

type check struct {
	name string
	run  func(ctx context.Context, c *kube.Client) error
}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that the platform is healthy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clusterClient()
			if err != nil {
				return err
			}
			checks := []check{
				{"nodes are ready", checkNodes},
				{"platform deployments are available", checkDeployments},
				{"internal CA issues certificates", checkIssuer},
				{"ingress serves HTTPS with the internal CA", checkHTTPS},
			}

			failed := 0
			for _, c := range checks {
				ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
				err := c.run(ctx, client)
				cancel()
				if err != nil {
					failed++
					fmt.Fprintf(cmd.OutOrStdout(), "✗ %s: %v\n", c.name, err)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", c.name)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d check(s) failed", failed)
			}
			return nil
		},
	}
}

func checkNodes(ctx context.Context, c *kube.Client) error {
	nodes, err := c.Core.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(nodes.Items) == 0 {
		return errors.New("no nodes")
	}
	for _, n := range nodes.Items {
		if !nodeReady(n) {
			return fmt.Errorf("node %s is not ready", n.Name)
		}
	}
	return nil
}

func nodeReady(n corev1.Node) bool {
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

func checkDeployments(ctx context.Context, c *kube.Client) error {
	var notReady []string
	for _, ref := range platformDeployments {
		ns, name, _ := strings.Cut(ref, "/")
		d, err := c.Core.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil || d.Status.AvailableReplicas == 0 {
			notReady = append(notReady, ref)
		}
	}
	if len(notReady) > 0 {
		return fmt.Errorf("not available: %v", notReady)
	}
	return nil
}

func checkIssuer(ctx context.Context, c *kube.Client) error {
	ready, err := c.ClusterIssuerReady(ctx, "paved-road-ca")
	if err != nil {
		return err
	}
	if !ready {
		return errors.New("ClusterIssuer paved-road-ca is not ready")
	}
	return nil
}

// checkHTTPS calls Argo CD through Traefik and verifies the certificate
// against the root CA read from the cluster, as a browser trusting it would.
func checkHTTPS(ctx context.Context, c *kube.Client) error {
	secret, err := c.Core.CoreV1().Secrets("cert-manager").Get(ctx, "paved-road-root-ca", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("read root CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(secret.Data["ca.crt"]) {
		return errors.New("root CA secret has no valid ca.crt")
	}
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://argocd.localhost/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("https://argocd.localhost/healthz returned %d", resp.StatusCode)
	}
	return nil
}
