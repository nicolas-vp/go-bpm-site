// Команда build собирает из презентации go-bpm.md страницу HTML для печати в PDF: A4 альбомом,
// каждый раздел между --- — отдельный лист.
//
//	go run ./presentation/build -out /tmp/go-bpm.html
//	google-chrome --headless --no-pdf-header-footer --print-to-pdf=presentation/go-bpm.pdf /tmp/go-bpm.html
package main

import (
	"flag"
	"html"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolas-vp/go-bpm-site/internal/md"
)

func main() {
	src := flag.String("src", "presentation/go-bpm.md", "презентация в Markdown")
	out := flag.String("out", "presentation/go-bpm.html", "куда записать HTML")
	flag.Parse()

	b, err := os.ReadFile(*src)
	if err != nil {
		log.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Dir(*src))
	if err != nil {
		log.Fatal(err)
	}
	doc := md.Render(string(b), md.Options{})
	// разделы между <hr> — листы презентации
	sheets := strings.Split(doc.HTML, "<hr>\n")
	var body strings.Builder
	for i, s := range sheets {
		class := "sheet"
		if i == 0 {
			class += " cover"
		}
		body.WriteString(`<section class="` + class + `">` + s + "</section>\n")
	}
	page := `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<title>` + html.EscapeString(doc.Title) + `</title>
<base href="file://` + html.EscapeString(dir) + `/">
<style>` + css + `</style>
</head>
<body>
` + body.String() + `</body>
</html>
`
	if err := os.WriteFile(*out, []byte(page), 0o644); err != nil {
		log.Fatal(err)
	}
}

const css = `
@page { size: A4 landscape; margin: 14mm 16mm 12mm; }
* { box-sizing: border-box; }
html { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
body { margin: 0; font-family: Inter, "Segoe UI", "Noto Sans", Arial, sans-serif; font-size: 10.5pt; line-height: 1.45; color: #15212b; background: #ffffff; }
.sheet { break-after: page; }
.sheet:last-child { break-after: auto; }
h1 { font-size: 26pt; line-height: 1.15; margin: 0 0 6mm; color: #15212b; }
h2 { font-size: 19pt; margin: 0 0 4mm; padding-bottom: 2mm; border-bottom: 2px solid #0b7896; color: #15212b; }
h3 { font-size: 12.5pt; margin: 4mm 0 2mm; color: #0b7896; }
p { margin: 0 0 2.5mm; }
ul, ol { margin: 0 0 2.5mm; padding-left: 6mm; }
li { margin: 0 0 1mm; }
a { color: #0b7896; text-decoration: none; }
strong { color: #15212b; }
.anchor { display: none; }
img { display: block; max-width: 100%; max-height: 66mm; margin: 2mm auto 3mm; border-radius: 4px; }
img[src*="elements"], img[src*="cycle"] { max-height: 50mm; }
img[src$=".png"] { border: 1px solid #e1e3dd; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
code { font-family: "JetBrains Mono", Consolas, "DejaVu Sans Mono", monospace; font-size: 0.88em; background: #f4f5f1; color: #24303a; padding: 0.5px 3px; border-radius: 3px; }
.code { margin: 2mm 0 3mm; break-inside: avoid; }
.code pre { margin: 0; padding: 3mm 4mm; background: #f4f5f1; border: 1px solid #e1e3dd; color: #24303a; border-radius: 6px; font-size: 8.8pt; line-height: 1.4; white-space: pre-wrap; tab-size: 4; }
.code code { background: none; padding: 0; }
.table { margin: 2mm 0 3mm; }
table { width: 100%; border-collapse: collapse; font-size: 9pt; }
th, td { border: 1px solid #e1e3dd; padding: 1.6mm 2.4mm; vertical-align: top; text-align: left; }
th { background: #e2f2f6; color: #0b7896; }
tr:nth-child(even) td { background: #fbfbf9; }
tr { break-inside: avoid; }
td:first-child { font-weight: 600; }
.tok-keyword { color: #a3236e; } .tok-string { color: #2f7d32; } .tok-number, .tok-flag { color: #b35c00; }
.tok-comment { color: #7b858d; font-style: italic; } .tok-func, .tok-key, .tok-tag { color: #0b6a8a; }
.tok-builtin { color: #6a3fb3; } .tok-attr { color: #a3236e; }
.cover h1 { font-size: 32pt; margin-top: 4mm; }
.cover img[src*="logo"] { height: 16mm; margin: 0 0 6mm; }
.cover h1 { color: #0b7896; }
.cover h1 + p { font-size: 14pt; color: #44515c; max-width: 230mm; }
.cover img { max-height: 92mm; margin: 6mm auto; }
.cover ul { list-style: none; padding: 0; font-size: 11.5pt; }
`
