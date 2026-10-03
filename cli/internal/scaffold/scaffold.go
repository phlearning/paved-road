// Package scaffold renders a new service from the golden path templates.
package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

// Languages the golden path supports, one template directory each.
var Languages = []string{"python", "go"}

// A DNS-1123 label short enough to leave room for suffixes such as "-tls".
var namePattern = regexp.MustCompile(`^[a-z]([-a-z0-9]{0,38}[a-z0-9])?$`)

// Service is the data every template receives.
type Service struct {
	Name            string
	Language        string
	Owner           string
	ImageRepository string
}

// Validate checks a service name and language before anything is written.
func Validate(name, language string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid name %q: use lowercase letters, digits and dashes, start with a letter, at most 40 characters", name)
	}
	if !slices.Contains(Languages, language) {
		return fmt.Errorf("unsupported language %q: choose one of %s", language, strings.Join(Languages, ", "))
	}
	return nil
}

// Render writes the "common" templates then the language templates into
// dest, which must not exist yet. It returns the created files, relative to
// dest. On error, dest is removed so a retry starts clean.
func Render(templates fs.FS, svc Service, dest string) (files []string, err error) {
	if err := Validate(svc.Name, svc.Language); err != nil {
		return nil, err
	}
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("%s already exists", dest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = os.RemoveAll(dest)
		}
	}()

	for _, root := range []string{"common", svc.Language} {
		err := fs.WalkDir(templates, root, func(name string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel := strings.TrimSuffix(strings.TrimPrefix(name, root+"/"), ".tmpl")
			if err := renderFile(templates, name, svc, filepath.Join(dest, filepath.FromSlash(rel))); err != nil {
				return fmt.Errorf("render %s: %w", name, err)
			}
			files = append(files, rel)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	slices.Sort(files)
	return files, nil
}

func renderFile(templates fs.FS, name string, svc Service, target string) error {
	raw, err := fs.ReadFile(templates, name)
	if err != nil {
		return err
	}
	tmpl, err := template.New(path.Base(name)).
		Delims("[[", "]]").
		Option("missingkey=error").
		Parse(string(raw))
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, svc); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, out.Bytes(), 0o644)
}
