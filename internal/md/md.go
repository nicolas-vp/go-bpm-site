// Package md - Markdown в HTML для документации go-bpm: то подмножество GFM, которым она написана.
//
// Блоки: заголовки ATX, абзацы, списки (маркированные и нумерованные, с вложенными блоками по отступу),
// цитаты, блоки кода ``` с языком, таблицы, горизонтальная черта. Внутри строк: `код`, **жирный**, *курсив*,
// ссылки и картинки, автоссылки <https://...>. Сырой HTML экранируется, кроме <details> и <summary> на своих строках.
package md

import (
	"html"
	"strconv"
	"strings"
)

// Heading - заголовок страницы для оглавления.
type Heading struct {
	Level int
	ID    string
	Text  string // без разметки
}

// Doc - результат разбора страницы.
type Doc struct {
	Title    string // первый заголовок # без разметки
	HTML     string
	Headings []Heading // заголовки ## и ###
}

// Options - как переписывать ссылки и картинки.
type Options struct {
	// Link получает адрес ссылки или картинки из исходника и возвращает адрес на сайте.
	Link func(dest string, image bool) string
}

// Render разбирает Markdown и строит HTML.
func Render(src string, opt Options) Doc {
	if opt.Link == nil {
		opt.Link = func(dest string, _ bool) string { return dest }
	}
	r := &renderer{opt: opt, slugs: map[string]int{}}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var b strings.Builder
	r.blocks(&b, lines)
	return Doc{Title: r.title, HTML: b.String(), Headings: r.headings}
}

type renderer struct {
	opt      Options
	slugs    map[string]int
	title    string
	headings []Heading
}

func (r *renderer) blocks(b *strings.Builder, lines []string) {
	for i := 0; i < len(lines); {
		line := lines[i]
		t := strings.TrimSpace(line)
		switch {
		case t == "":
			i++
		case strings.HasPrefix(t, "```"):
			i = r.fence(b, lines, i)
		case headingLevel(t) > 0:
			r.heading(b, t)
			i++
		case isDetails(t):
			r.details(b, t)
			i++
		case t == "---" || t == "***" || t == "___":
			b.WriteString("<hr>\n")
			i++
		case strings.HasPrefix(t, ">"):
			i = r.quote(b, lines, i)
		case isTableStart(lines, i):
			i = r.table(b, lines, i)
		case listMarker(line) != nil:
			i = r.list(b, lines, i)
		default:
			i = r.paragraph(b, lines, i)
		}
	}
}

func headingLevel(t string) int {
	n := 0
	for n < len(t) && t[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || n == len(t) || t[n] != ' ' {
		return 0
	}
	return n
}

func (r *renderer) heading(b *strings.Builder, t string) {
	level := headingLevel(t)
	raw := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(t[level:]), "#"))
	text := plainText(raw)
	id := r.slug(text)
	switch {
	case level == 1 && r.title == "":
		r.title = text
	case level == 2 || level == 3:
		r.headings = append(r.headings, Heading{Level: level, ID: id, Text: text})
	}
	tag := "h" + string(rune('0'+level))
	b.WriteString("<" + tag + ` id="` + html.EscapeString(id) + `">`)
	b.WriteString(r.inline(raw))
	if level > 1 {
		b.WriteString(`<a class="anchor" href="#` + html.EscapeString(id) + `" aria-label="Ссылка на раздел">#</a>`)
	}
	b.WriteString("</" + tag + ">\n")
}

// slug - якорь как у GitHub: нижний регистр, пробелы в дефисы, без знаков препинания; повторы с -1, -2.
func (r *renderer) slug(text string) string {
	var s strings.Builder
	for _, c := range strings.ToLower(text) {
		switch {
		case c == ' ':
			s.WriteRune('-')
		case c == '-' || c == '_' || isLetterOrDigit(c):
			s.WriteRune(c)
		}
	}
	id := s.String()
	if n := r.slugs[id]; n > 0 {
		r.slugs[id] = n + 1
		id += "-" + strconv.Itoa(n)
	} else {
		r.slugs[id] = 1
	}
	return id
}

func (r *renderer) fence(b *strings.Builder, lines []string, i int) int {
	indent := leadingSpaces(lines[i])
	t := strings.TrimSpace(lines[i])
	lang := strings.TrimSpace(strings.TrimLeft(t, "`"))
	var code []string
	i++
	for ; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			i++
			break
		}
		code = append(code, trimIndent(lines[i], indent))
	}
	b.WriteString(`<div class="code"`)
	if lang != "" {
		b.WriteString(` data-lang="` + html.EscapeString(lang) + `"`)
	}
	b.WriteString(`><pre><code>`)
	b.WriteString(Highlight(strings.Join(code, "\n"), lang))
	b.WriteString("</code></pre></div>\n")
	return i
}

func (r *renderer) quote(b *strings.Builder, lines []string, i int) int {
	var inner []string
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(t, ">") {
			break
		}
		t = strings.TrimPrefix(t, ">")
		inner = append(inner, strings.TrimPrefix(t, " "))
	}
	b.WriteString("<blockquote>\n")
	r.blocks(b, inner)
	b.WriteString("</blockquote>\n")
	return i
}

func (r *renderer) paragraph(b *strings.Builder, lines []string, i int) int {
	var text []string
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" || strings.HasPrefix(t, "```") || headingLevel(t) > 0 || strings.HasPrefix(t, ">") || isDetails(t) ||
			(len(text) > 0 && (listMarker(lines[i]) != nil || isTableStart(lines, i))) {
			break
		}
		text = append(text, t)
	}
	if len(text) == 1 && isImageOnly(text[0]) {
		b.WriteString(`<figure>` + r.inline(text[0]) + "</figure>\n")
		return i
	}
	b.WriteString("<p>" + r.inline(strings.Join(text, "\n")) + "</p>\n")
	return i
}

// isDetails - <details> и <summary> на отдельных строках: единственный сырой HTML, который пропускается.
func isDetails(t string) bool {
	return t == "<details>" || t == "</details>" ||
		strings.HasPrefix(t, "<summary>") && strings.HasSuffix(t, "</summary>")
}

func (r *renderer) details(b *strings.Builder, t string) {
	if strings.HasPrefix(t, "<summary>") {
		inner := strings.TrimSuffix(strings.TrimPrefix(t, "<summary>"), "</summary>")
		b.WriteString("<summary>" + r.inline(inner) + "</summary>\n")
		return
	}
	b.WriteString(t + "\n")
}

func isImageOnly(t string) bool {
	return strings.HasPrefix(t, "![") && strings.HasSuffix(t, ")") && strings.Count(t, "![") == 1
}

type marker struct {
	ordered bool
	width   int // отступ содержимого пункта от начала строки
	start   string
}

func listMarker(line string) *marker {
	sp := leadingSpaces(line)
	t := line[sp:]
	if len(t) >= 2 && (t[0] == '-' || t[0] == '*' || t[0] == '+') && t[1] == ' ' {
		return &marker{width: sp + 2}
	}
	n := 0
	for n < len(t) && n < 9 && t[n] >= '0' && t[n] <= '9' {
		n++
	}
	if n > 0 && n+1 < len(t) && (t[n] == '.' || t[n] == ')') && t[n+1] == ' ' {
		return &marker{ordered: true, width: sp + n + 2, start: t[:n]}
	}
	return nil
}

func (r *renderer) list(b *strings.Builder, lines []string, i int) int {
	first := listMarker(lines[i])
	base := leadingSpaces(lines[i])
	type item struct{ lines []string }
	var items []item
	loose := false
	for i < len(lines) {
		m := listMarker(lines[i])
		if m == nil || m.ordered != first.ordered || leadingSpaces(lines[i]) != base {
			break
		}
		it := item{lines: []string{lines[i][m.width:]}}
		i++
		for i < len(lines) {
			line := lines[i]
			if strings.TrimSpace(line) == "" {
				// пустая строка: пункт продолжается, если дальше строка с отступом содержимого
				j := i
				for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
					j++
				}
				if j < len(lines) && leadingSpaces(lines[j]) >= m.width {
					for ; i < j; i++ {
						it.lines = append(it.lines, "")
					}
					loose = true
					continue
				}
				if j < len(lines) && listMarker(lines[j]) != nil && leadingSpaces(lines[j]) == base {
					loose = true
				}
				i = j
				break
			}
			if leadingSpaces(line) >= m.width {
				it.lines = append(it.lines, line[m.width:])
				i++
				continue
			}
			if listMarker(line) != nil || leadingSpaces(line) < base {
				break
			}
			// ленивое продолжение абзаца пункта
			it.lines = append(it.lines, strings.TrimSpace(line))
			i++
		}
		items = append(items, it)
		if i > 0 && strings.TrimSpace(lines[i-1]) == "" && (i >= len(lines) || listMarker(lines[i]) == nil) {
			break
		}
	}
	tag := "ul"
	if first.ordered {
		tag = "ol"
	}
	b.WriteString("<" + tag)
	if first.ordered && first.start != "1" {
		b.WriteString(` start="` + first.start + `"`)
	}
	b.WriteString(">\n")
	for _, it := range items {
		b.WriteString("<li>")
		var inner strings.Builder
		r.blocks(&inner, it.lines)
		s := inner.String()
		if !loose {
			s = unwrapFirstParagraph(s)
		}
		b.WriteString(strings.TrimSuffix(s, "\n"))
		b.WriteString("</li>\n")
	}
	b.WriteString("</" + tag + ">\n")
	return i
}

// unwrapFirstParagraph - в плотном списке первый абзац пункта пишется без <p>.
func unwrapFirstParagraph(s string) string {
	if !strings.HasPrefix(s, "<p>") {
		return s
	}
	end := strings.Index(s, "</p>\n")
	if end < 0 {
		return s
	}
	return s[3:end] + "\n" + s[end+5:]
}

func isTableStart(lines []string, i int) bool {
	if i+1 >= len(lines) {
		return false
	}
	head, sep := strings.TrimSpace(lines[i]), strings.TrimSpace(lines[i+1])
	if !strings.HasPrefix(head, "|") || !strings.HasPrefix(sep, "|") {
		return false
	}
	for _, c := range sep {
		if !strings.ContainsRune("|-: ", c) {
			return false
		}
	}
	return strings.Contains(sep, "-")
}

func splitRow(t string) []string {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	var cells []string
	var cur strings.Builder
	inCode := false
	for i := 0; i < len(t); i++ {
		c := t[i]
		switch {
		case c == '`':
			inCode = !inCode
			cur.WriteByte(c)
		case c == '\\' && i+1 < len(t) && t[i+1] == '|':
			cur.WriteByte('|')
			i++
		case c == '|' && !inCode:
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

func (r *renderer) table(b *strings.Builder, lines []string, i int) int {
	head := splitRow(lines[i])
	var align []string
	for _, s := range splitRow(lines[i+1]) {
		switch {
		case strings.HasPrefix(s, ":") && strings.HasSuffix(s, ":"):
			align = append(align, "center")
		case strings.HasSuffix(s, ":"):
			align = append(align, "right")
		default:
			align = append(align, "")
		}
	}
	cell := func(tag, s string, n int) {
		b.WriteString("<" + tag)
		if n < len(align) && align[n] != "" {
			b.WriteString(` style="text-align:` + align[n] + `"`)
		}
		b.WriteString(">" + r.inline(s) + "</" + tag + ">")
	}
	b.WriteString(`<div class="table"><table>`)
	empty := true
	for _, h := range head {
		if h != "" {
			empty = false
		}
	}
	if !empty {
		b.WriteString("<thead><tr>")
		for n, h := range head {
			cell("th", h, n)
		}
		b.WriteString("</tr></thead>")
	}
	b.WriteString("<tbody>\n")
	for i += 2; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
		b.WriteString("<tr>")
		row := splitRow(lines[i])
		for n := range head {
			s := ""
			if n < len(row) {
				s = row[n]
			}
			cell("td", s, n)
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody></table></div>\n")
	return i
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

func trimIndent(s string, n int) string {
	if leadingSpaces(s) >= n {
		return s[n:]
	}
	return strings.TrimLeft(s, " ")
}
