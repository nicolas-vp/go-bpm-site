package main

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	site "github.com/nicolas-vp/go-bpm-site"
)

func TestLanguageNavigation(t *testing.T) {
	if len(englishLabels) != len(pages) {
		t.Fatal("для каждой страницы нужен английский пункт меню")
	}
	for _, lang := range []string{"ru", "en"} {
		t.Run(lang, func(t *testing.T) {
			pattern := "templates/*.html"
			current, alternate, linkLang := "/docs/api/", "/en/docs/api/", "en"
			if lang == "en" {
				pattern = "templates/en/*.html"
				current, alternate, linkLang = alternate, current, "ru"
			}
			tpl, err := template.New("").Funcs(template.FuncMap{"contains": strings.Contains, "indent": func(i int) bool { return i > 2 }}).ParseFS(site.Templates, pattern)
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(t.TempDir(), "index.html")
			l := layout{Lang: lang, Title: "HTTP API", Section: "docs", Path: current, SiteURL: "https://example.com", Page: docPage{}}
			if lang == "en" {
				l.AlternatePath = alternate
			}
			if err := render(tpl, "doc.html", l, file); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				`<html lang="` + lang + `">`,
				`<link rel="canonical" href="https://example.com` + current + `">`,
				`<link rel="alternate" hreflang="` + linkLang + `" href="https://example.com` + alternate + `">`,
				`class="language-switch" href="` + alternate + `" lang="` + linkLang + `"`,
			} {
				if !strings.Contains(string(data), want) {
					t.Errorf("нет %s", want)
				}
			}
			nav := strings.Split(strings.Split(string(data), `<nav id="main-nav"`)[1], `</nav>`)[0]
			if lang == "en" && strings.Contains(nav, `href="/docs/`) {
				t.Error("английская навигация ведёт в русский раздел")
			}
		})
	}
}
