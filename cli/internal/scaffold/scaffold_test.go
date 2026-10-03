package scaffold

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/phlearning/paved-road/templates"
)

func TestValidate(t *testing.T) {
	valid := []string{"api", "orders-api", "a1"}
	invalid := []string{"", "Api", "1api", "api-", "my_api", strings.Repeat("a", 41)}

	for _, name := range valid {
		if err := Validate(name, "go"); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range invalid {
		if err := Validate(name, "go"); err == nil {
			t.Errorf("Validate(%q) = nil, want an error", name)
		}
	}
	if err := Validate("api", "rust"); err == nil {
		t.Error("Validate with an unknown language should fail")
	}
}

func TestRenderEveryLanguage(t *testing.T) {
	for _, lang := range Languages {
		t.Run(lang, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "orders")
			svc := Service{Name: "orders", Language: lang, Owner: "team-a", ImageRepository: "ghcr.io/acme/orders"}

			files, err := Render(templates.FS, svc, dest)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Dockerfile", "README.md", "service.yaml", "values.yaml"} {
				if !slices.Contains(files, want) {
					t.Errorf("missing %s in %v", want, files)
				}
			}
			for _, f := range files {
				if strings.HasSuffix(f, ".tmpl") {
					t.Errorf("%s kept its .tmpl suffix", f)
				}
				body, _ := os.ReadFile(filepath.Join(dest, f))
				if strings.Contains(string(body), "[[") {
					t.Errorf("%s has unrendered placeholders", f)
				}
			}

			values, _ := os.ReadFile(filepath.Join(dest, "values.yaml"))
			if !strings.Contains(string(values), "repository: ghcr.io/acme/orders") {
				t.Errorf("values.yaml does not point at the image repository:\n%s", values)
			}
		})
	}
}

func TestRenderRefusesExistingDirectory(t *testing.T) {
	dest := t.TempDir()
	_, err := Render(templates.FS, Service{Name: "orders", Language: "go"}, dest)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("got %v, want an 'already exists' error", err)
	}
}
