// Команда build собирает статический сайт go-bpm: главная, документация из docs/ репозитория go-bpm, страница демо.
//
//	go run ./cmd/build -docs ../go-bpm/docs -out dist
//
// Документация не копируется в этот репозиторий: она пишется в go-bpm вместе с кодом и забирается на сборке.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	site "github.com/nicolas-vp/go-bpm-site"
	"github.com/nicolas-vp/go-bpm-site/internal/md"
)

// pages - страницы документации в порядке меню. README.md - обзор, /docs/. Страницы с site: true пишутся
// в этом репозитории (content/), остальные берутся из docs/ go-bpm.
//
// desc - описание для поисковиков (meta description, до 160 знаков). Пустое - берётся первый абзац страницы.
var pages = []struct {
	file, slug, label string
	site              bool
	desc              string
}{
	{"README.md", "", "Обзор", false,
		"Документация go-bpm — BPMN-движка для Go на PostgreSQL: процессы в BPMN, правила в DMN, генерация Go-кода, API, консоль, эксплуатация."},
	{"quickstart.md", "quickstart", "Быстрый старт", false,
		"Демо go-bpm за пять минут и свой сервис с нуля: процесс в Camunda Modeler, генерация Go-кода, движок на PostgreSQL."},
	{"concepts.md", "concepts", "Принципы работы", false,
		"Как go-bpm исполняет BPMN и DMN: токен, шаги и транзакции, очередь задач в PostgreSQL, повторы и переменные процесса."},
	{"bpmn.md", "bpmn", "Моделирование BPMN", false,
		"Какие элементы BPMN поддерживает go-bpm и как их настроить в Camunda Modeler: задачи, шлюзы, таймеры, подпроцессы, ограничения."},
	{"dmn.md", "dmn", "Решения DMN", false,
		"Таблицы решений DMN в go-bpm компилируются в Go-функции: ячейки, hit policy, история сработавших правил в PostgreSQL."},
	{"generator.md", "generator", "Генератор кода", false,
		"go-bpm-gen превращает BPMN и DMN в типизированный Go-код: запуск, раскладка пакетов, модель данных, методы и обработчики."},
	{"examples.md", "examples", "Разбор примеров", false,
		"Разбор процессов демо go-bpm: задачи пользователей, решения DMN, call activity, параллельные service task, таймеры и ошибки."},
	{"running.md", "running", "Запуск и эксплуатация", false,
		"Эксплуатация go-bpm: настройки движка, таблицы в PostgreSQL, масштабирование, мониторинг, повтор и отмена упавших экземпляров."},
	{"api.md", "api", "HTTP API и консоль", false,
		"HTTP API go-bpm и веб-консоль /ui/: процессы, экземпляры, задачи пользователей, упавшие задачи, термокарта и динамика."},
	{"testing.md", "testing", "Тестирование процессов", true,
		"Как тестировать процессы go-bpm обычным go test: юнит-тесты методов, условий и решений DMN, e2e на настоящем PostgreSQL."},
	{"performance.md", "performance", "Насколько быстрый go-bpm", true,
		"Нагрузочный тест go-bpm по методике статьи о Camunda и Zeebe: экземпляры в секунду с историей в PostgreSQL и без, на той же машине."},
	{"roadmap.md", "roadmap", "Дорожная карта", true,
		"Дорожная карта go-bpm: документация, метрики и авторизация, сообщения BPMN, multi-instance и компенсации — от простого к сложному."},
	{"about.md", "about", "О нас", true,
		"Кто разрабатывает go-bpm — BPMN-движок для Go на PostgreSQL — и как с нами связаться."},
}

// indexNowKey - ключ IndexNow: файл /<ключ>.txt подтверждает, что уведомления о новых страницах шлёт владелец сайта.
// Ключ не секрет, он и так лежит на сайте.
const indexNowKey = "5f3c9a2e8b7d4e1fa06c2b9d8e7f1a34"

type config struct {
	docs, out, repo, ref, commit string
	siteRepo, siteURL            string
	benchResults                 string
}

func main() {
	var c config
	flag.StringVar(&c.docs, "docs", "../go-bpm/docs", "каталог docs репозитория go-bpm")
	flag.StringVar(&c.out, "out", "dist", "куда собрать сайт")
	flag.StringVar(&c.repo, "repo", "https://github.com/nicolas-vp/go-bpm", "репозиторий go-bpm для ссылок на код")
	flag.StringVar(&c.ref, "ref", "main", "ветка go-bpm для ссылок на код")
	flag.StringVar(&c.commit, "commit", "", "коммит go-bpm, из которого взята документация (для подвала)")
	flag.StringVar(&c.benchResults, "bench-results", "bench/results", "результаты нагрузочного теста для страницы производительности")
	flag.StringVar(&c.siteURL, "site-url", "https://go-bpm.tech", "адрес сайта для canonical, Open Graph и sitemap")
	flag.StringVar(&c.siteRepo, "site-repo", "https://github.com/nicolas-vp/go-bpm-site", "репозиторий сайта для ссылок «править»")
	flag.Parse()
	if err := build(c); err != nil {
		log.Fatal(err)
	}
}

// navItem - пункт меню документации.
type navItem struct {
	URL, Label string
	Active     bool
}

// docPage - данные шаблона страницы документации.
type docPage struct {
	Title    string
	Content  template.HTML
	Headings []md.Heading
	Nav      []navItem
	Prev     *navItem
	Next     *navItem
	EditURL  string
}

// layout - общие данные всех страниц.
type layout struct {
	Title       string
	Description string
	Section     string // home, docs, demo - подсветка в шапке
	SiteURL     string
	Path        string // адрес страницы от корня, для canonical; пустой - страницы нет в поиске (404)
	JSONLD      template.JS
	Repo        string
	Commit      string
	CommitURL   string
	Year        int
	Page        any
}

// searchEntry - раздел страницы в поисковом индексе.
type searchEntry struct {
	URL   string `json:"u"`
	Page  string `json:"p"`
	Title string `json:"t"`
	Text  string `json:"x"`
}

func build(c config) error {
	tpl, err := template.New("").Funcs(template.FuncMap{
		"indent":   func(level int) bool { return level > 2 },
		"contains": strings.Contains,
	}).ParseFS(site.Templates, "templates/*.html")
	if err != nil {
		return err
	}
	if err := os.RemoveAll(c.out); err != nil {
		return err
	}
	base := layout{Repo: c.repo, Year: time.Now().Year(), SiteURL: c.siteURL}
	var sitemap []string
	if c.commit != "" {
		base.Commit = c.commit[:min(7, len(c.commit))]
		base.CommitURL = c.repo + "/commit/" + c.commit
	}

	published := map[string]string{} // файл docs -> адрес на сайте
	for _, p := range pages {
		published[p.file] = pageURL(p.slug)
	}

	var nav []navItem
	for _, p := range pages {
		nav = append(nav, navItem{URL: pageURL(p.slug), Label: p.label})
	}

	var (
		index   []searchEntry
		anchors = map[string]bool{} // адрес#якорь, которые есть на сайте
		links   []string            // внутренние ссылки для проверки
	)
	for i, p := range pages {
		srcFile, editURL := filepath.Join(c.docs, p.file), c.repo+"/edit/"+c.ref+"/docs/"+p.file
		if p.site {
			srcFile, editURL = filepath.Join("content", p.file), c.siteRepo+"/edit/main/content/"+p.file
		}
		src, err := os.ReadFile(srcFile)
		if err != nil {
			return err
		}
		if p.site {
			if src, err = includeCode(src, filepath.Dir(filepath.Clean(c.docs))); err != nil {
				return fmt.Errorf("%s: %w", srcFile, err)
			}
		}
		url := pageURL(p.slug)
		doc := md.Render(string(src), md.Options{Link: func(dest string, image bool) string {
			u := rewriteLink(c, published, dest, image)
			if strings.HasPrefix(u, "/docs/") && !image {
				links = append(links, p.file+" → "+u)
			}
			if strings.HasPrefix(u, "#") {
				links = append(links, p.file+" → "+url+u)
			}
			return u
		}})
		if p.file == "performance.md" {
			perf, err := loadPerf(filepath.Join("content", "performance-article.json"), c.benchResults)
			if err != nil {
				return err
			}
			if doc.HTML, err = perf.render(doc.HTML); err != nil {
				return fmt.Errorf("%s: %w", p.file, err)
			}
		}
		if p.file == "roadmap.md" {
			roadmap, err := loadRoadmap(filepath.Join("content", "roadmap.json"))
			if err != nil {
				return err
			}
			if doc.HTML, err = roadmap.render(doc.HTML, doc.Headings); err != nil {
				return fmt.Errorf("%s: %w", p.file, err)
			}
		}
		anchors[url] = true
		for _, h := range doc.Headings {
			anchors[url+"#"+h.ID] = true
		}
		for _, id := range allIDs(doc.HTML) {
			anchors[url+"#"+id] = true
		}
		index = append(index, sections(url, doc)...)

		pnav := make([]navItem, len(nav))
		copy(pnav, nav)
		pnav[i].Active = true
		dp := docPage{
			Title:    doc.Title,
			Content:  template.HTML(doc.HTML),
			Headings: doc.Headings,
			Nav:      pnav,
			EditURL:  editURL,
		}
		if i > 0 {
			dp.Prev = &nav[i-1]
		}
		if i+1 < len(nav) {
			dp.Next = &nav[i+1]
		}
		l := base
		l.Title, l.Section, l.Page, l.Path = doc.Title, "docs", dp, url
		l.Description = p.desc
		if l.Description == "" {
			l.Description = description(string(src))
		}
		sitemap = append(sitemap, url)
		if err := render(tpl, "doc.html", l, filepath.Join(c.out, url, "index.html")); err != nil {
			return err
		}
	}

	for _, link := range links {
		target := link[strings.Index(link, "→ ")+len("→ "):]
		if !anchors[target] {
			log.Printf("предупреждение: ссылка ведёт в никуда: %s", link)
		}
	}

	home := base
	home.Title, home.Section, home.Path = "BPMN-движок для Go на PostgreSQL", "home", "/"
	home.Description = "go-bpm — открытый BPMN-движок для Go: схемы BPMN и таблицы DMN превращаются в типизированный Go-код " +
		"и исполняются на очереди задач в PostgreSQL."
	if home.JSONLD, err = softwareLD(c, home.Description); err != nil {
		return err
	}
	if err := render(tpl, "home.html", home, filepath.Join(c.out, "index.html")); err != nil {
		return err
	}
	demo := base
	demo.Title, demo.Section, demo.Path = "Живое демо", "demo", "/demo/"
	demo.Description = "Живое демо go-bpm: запустите процесс BPMN, согласуйте заявку, посмотрите историю на схеме, уроните " +
		"экземпляр и почините его из консоли."
	if err := render(tpl, "demo.html", demo, filepath.Join(c.out, "demo", "index.html")); err != nil {
		return err
	}
	notFound := base
	notFound.Title = "Страница не найдена"
	if err := render(tpl, "404.html", notFound, filepath.Join(c.out, "404.html")); err != nil {
		return err
	}

	if err := writeSEO(c, append([]string{"/", "/demo/"}, sitemap...)); err != nil {
		return err
	}

	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if err := write(filepath.Join(c.out, "search.json"), data); err != nil {
		return err
	}
	if err := copyFS(site.Static, "static", filepath.Join(c.out, "static")); err != nil {
		return err
	}
	if err := write(filepath.Join(c.out, "go-bpm.pdf"), site.Presentation); err != nil {
		return err
	}
	return copyDir(filepath.Join(c.docs, "images"), filepath.Join(c.out, "docs", "images"))
}

func pageURL(slug string) string {
	if slug == "" {
		return "/docs/"
	}
	return "/docs/" + slug + "/"
}

// rewriteLink переводит ссылку из docs/*.md в адрес на сайте: страницы - на сайт, картинки - в /docs/images/,
// остальное в репозитории (код, context.md) - на GitHub.
func rewriteLink(c config, published map[string]string, dest string, image bool) string {
	if dest == "" || strings.HasPrefix(dest, "#") || strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:") {
		return dest
	}
	file, anchor, _ := strings.Cut(dest, "#")
	if anchor != "" {
		anchor = "#" + anchor
	}
	if u, ok := published[file]; ok {
		return u + anchor
	}
	p := path.Clean(path.Join("docs", file))
	if strings.HasPrefix(p, "docs/images/") {
		return "/" + p
	}
	if p == "." {
		return c.repo
	}
	if image {
		return c.repo + "/raw/" + c.ref + "/" + p
	}
	return c.repo + "/blob/" + c.ref + "/" + p + anchor
}

var (
	idRe  = regexp.MustCompile(` id="([^"]+)"`)
	tagRe = regexp.MustCompile(`<[^>]+>`)
)

func allIDs(h string) []string {
	var ids []string
	for _, m := range idRe.FindAllStringSubmatch(h, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

// sections режет страницу по заголовкам ## и ### для поиска.
func sections(url string, doc md.Doc) []searchEntry {
	h := doc.HTML
	var out []searchEntry
	add := func(id, title, body string) {
		text := strings.Join(strings.Fields(unescape(tagRe.ReplaceAllString(body, " "))), " ")
		if len([]rune(text)) > 600 {
			text = string([]rune(text)[:600])
		}
		u := url
		if id != "" {
			u += "#" + id
		}
		out = append(out, searchEntry{URL: u, Page: doc.Title, Title: title, Text: text})
	}
	prevID, prevTitle, prevStart := "", doc.Title, 0
	for _, hd := range doc.Headings {
		marker := `<h` + fmt.Sprint(hd.Level) + ` id="` + template.HTMLEscapeString(hd.ID) + `"`
		pos := strings.Index(h, marker)
		if pos < 0 {
			continue
		}
		add(prevID, prevTitle, h[prevStart:pos])
		prevID, prevTitle = hd.ID, hd.Text
		prevStart = pos + strings.Index(h[pos:], "</h") // заголовок не дублируем в тексте
	}
	add(prevID, prevTitle, h[prevStart:])
	return out
}

func unescape(s string) string {
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&", "&#34;", `"`, "&#39;", "'", "&quot;", `"`)
	return r.Replace(s)
}

// description - первый абзац страницы для <meta name="description">.
func description(src string) string {
	for _, block := range strings.Split(src, "\n\n") {
		t := strings.TrimSpace(block)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "|") || strings.HasPrefix(t, "!") ||
			strings.HasPrefix(t, "```") || strings.HasPrefix(t, ">") || strings.HasPrefix(t, "-") {
			continue
		}
		t = strings.Join(strings.Fields(t), " ")
		t = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`).ReplaceAllString(t, "$1")
		t = strings.NewReplacer("`", "", "**", "").Replace(t)
		return shorten(t, 160)
	}
	return ""
}

func render(tpl *template.Template, name string, data layout, file string) error {
	var b bytes.Buffer
	if err := tpl.ExecuteTemplate(&b, name, data); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return write(file, b.Bytes())
}

func write(file string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}

func copyFS(fsys fs.FS, root, dst string) error {
	return fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		return write(filepath.Join(dst, rel), data)
	})
}

// copyDir копирует картинки документации; исходники схем (src/*.bpmn) сайту не нужны.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "src" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		return write(filepath.Join(dst, rel), data)
	})
}

// shorten обрезает описание до max знаков: по концу предложения, если оно не слишком короткое, иначе по слову.
func shorten(t string, max int) string {
	r := []rune(t)
	if len(r) <= max {
		return t
	}
	cut := string(r[:max])
	if i := strings.LastIndexAny(cut, ".!?"); i >= max/2 {
		return cut[:i+1]
	}
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, ",;:—- ") + "…"
}

// softwareLD - разметка schema.org для главной: поисковики видят, что это открытая библиотека на Go.
func softwareLD(c config, desc string) (template.JS, error) {
	data, err := json.Marshal(map[string]any{
		"@context":            "https://schema.org",
		"@type":               "SoftwareSourceCode",
		"name":                "go-bpm",
		"description":         desc,
		"url":                 c.siteURL + "/",
		"codeRepository":      c.repo,
		"programmingLanguage": "Go",
		"license":             "https://www.apache.org/licenses/LICENSE-2.0",
		"keywords":            "BPMN, DMN, Go, Golang, PostgreSQL, workflow engine, BPM, Camunda",
		"author": []map[string]string{
			{"@type": "Person", "name": "Николай Первухин"},
			{"@type": "Person", "name": "Сергей Потехин"},
		},
	})
	return template.JS(data), err
}

// writeSEO пишет файлы для поисковиков: robots.txt, sitemap.xml, ключ IndexNow и иконки в корне сайта.
func writeSEO(c config, paths []string) error {
	robots := "User-agent: *\n" +
		"Disallow: /api/\n" +
		"Disallow: /search.json\n\n" +
		"Sitemap: " + c.siteURL + "/sitemap.xml\n"
	if err := write(filepath.Join(c.out, "robots.txt"), []byte(robots)); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "  <url><loc>%s</loc></url>\n", c.siteURL+p)
	}
	b.WriteString("</urlset>\n")
	if err := write(filepath.Join(c.out, "sitemap.xml"), []byte(b.String())); err != nil {
		return err
	}
	if err := write(filepath.Join(c.out, indexNowKey+".txt"), []byte(indexNowKey)); err != nil {
		return err
	}
	for _, f := range []string{"favicon.ico", "apple-touch-icon.png"} {
		data, err := fs.ReadFile(site.Static, "static/"+f)
		if err != nil {
			return err
		}
		if err := write(filepath.Join(c.out, f), data); err != nil {
			return err
		}
	}
	return nil
}
