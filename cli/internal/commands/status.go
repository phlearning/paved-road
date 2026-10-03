package commands

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/phlearning/paved-road/cli/internal/kube"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status [NAME]",
		Short: "Show the delivery and runtime state of services",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clusterClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			apps, err := client.Applications(ctx)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				apps = filter(apps, args[0])
				if len(apps) == 0 {
					return fmt.Errorf("no service %q is deployed by Argo CD (is it pushed to the main branch?)", args[0])
				}
			}
			if len(apps) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No services yet. Create one with: platformctl new-service NAME")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "SERVICE\tSYNC\tHEALTH\tREADY\tREVISION\tURL")
			for _, app := range apps {
				ready := readyReplicas(ctx, client, app.Service)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\thttps://%s.localhost\n",
					app.Service, orUnknown(app.Sync), orUnknown(app.Health), ready, app.Revision, app.Service)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if len(args) == 1 && apps[0].Message != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "\nLast sync: %s\n", apps[0].Message)
			}
			return nil
		},
	}
}

func filter(apps []kube.Application, service string) []kube.Application {
	for _, app := range apps {
		if app.Service == service {
			return []kube.Application{app}
		}
	}
	return nil
}

// readyReplicas reads the Deployment the shared chart creates: same name as
// the service, in the namespace of the same name.
func readyReplicas(ctx context.Context, client *kube.Client, service string) string {
	deploy, err := client.Core.AppsV1().Deployments(service).Get(ctx, service, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return "-"
	}
	if err != nil {
		return "?"
	}
	return formatReady(deploy)
}

func formatReady(d *appsv1.Deployment) string {
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	return fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, desired)
}

func orUnknown(s string) string {
	if s == "" {
		return "Unknown"
	}
	return s
}
