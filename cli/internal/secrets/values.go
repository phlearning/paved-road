// Package secrets edits the sealed secrets of a service's values.yaml while
// keeping its comments and layout.
package secrets

import (
	"bytes"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Set writes secrets.KEY = ciphertext into the values file, creating the
// "secrets" mapping when needed and replacing an existing key in place.
func Set(path, key, ciphertext string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("%s is not a YAML mapping", path)
	}
	root := doc.Content[0]

	secrets := lookup(root, "secrets")
	if secrets == nil || secrets.Kind != yaml.MappingNode {
		fresh := &yaml.Node{Kind: yaml.MappingNode}
		if secrets != nil {
			*secrets = *fresh
		} else {
			root.Content = append(root.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "secrets", HeadComment: "Sealed with platformctl seal: only this cluster can decrypt them."},
				fresh)
		}
		secrets = lookup(root, "secrets")
	}

	value := &yaml.Node{Kind: yaml.ScalarNode, Value: ciphertext}
	if existing := lookup(secrets, key); existing != nil {
		*existing = *value
	} else {
		secrets.Content = append(secrets.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}

func lookup(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}
