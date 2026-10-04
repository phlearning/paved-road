// Package commands implements the platformctl command line.
package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/phlearning/paved-road/cli/internal/kube"
	"github.com/phlearning/paved-road/cli/internal/workspace"
)

// Execute runs platformctl and exits with a non-zero code on failure.
func Execute() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "platformctl",
		Short:        "Self-service CLI of the paved-road platform",
		SilenceUsage: true,
	}
	root.AddCommand(newServiceCommand(), newStatusCommand(), newDoctorCommand(), newAskCommand(), newSealCommand())
	return root
}

func currentWorkspace() (*workspace.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return workspace.Find(cwd)
}

func clusterClient() (*kube.Client, error) {
	ws, err := currentWorkspace()
	if err != nil {
		return nil, err
	}
	return kube.New(ws.Kubeconfig())
}
