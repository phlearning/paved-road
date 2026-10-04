package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const values = `# Deployment settings.
name: api

image:
  repository: ghcr.io/acme/api # pinned by CI
  tag: abc
`

func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "values.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) (string, map[string]any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	return string(raw), parsed
}

func TestSetAddsSecretsAndKeepsComments(t *testing.T) {
	path := write(t, values)
	if err := Set(path, "API_KEY", "AgB1"); err != nil {
		t.Fatal(err)
	}
	text, parsed := read(t, path)

	if got := parsed["secrets"].(map[string]any)["API_KEY"]; got != "AgB1" {
		t.Errorf("secrets.API_KEY = %v", got)
	}
	for _, comment := range []string{"# Deployment settings.", "# pinned by CI"} {
		if !strings.Contains(text, comment) {
			t.Errorf("comment %q lost:\n%s", comment, text)
		}
	}
}

func TestSetReplacesAnExistingKey(t *testing.T) {
	path := write(t, values+"secrets:\n  API_KEY: old\n  OTHER: keep\n")
	if err := Set(path, "API_KEY", "new"); err != nil {
		t.Fatal(err)
	}
	_, parsed := read(t, path)
	s := parsed["secrets"].(map[string]any)
	if s["API_KEY"] != "new" || s["OTHER"] != "keep" {
		t.Errorf("secrets = %v", s)
	}
}

func TestSetReplacesAnEmptySecretsMapping(t *testing.T) {
	path := write(t, values+"secrets: {}\n")
	if err := Set(path, "API_KEY", "x"); err != nil {
		t.Fatal(err)
	}
	_, parsed := read(t, path)
	if parsed["secrets"].(map[string]any)["API_KEY"] != "x" {
		t.Errorf("secrets = %v", parsed["secrets"])
	}
}
