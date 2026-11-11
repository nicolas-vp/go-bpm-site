package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	site "github.com/nicolas-vp/go-bpm-site"
	"github.com/nicolas-vp/go-bpm-site/internal/md"
)

// Английские страницы имеют собственные тексты и индекс поиска; адреса API и ресурсов общие.
var englishLabels = []string{
	"Overview", "Quick start", "How it works", "BPMN modeling", "DMN decisions", "Code generator",
	"AI agent in the modeler", "Demo walkthrough", "Example: account opening", "BPMN for LLM orchestration",
	"Testing processes", "Running and operating", "HTTP API and console", "Performance", "Astra Linux compatibility",
	"Roadmap", "About us",
}

func buildEnglish(c config, base layout) ([]string, error) {
	tpl, err := template.New("").Funcs(template.FuncMap{"indent": func(level int) bool { return level > 2 }, "contains": strings.Contains}).ParseFS(site.Templates, "templates/en/*.html")
	if err != nil {
		return nil, err
	}
	base.Lang = "en"
	base.Updated = strings.ReplaceAll(base.Updated, "МСК", "MSK")
	var paths []string
	var index []searchEntry
	var nav []navItem
	groups := map[string]string{"Начало работы": "Getting started", "Процессы и примеры": "Processes and examples", "Эксплуатация": "Operations", "О проекте": "About the project"}
	published := map[string]string{}
	for i, p := range pages {
		u := "/en" + pageURL(p.slug)
		nav = append(nav, navItem{URL: u, Label: englishLabels[i], Group: groups[p.group]})
		published[p.file] = u
	}
	anchors := map[string]bool{}
	var links []string
	for i, p := range pages {
		source := filepath.Join("content", "en", p.file)
		src, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		src, err = includeCode(src, filepath.Dir(filepath.Clean(c.docs)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		u := nav[i].URL
		doc := md.Render(string(src), md.Options{AnchorLabel: "Link to section", Link: func(dest string, image bool) string {
			result := rewriteLink(c, published, dest, image)
			if strings.HasPrefix(dest, "/") {
				result = dest
			}
			if !image && strings.HasPrefix(result, "/en/docs/") {
				links = append(links, result)
			}
			if !image && strings.HasPrefix(result, "#") {
				links = append(links, u+result)
			}
			return result
		}})
		if err := englishWidgets(c, p.file, &doc); err != nil {
			return nil, err
		}
		anchors[u] = true
		for _, h := range doc.Headings {
			anchors[u+"#"+h.ID] = true
		}
		for _, id := range allIDs(doc.HTML) {
			anchors[u+"#"+id] = true
		}
		pnav := append([]navItem(nil), nav...)
		pnav[i].Active = true
		dp := docPage{Title: doc.Title, Content: template.HTML(doc.HTML), Headings: doc.Headings, Nav: groupNavigation(pnav), EditURL: c.siteRepo + "/edit/main/" + filepath.ToSlash(source)}
		if i > 0 {
			dp.Prev = &nav[i-1]
		}
		if i+1 < len(nav) {
			dp.Next = &nav[i+1]
		}
		l := base
		l.Title, l.Section, l.Path, l.AlternatePath, l.Page = doc.Title, "docs", u, pageURL(p.slug), dp
		l.Description = description(string(src))
		if err := render(tpl, "doc.html", l, filepath.Join(c.out, u, "index.html")); err != nil {
			return nil, err
		}
		index = append(index, sections(u, doc)...)
		paths = append(paths, u)
	}
	for _, link := range links {
		if !anchors[link] {
			return nil, fmt.Errorf("английская документация: ссылка ведёт в никуда: %s", link)
		}
	}
	for _, p := range []struct{ section, title, description string }{
		{"home", "A BPMN engine for Go on PostgreSQL", "Turn BPMN diagrams and DMN decision tables into typed Go code, running on a PostgreSQL task queue inside your service."},
		{"demo", "Live demo", "Run BPMN processes, complete user tasks and follow instance history in the live go-bpm demo."},
		{"modeler", "BPMN and DMN modeler", "A desktop BPMN and DMN editor for go-bpm on Linux, Windows and macOS, with generator validation and an MCP server."},
	} {
		l := base
		l.Section, l.Title, l.Description = p.section, p.title, p.description
		l.AlternatePath = "/"
		if p.section != "home" {
			l.AlternatePath = "/" + p.section + "/"
		}
		l.Path = "/en" + l.AlternatePath
		if p.section == "home" {
			l.JSONLD, err = softwareLD(c, l.Description)
			if err != nil {
				return nil, err
			}
			l.JSONLD = template.JS(strings.Replace(string(l.JSONLD), `"url":"`+c.siteURL+`/"`, `"url":"`+c.siteURL+`/en/"`, 1))
		}
		if err := render(tpl, p.section+".html", l, filepath.Join(c.out, l.Path, "index.html")); err != nil {
			return nil, err
		}
		paths = append(paths, l.Path)
	}
	data, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}
	return paths, write(filepath.Join(c.out, "en", "search.json"), data)
}

// Значения и статусы берутся из общих данных; переводятся только подписи готовых виджетов.
func englishWidgets(c config, file string, doc *md.Doc) error {
	if file == "performance.md" {
		perf, err := loadPerf(filepath.Join("content", "performance-article.json"), c.benchResults)
		if err != nil {
			return err
		}
		doc.HTML, err = perf.render(doc.HTML)
		if err != nil {
			return err
		}
	}
	if file == "roadmap.md" {
		roadmap, err := loadRoadmap(filepath.Join("content", "roadmap.json"))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join("content", "en", "roadmap.json"))
		if err != nil {
			return err
		}
		var translated roadmapData
		if err = json.Unmarshal(data, &translated); err != nil {
			return err
		}
		if len(roadmap.Stages) != len(translated.Stages) {
			return fmt.Errorf("нужно обновить английский перевод этапов дорожной карты")
		}
		for i := range roadmap.Stages {
			s, t := &roadmap.Stages[i], translated.Stages[i]
			if s.Id != t.Id || len(s.Items) != len(t.Items) {
				return fmt.Errorf("нужно обновить английский перевод этапа %s", s.Id)
			}
			s.Heading, s.Short = t.Heading, t.Short
			for j := range s.Items {
				if s.Items[j].Title != t.Items[j].Source {
					return fmt.Errorf("нужно обновить английский перевод пункта %s", s.Items[j].Title)
				}
				s.Items[j].Title, s.Items[j].Text = t.Items[j].Title, t.Items[j].Text
				if s.Items[j].Doc != "" {
					s.Items[j].Doc = "/en" + strings.Split(s.Items[j].Doc, "#")[0]
				}
			}
		}
		doc.HTML, err = roadmap.render(doc.HTML, doc.Headings)
		if err != nil {
			return err
		}
	}
	if file == "performance.md" || file == "roadmap.md" {
		doc.HTML = strings.NewReplacer(
			"Экземпляров в секунду с историей или экспортом", "Instances per second with history or export",
			"Экземпляров в секунду без истории", "Instances per second without history",
			"С историей или экспортом", "With history or export", "Без истории", "Without history", "Система", "System",
			"без истории", "without history", "история в PostgreSQL", "history in PostgreSQL", "история", "history", "методы", "methods",
			"без экспорта", "without export", "экспорт в", "export to", "в памяти", "in memory", "(файл)", "(file)",
			" экз./с на 100 тыс.", " instances/s at 100k", ", медиана ", ", median of ", " прогонов, от ", " runs, from ", " после прогрева", " after warm-up",
			"; от ", "; from ", " до ", " to ", " тыс.", "k", " потоков", " threads",
			"; до и после прогонов база выдерживала от ", "; before and after runs, the database sustained from ",
			" коммитов в секунду с ожиданием диска", " disk-flushed commits per second",
			"Сложность ", "Effort ", " из ", " of ", "Сложность", "Effort", "Этап ", "Stage ",
			"В документации", "Documentation", "Сделано", "Done", "сделано", "done", "быстро", "quick", "долго", "lengthy",
		).Replace(doc.HTML)
	}
	return nil
}
