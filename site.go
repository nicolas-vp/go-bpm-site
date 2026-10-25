// Package site - шаблоны и статика сайта, встроенные в команду сборки.
package site

import "embed"

//go:embed templates
var Templates embed.FS

//go:embed static
var Static embed.FS
