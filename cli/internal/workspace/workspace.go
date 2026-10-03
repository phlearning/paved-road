// Package workspace locates the paved-road repository and its shared settings.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

const configPath = "platform/config.yaml"

// Config mirrors platform/config.yaml.
type Config struct {
	Registry string `json:"registry"`
	RepoURL  string `json:"repoURL"`
	Revision string `json:"revision"`
}

// Workspace is a checkout of the paved-road repository.
type Workspace struct {
	Root   string
	Config Config
}

// Find walks up from start until it finds the repository root, the way git
// finds .git, so platformctl works from any subdirectory.
func Find(start string) (*Workspace, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		raw, err := os.ReadFile(filepath.Join(dir, configPath))
		if err == nil {
			ws := &Workspace{Root: dir}
			if err := yaml.UnmarshalStrict(raw, &ws.Config); err != nil {
				return nil, fmt.Errorf("parse %s: %w", configPath, err)
			}
			return ws, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("not inside a paved-road repository (no %s found above %s)", configPath, start)
		}
		dir = parent
	}
}

// AppsDir is where services created by the golden path live.
func (w *Workspace) AppsDir() string {
	return filepath.Join(w.Root, "apps")
}

// Kubeconfig returns the kubeconfig to use: $KUBECONFIG when set, otherwise
// the one written by `make up`, otherwise "" (client-go defaults).
func (w *Workspace) Kubeconfig() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	local := filepath.Join(w.Root, ".kube", "config")
	if _, err := os.Stat(local); err == nil {
		return local
	}
	return ""
}
