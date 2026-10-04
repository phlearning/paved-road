package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/phlearning/paved-road/cli/internal/secrets"
)

var envName = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func newSealCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "seal SERVICE KEY",
		Short: "Encrypt a secret for a service and store it in its values.yaml",
		Long: `Read a secret value from the terminal (hidden) or from standard input,
encrypt it with the cluster's Sealed Secrets key for this service only, and
write the ciphertext under "secrets:" in apps/SERVICE/values.yaml. The service
receives it as the environment variable KEY once the change is deployed.`,
		Example: "  platformctl seal rag-assistant ANTHROPIC_API_KEY\n  op read op://dev/anthropic/key | platformctl seal rag-assistant ANTHROPIC_API_KEY",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, key := args[0], args[1]
			if !envName.MatchString(key) {
				return fmt.Errorf("invalid key %q: use an environment variable name such as API_KEY", key)
			}
			ws, err := currentWorkspace()
			if err != nil {
				return err
			}
			valuesPath := filepath.Join(ws.AppsDir(), service, "values.yaml")
			if _, err := os.Stat(valuesPath); err != nil {
				return fmt.Errorf("no service %q: %w", service, err)
			}

			value, err := readSecret(cmd, key)
			if err != nil {
				return err
			}

			// kubeseal fetches the controller's public key and encrypts for
			// this secret name and namespace only ("strict" scope).
			seal := exec.CommandContext(cmd.Context(), "kubeseal", "--raw",
				"--scope", "strict", "--namespace", service, "--name", service+"-secrets")
			seal.Env = append(os.Environ(), "KUBECONFIG="+ws.Kubeconfig())
			seal.Stdin = bytes.NewReader(value)
			var stderr bytes.Buffer
			seal.Stderr = &stderr
			ciphertext, err := seal.Output()
			if err != nil {
				return fmt.Errorf("kubeseal: %v: %s", err, strings.TrimSpace(stderr.String()))
			}

			if err := secrets.Set(valuesPath, key, strings.TrimSpace(string(ciphertext))); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sealed %s into apps/%s/values.yaml. Commit and push to deploy it.\n", key, service)
			return nil
		},
	}
}

// readSecret reads without echo from a terminal, or the whole of stdin when piped.
func readSecret(cmd *cobra.Command, key string) ([]byte, error) {
	var value []byte
	var err error
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", key)
		value, err = term.ReadPassword(fd)
		fmt.Fprintln(cmd.ErrOrStderr())
	} else {
		value, err = io.ReadAll(cmd.InOrStdin())
	}
	if err != nil {
		return nil, err
	}
	value = bytes.TrimRight(value, "\r\n")
	if len(value) == 0 {
		return nil, errors.New("empty value")
	}
	return value, nil
}
