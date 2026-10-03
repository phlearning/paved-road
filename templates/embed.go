// Package templates holds the service scaffolds of the golden path.
//
// Every file is a text/template with [[ ]] delimiters (so Helm and Go syntax
// pass through untouched) and a .tmpl suffix (so the Go toolchain ignores the
// scaffold sources). "common" is applied to every service, then the
// language directory.
package templates

import "embed"

//go:embed all:common all:python all:go
var FS embed.FS
