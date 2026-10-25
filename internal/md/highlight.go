package md

import (
	"html"
	"strings"
)

// Highlight раскрашивает код: комментарии, строки, числа, ключевые слова, теги XML. Классы — tok-*.
// Не настоящий лексер, но для примеров документации (go, sh, json, sql, xml) хватает.
func Highlight(code, lang string) string {
	switch lang {
	case "go":
		return lex(code, goSyntax)
	case "sh", "bash", "shell":
		return lex(code, shSyntax)
	case "json":
		return lex(code, jsonSyntax)
	case "sql":
		return lex(code, sqlSyntax)
	case "yaml", "yml":
		return lex(code, yamlSyntax)
	case "xml", "html", "bpmn", "dmn":
		return xml(code)
	}
	return html.EscapeString(code)
}

type syntax struct {
	line     []string // начала однострочных комментариев
	block    [2]string
	quotes   string
	keywords map[string]bool
	builtins map[string]bool
	fold     bool // ключевые слова без учёта регистра
	keys     bool // "строка": - ключ объекта
	flags    bool // -флаг после пробела
	shell    bool // слова с дефисами и точками, команда подсвечивается только в начале
}

func words(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

var (
	goSyntax = syntax{
		line: []string{"//"}, block: [2]string{"/*", "*/"}, quotes: "\"'`",
		keywords: words(`break case chan const continue default defer else fallthrough for func go goto if import
			interface map package range return select struct switch type var`),
		builtins: words(`true false nil iota append cap close len make new panic recover print println any error
			string bool int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 float32 float64 byte rune`),
	}
	shSyntax = syntax{
		line: []string{"#"}, quotes: "\"'", flags: true, shell: true,
		keywords: words(`if then else fi for do done in while case esac export`),
		builtins: words(`curl go make docker git cp cd createdb psql open ssh echo cat sudo systemctl`),
	}
	jsonSyntax = syntax{quotes: "\"", builtins: words("true false null"), keys: true}
	sqlSyntax  = syntax{
		line: []string{"--"}, block: [2]string{"/*", "*/"}, quotes: "'", fold: true,
		keywords: words(`select from where and or not in is null update set delete insert into values create table
			index on primary key default returning order by limit group having join left as with count distinct
			interval now begin commit alter drop if exists using unique check`),
	}
	yamlSyntax = syntax{line: []string{"#"}, quotes: "\"'", builtins: words("true false null")}
)

func span(class, s string) string {
	return `<span class="tok-` + class + `">` + html.EscapeString(s) + "</span>"
}

func lex(code string, sx syntax) string {
	var b strings.Builder
	command := true // shell: следующее слово - команда
	for i := 0; i < len(code); {
		rest := code[i:]
		if sx.block[0] != "" && strings.HasPrefix(rest, sx.block[0]) {
			end := strings.Index(rest[2:], sx.block[1])
			n := len(rest)
			if end >= 0 {
				n = end + 2 + len(sx.block[1])
			}
			b.WriteString(span("comment", rest[:n]))
			i += n
			continue
		}
		if c := lineComment(rest, sx.line, i == 0 || isSpace(code[i-1])); c > 0 {
			b.WriteString(span("comment", rest[:c]))
			i += c
			continue
		}
		c := code[i]
		switch {
		case strings.IndexByte(sx.quotes, c) >= 0:
			n := quoted(rest, c)
			class := "string"
			if sx.keys && strings.HasPrefix(strings.TrimLeft(rest[n:], " "), ":") {
				class = "key"
			}
			b.WriteString(span(class, rest[:n]))
			i += n
		case c >= '0' && c <= '9' && (i == 0 || !isIdent(code[i-1])):
			n := 1
			for n < len(rest) && (isIdent(rest[n]) || rest[n] == '.') {
				n++
			}
			b.WriteString(span("number", rest[:n]))
			i += n
		case isIdent(c) || c == '-' && sx.flags && (i == 0 || isSpace(code[i-1])):
			n := 1
			for n < len(rest) && (isIdent(rest[n]) || rest[n] == '-' && (c == '-' || sx.shell) ||
				sx.shell && strings.IndexByte("./@:=", rest[n]) >= 0) {
				n++
			}
			w := rest[:n]
			wasCommand := command
			command = false
			key := w
			if sx.fold {
				key = strings.ToLower(w)
			}
			switch {
			case sx.keywords[key]:
				b.WriteString(span("keyword", w))
			case sx.builtins[key] && (!sx.shell || wasCommand):
				b.WriteString(span("builtin", w))
			case c == '-' && n > 1:
				b.WriteString(span("flag", w))
			case n < len(rest) && rest[n] == '(' && sx.keywords != nil:
				b.WriteString(span("func", w))
			default:
				b.WriteString(html.EscapeString(w))
			}
			i += n
		default:
			if sx.shell && (c == '\n' && (i == 0 || code[i-1] != '\\') || c == '|' || c == ';' || c == '&' || c == '(') {
				command = true
			}
			b.WriteString(html.EscapeString(string(c)))
			i++
		}
	}
	return b.String()
}

func lineComment(rest string, starts []string, afterSpace bool) int {
	for _, s := range starts {
		if strings.HasPrefix(rest, s) && (s != "#" || afterSpace) {
			if n := strings.IndexByte(rest, '\n'); n >= 0 {
				return n
			}
			return len(rest)
		}
	}
	return 0
}

func quoted(s string, q byte) int {
	for n := 1; n < len(s); n++ {
		switch {
		case s[n] == '\\' && q != '`':
			n++
		case s[n] == q:
			return n + 1
		case s[n] == '\n' && q != '`':
			return n
		}
	}
	return len(s)
}

func isIdent(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' }

func xml(code string) string {
	var b strings.Builder
	for i := 0; i < len(code); {
		rest := code[i:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			n := strings.Index(rest, "-->")
			if n < 0 {
				n = len(rest)
			} else {
				n += 3
			}
			b.WriteString(span("comment", rest[:n]))
			i += n
		case rest[0] == '<':
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				end = len(rest) - 1
			}
			b.WriteString(tag(rest[:end+1]))
			i += end + 1
		default:
			n := strings.IndexByte(rest, '<')
			if n < 0 {
				n = len(rest)
			}
			b.WriteString(html.EscapeString(rest[:n]))
			i += n
		}
	}
	return b.String()
}

// tag раскрашивает один тег: имя, атрибуты, значения.
func tag(t string) string {
	var b strings.Builder
	n := 1
	for n < len(t) && (t[n] == '/' || t[n] == '?') {
		n++
	}
	b.WriteString(html.EscapeString(t[:n]))
	m := n
	for m < len(t) && !isSpace(t[m]) && t[m] != '>' && t[m] != '/' {
		m++
	}
	b.WriteString(span("tag", t[n:m]))
	for i := m; i < len(t); {
		c := t[i]
		switch {
		case c == '"' || c == '\'':
			q := quoted(t[i:], c)
			b.WriteString(span("string", t[i:i+q]))
			i += q
		case isIdent(c) || c == ':':
			j := i
			for j < len(t) && (isIdent(t[j]) || t[j] == ':' || t[j] == '-') {
				j++
			}
			b.WriteString(span("attr", t[i:j]))
			i = j
		default:
			b.WriteString(html.EscapeString(string(c)))
			i++
		}
	}
	return b.String()
}
