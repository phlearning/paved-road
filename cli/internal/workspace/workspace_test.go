package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWalksUpToTheRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "platform"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "registry: ghcr.io/acme/paved-road\nrepoURL: https://example.com/repo.git\nrevision: main\n"
	if err := os.WriteFile(filepath.Join(root, configPath), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "apps", "orders", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := Find(nested)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Root != root {
		t.Errorf("Root = %s, want %s", ws.Root, root)
	}
	if ws.Config.Registry != "ghcr.io/acme/paved-road" {
		t.Errorf("Registry = %q", ws.Config.Registry)
	}
}

func TestFindFailsOutsideARepository(t *testing.T) {
	if _, err := Find(t.TempDir()); err == nil {
		t.Fatal("want an error outside a paved-road repository")
	}
}

func TestKubeconfigPrefersTheEnvironment(t *testing.T) {
	ws := &Workspace{Root: t.TempDir()}
	t.Setenv("KUBECONFIG", "/custom/config")
	if got := ws.Kubeconfig(); got != "/custom/config" {
		t.Errorf("Kubeconfig() = %q", got)
	}
}
