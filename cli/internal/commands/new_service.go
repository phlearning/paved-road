package commands

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/phlearning/paved-road/cli/internal/scaffold"
	"github.com/phlearning/paved-road/templates"
)

func newServiceCommand() *cobra.Command {
	var language, owner string

	cmd := &cobra.Command{
		Use:   "new-service NAME",
		Short: "Create a service from the golden path",
		Long: `Create apps/NAME with application code, tests, a Dockerfile and the
values for the shared chart. Once merged, the CI builds the image and Argo CD
deploys it to https://NAME.localhost with TLS, metrics and a dashboard.`,
		Example: "  platformctl new-service orders-api --lang go",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := currentWorkspace()
			if err != nil {
				return err
			}
			name := args[0]
			if owner == "" {
				owner = gitUserName()
			}
			svc := scaffold.Service{
				Name:            name,
				Language:        language,
				Owner:           owner,
				ImageRepository: ws.Config.Registry + "/" + name,
			}
			dest := filepath.Join(ws.AppsDir(), name)
			files, err := scaffold.Render(templates.FS, svc, dest)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Created apps/%s (%s):\n", name, language)
			for _, f := range files {
				fmt.Fprintf(out, "  %s\n", f)
			}
			fmt.Fprintf(out, `
Next steps:
  git add apps/%[1]s && git commit -m "Add %[1]s service" && git push
  platformctl status %[1]s
`, name)
			return nil
		},
	}
	cmd.Flags().StringVarP(&language, "lang", "l", "python", "language: "+strings.Join(scaffold.Languages, ", "))
	cmd.Flags().StringVar(&owner, "owner", "", "owning person or team (default: git user.name)")
	return cmd
}

func gitUserName() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
