package main

// Дорожная карта: этапы и задачи - в content/roadmap.json, страница - content/roadmap.md с подстановками:
//
//	@@roadmap:stages@@ - обзор этапов и шкала сложности
//	@@roadmap:<id>@@   - карточки задач этапа
//
// Заголовки этапов остаются в Markdown (оглавление, якоря), обзор ссылается на них по тексту заголовка.

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"

	"github.com/nicolas-vp/go-bpm-site/internal/md"
)

type roadmapItem struct {
	Icon  string `json:"icon"`
	Level int    `json:"level"` // сложность: 1 - быстро, 5 - долго
	Title string `json:"title"`
	Text  string `json:"text"`
}

type roadmapStage struct {
	Id      string        `json:"id"`
	Heading string        `json:"heading"` // текст заголовка ## в roadmap.md
	Short   string        `json:"short"`
	Icon    string        `json:"icon"`
	Items   []roadmapItem `json:"items"`
}

type roadmapData struct {
	Stages []roadmapStage `json:"stages"`
}

const roadmapLevels = 5

// roadmapIcons - иконки 24×24 в стиле сайта: линии currentColor, без заливки.
var roadmapIcons = map[string]string{
	"book":      `<path d="M5 4.5A1.5 1.5 0 0 1 6.5 3H19v15H6.5A1.5 1.5 0 0 0 5 19.5z"/><path d="M5 19.5A1.5 1.5 0 0 0 6.5 21H19"/>`,
	"gear":      `<circle cx="12" cy="12" r="3"/><path d="M12 3v2.5M12 18.5V21M3 12h2.5M18.5 12H21M5.6 5.6l1.8 1.8M16.6 16.6l1.8 1.8M5.6 18.4l1.8-1.8M16.6 7.4l1.8-1.8"/>`,
	"puzzle":    `<rect x="3" y="13" width="8" height="8" rx="1.5"/><circle cx="17" cy="17" r="4"/><path d="M12 3l4.5 7.5h-9z"/>`,
	"mountain":  `<path d="M2.5 20l6.5-11 4 6.5 2.5-4 6 8.5z"/><path d="M7.3 12l1.7 1.2 1.7-1.2"/>`,
	"repeat":    `<path d="M17 3l3 3-3 3"/><path d="M4 11V9a3 3 0 0 1 3-3h13"/><path d="M7 21l-3-3 3-3"/><path d="M20 13v2a3 3 0 0 1-3 3H4"/>`,
	"hourglass": `<path d="M6 3h12M6 21h12"/><path d="M7 3v3l5 6 5-6V3M7 21v-3l5-6 5 6v3"/>`,
	"shield":    `<path d="M12 3l7 3v5c0 4.5-3 8-7 10-4-2-7-5.5-7-10V6z"/><path d="M9 12l2 2 4-4"/>`,
	"branch":    `<circle cx="6" cy="5" r="2"/><circle cx="6" cy="19" r="2"/><circle cx="18" cy="7" r="2"/><path d="M6 7v10"/><path d="M18 9c0 4.5-6 4-11.2 8.3"/>`,
	"layers":    `<path d="M12 3l9 5-9 5-9-5z"/><path d="M3 12.5l9 5 9-5"/><path d="M3 16.5l9 5 9-5"/>`,
	"migrate":   `<path d="M4 8h15"/><path d="M15 4l4 4-4 4"/><path d="M20 16H5"/><path d="M9 12l-4 4 4 4"/>`,
	"chart":     `<path d="M4 4v16h16"/><path d="M8.5 16v-4M12.5 16V8M16.5 16v-6"/>`,
	"key":       `<circle cx="8" cy="15" r="4"/><path d="M11 12l9-9M16 7l2.5 2.5M13.5 9.5l2 2"/>`,
	"journal":   `<rect x="5" y="3" width="14" height="18" rx="2"/><path d="M9 8h6M9 12h6M9 16h3.5"/>`,
	"broom":     `<path d="M4 7h16M10 7V4h4v3"/><path d="M6 7l1 13h10l1-13"/><path d="M10 11v5.5M14 11v5.5"/>`,
	"plug":      `<path d="M9 3v5M15 3v5"/><path d="M7 8h10v3a5 5 0 0 1-10 0z"/><path d="M12 16v5"/>`,
	"badge":     `<path d="M12 3l8 4.5v9L12 21l-8-4.5v-9z"/><path d="M4 7.5l8 4.5 8-4.5M12 12v9"/>`,
	"mail":      `<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3.5 7l8.5 6 8.5-6"/>`,
	"alarm":     `<circle cx="12" cy="13" r="7"/><path d="M12 9.5V13l2.5 2"/><path d="M5 3.5L2.5 6M19 3.5L21.5 6"/>`,
	"bell":      `<path d="M6 16v-5a6 6 0 0 1 12 0v5l1.5 2h-15z"/><path d="M10 20.5a2 2 0 0 0 4 0"/>`,
	"calendar":  `<rect x="3.5" y="5" width="17" height="15" rx="2"/><path d="M3.5 10h17M8 3v4M16 3v4"/>`,
	"database":  `<ellipse cx="12" cy="6" rx="7" ry="3"/><path d="M5 6v12c0 1.7 3.1 3 7 3s7-1.3 7-3V6"/><path d="M5 12c0 1.7 3.1 3 7 3s7-1.3 7-3"/>`,
	"gauge":     `<path d="M4 17.5a8 8 0 1 1 16 0"/><path d="M12 17.5l3.8-5.5"/><circle cx="12" cy="17.5" r="1"/>`,
	"stack":     `<rect x="8" y="3" width="13" height="13" rx="2"/><path d="M5 7v11a2 2 0 0 0 2 2h11"/>`,
	"undo":      `<path d="M9 14L4 9l5-5"/><path d="M4 9h11a5 5 0 0 1 0 10h-3"/>`,
	"split":     `<path d="M12 21v-8"/><path d="M12 13L5.5 6.5M12 13l6.5-6.5"/><path d="M5 11V6h5M19 11V6h-5"/>`,
	"bolt":      `<path d="M13 2.5L4.5 14H11l-1 7.5L18.5 10H12z"/>`,
	"import":    `<path d="M12 3v12"/><path d="M7 10l5 5 5-5"/><path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2"/>`,
}

func loadRoadmap(file string) (*roadmapData, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var r roadmapData
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	for _, s := range r.Stages {
		if _, ok := roadmapIcons[s.Icon]; !ok {
			return nil, fmt.Errorf("%s: этап %s: нет иконки %q", file, s.Id, s.Icon)
		}
		for _, it := range s.Items {
			if _, ok := roadmapIcons[it.Icon]; !ok {
				return nil, fmt.Errorf("%s: %s: нет иконки %q", file, it.Title, it.Icon)
			}
			if it.Level < 1 || it.Level > roadmapLevels {
				return nil, fmt.Errorf("%s: %s: сложность %d не от 1 до %d", file, it.Title, it.Level, roadmapLevels)
			}
		}
	}
	return &r, nil
}

var roadmapRe = regexp.MustCompile(`@@roadmap:([a-z]+)@@`)

// render заменяет подстановки в HTML страницы; headings - заголовки страницы, на них ссылается обзор этапов.
func (r *roadmapData) render(page string, headings []md.Heading) (string, error) {
	anchors := map[string]string{}
	for _, h := range headings {
		anchors[h.Text] = h.ID
	}
	var firstErr error
	fail := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}
	used := map[string]bool{}
	out := roadmapRe.ReplaceAllStringFunc(page, func(m string) string {
		key := roadmapRe.FindStringSubmatch(m)[1]
		if key == "stages" {
			s, err := r.overview(anchors)
			if err != nil {
				fail(err)
			}
			return s
		}
		for _, s := range r.Stages {
			if s.Id == key {
				used[key] = true
				return r.cards(s)
			}
		}
		fail(fmt.Errorf("нет этапа %s", key))
		return m
	})
	for _, s := range r.Stages {
		if !used[s.Id] {
			fail(fmt.Errorf("этап %s не выведен на странице", s.Id))
		}
	}
	// подстановка-блок стоит отдельным абзацем: <p> вокруг не нужен
	out = regexp.MustCompile(`<p>(<div class="rm-[^"]+"[\s\S]*?</div><!--rm-->)</p>`).ReplaceAllString(out, "$1")
	return out, firstErr
}

func roadmapIcon(name string) string {
	return `<svg viewBox="0 0 24 24" aria-hidden="true">` + roadmapIcons[name] + `</svg>`
}

// roadmapText - текст карточки: слова с / или _ и KindPrefix - код.
var roadmapCodeRe = regexp.MustCompile(`(/[a-z0-9/]+|\b[a-z]+_[a-z_]+\b|\bKindPrefix\b|\bgovulncheck\b|\btimeDate\b|\btimeCycle\b|\bcamunda:\*)`)

func roadmapText(s string) string {
	return roadmapCodeRe.ReplaceAllString(html.EscapeString(s), "<code>$1</code>")
}

func meter(level int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<span class="rm-meter" role="img" aria-label="Сложность %d из %d">`, level, roadmapLevels)
	for i := 1; i <= roadmapLevels; i++ {
		if i <= level {
			b.WriteString(`<i class="on"></i>`)
		} else {
			b.WriteString(`<i></i>`)
		}
	}
	b.WriteString(`</span>`)
	return b.String()
}

func (r *roadmapData) overview(anchors map[string]string) (string, error) {
	var b strings.Builder
	b.WriteString(`<div class="rm-overview"><ol class="rm-stages">`)
	for i, s := range r.Stages {
		id, ok := anchors[s.Heading]
		if !ok {
			return "", fmt.Errorf("этап %s: нет заголовка %q", s.Id, s.Heading)
		}
		fmt.Fprintf(&b, `<li><a class="rm-stage" href="#%s"><span class="rm-stage-icon">%s</span>`+
			`<span class="rm-stage-num">Этап %d</span><span class="rm-stage-name">%s</span><span class="rm-dots">`,
			html.EscapeString(id), roadmapIcon(s.Icon), i+1, html.EscapeString(s.Short))
		for _, it := range s.Items {
			fmt.Fprintf(&b, `<i class="rm-l%d" data-tip="%s"></i>`, it.Level, html.EscapeString(it.Title))
		}
		b.WriteString(`</span></a></li>`)
	}
	b.WriteString(`</ol><div class="rm-scale" aria-hidden="true"><span>Сложность</span><span class="rm-scale-bar">`)
	for i := 1; i <= roadmapLevels; i++ {
		fmt.Fprintf(&b, `<i class="rm-l%d"></i>`, i)
	}
	b.WriteString(`</span><span class="rm-scale-ends"><span>быстро</span><span>долго</span></span></div></div><!--rm-->`)
	return b.String(), nil
}

func (r *roadmapData) cards(s roadmapStage) string {
	var b strings.Builder
	b.WriteString(`<div class="rm-grid">`)
	for _, it := range s.Items {
		fmt.Fprintf(&b, `<article class="rm-card rm-l%d"><div class="rm-card-head"><span class="rm-icon">%s</span>%s</div>`+
			`<h3 class="rm-title">%s</h3><p class="rm-text">%s</p></article>`,
			it.Level, roadmapIcon(it.Icon), meter(it.Level), html.EscapeString(it.Title), roadmapText(it.Text))
	}
	b.WriteString(`</div><!--rm-->`)
	return b.String()
}
