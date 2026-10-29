// Package site - шаблоны и статика сайта, встроенные в команду сборки.
package site

import "embed"

//go:embed templates
var Templates embed.FS

//go:embed static
var Static embed.FS

// Presentation - презентация go-bpm, на сайте лежит по адресу /go-bpm.pdf (собирается make presentation).
//
//go:embed presentation/go-bpm.pdf
var Presentation []byte
