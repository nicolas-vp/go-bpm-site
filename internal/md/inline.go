package md

import (
	"html"
	"strings"
	"unicode"
)

func isLetterOrDigit(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || unicode.Is(unicode.Mn, c)
}

// inline переводит разметку строки в HTML.
func (r *renderer) inline(s string) string {
	var b strings.Builder
	r.span(&b, s)
	return b.String()
}

func (r *renderer) span(b *strings.Builder, s string) {
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && strings.IndexByte("\\`*_[]()#+-.!|<>", s[i+1]) >= 0:
			b.WriteString(html.EscapeString(s[i+1 : i+2]))
			i += 2
		case c == '`':
			n := run(s[i:], '`')
			end := strings.Index(s[i+n:], strings.Repeat("`", n))
			if end < 0 {
				b.WriteString(s[i : i+n])
				i += n
				continue
			}
			code := s[i+n : i+n+end]
			if len(code) > 1 && code[0] == ' ' && code[len(code)-1] == ' ' {
				code = code[1 : len(code)-1]
			}
			b.WriteString("<code>" + html.EscapeString(code) + "</code>")
			i += 2*n + end
		case c == '!' && i+1 < len(s) && s[i+1] == '[':
			if text, dest, n, ok := link(s[i+1:]); ok {
				b.WriteString(`<img src="` + html.EscapeString(r.opt.Link(dest, true)) + `" alt="` +
					html.EscapeString(plainText(text)) + `" loading="lazy">`)
				i += 1 + n
				continue
			}
			b.WriteByte('!')
			i++
		case c == '[':
			if text, dest, n, ok := link(s[i:]); ok {
				href := r.opt.Link(dest, false)
				b.WriteString(`<a href="` + html.EscapeString(href) + `"`)
				if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
					b.WriteString(` rel="noopener"`)
				}
				b.WriteString(">")
				r.span(b, text)
				b.WriteString("</a>")
				i += n
				continue
			}
			b.WriteByte('[')
			i++
		case c == '<' && (strings.HasPrefix(s[i:], "<http://") || strings.HasPrefix(s[i:], "<https://")):
			if end := strings.IndexByte(s[i:], '>'); end > 0 {
				u := s[i+1 : i+end]
				b.WriteString(`<a href="` + html.EscapeString(u) + `" rel="noopener">` + html.EscapeString(u) + "</a>")
				i += end + 1
				continue
			}
			b.WriteString("&lt;")
			i++
		case c == '*' || c == '_':
			n := run(s[i:], c)
			if n >= 2 && canOpen(s, i, 2) {
				if end := closing(s, i+2, c, 2); end > 0 {
					b.WriteString("<strong>")
					r.span(b, s[i+2:end])
					b.WriteString("</strong>")
					i = end + 2
					continue
				}
			}
			if n == 1 && canOpen(s, i, 1) && (c == '*' || wordBoundary(s, i)) {
				if end := closing(s, i+1, c, 1); end > 0 {
					b.WriteString("<em>")
					r.span(b, s[i+1:end])
					b.WriteString("</em>")
					i = end + 1
					continue
				}
			}
			b.WriteString(s[i : i+n])
			i += n
		case c == '\n':
			b.WriteByte('\n')
			i++
		default:
			j := i + 1
			for j < len(s) && strings.IndexByte("\\`![<*_\n", s[j]) < 0 {
				j++
			}
			b.WriteString(html.EscapeString(s[i:j]))
			i = j
		}
	}
}

func run(s string, c byte) int {
	n := 0
	for n < len(s) && s[n] == c {
		n++
	}
	return n
}

// canOpen - за открывающим маркером не пробел.
func canOpen(s string, i, n int) bool {
	return i+n < len(s) && s[i+n] != ' ' && s[i+n] != '\n'
}

// wordBoundary - _ открывает курсив только не внутри слова (snake_case не трогаем).
func wordBoundary(s string, i int) bool {
	if i == 0 {
		return true
	}
	p := rune(s[i-1])
	return !(p < 128 && (unicode.IsLetter(p) || unicode.IsDigit(p)))
}

// closing ищет закрывающий маркер из n символов c, пропуская код в обратных кавычках.
func closing(s string, from int, c byte, n int) int {
	marker := strings.Repeat(string(c), n)
	for j := from; j+n <= len(s); j++ {
		if s[j] == '`' {
			k := run(s[j:], '`')
			if end := strings.Index(s[j+k:], strings.Repeat("`", k)); end >= 0 {
				j += 2*k + end - 1
				continue
			}
		}
		if strings.HasPrefix(s[j:], marker) && s[j-1] != ' ' && (n == 2 || j+1 >= len(s) || s[j+1] != c) {
			if c == '_' && j+n < len(s) && isLetterOrDigit(rune(s[j+n])) {
				continue
			}
			return j
		}
	}
	return -1
}

// link разбирает [текст](адрес "заголовок") в начале s.
func link(s string) (text, dest string, n int, ok bool) {
	depth := 0
	end := -1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '`':
			k := run(s[i:], '`')
			if e := strings.Index(s[i+k:], strings.Repeat("`", k)); e >= 0 {
				i += 2*k + e - 1
			}
		case '[':
			depth++
		case ']':
			depth--
		}
		if depth == 0 {
			end = i
			break
		}
	}
	if end < 0 || end+1 >= len(s) || s[end+1] != '(' {
		return "", "", 0, false
	}
	close := strings.IndexByte(s[end+2:], ')')
	if close < 0 {
		return "", "", 0, false
	}
	dest = strings.TrimSpace(s[end+2 : end+2+close])
	if sp := strings.Index(dest, ` "`); sp >= 0 {
		dest = dest[:sp]
	}
	dest = strings.Trim(dest, "<>")
	return s[1:end], dest, end + 3 + close, true
}

// plainText - текст без разметки: для заголовков, оглавления, alt.
func plainText(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == '\\' && i+1 < len(s):
			b.WriteByte(s[i+1])
			i += 2
		case c == '`' || c == '*':
			i++
		case c == '!' && i+1 < len(s) && s[i+1] == '[':
			i++
		case c == '[':
			if text, _, n, ok := link(s[i:]); ok {
				b.WriteString(plainText(text))
				i += n
				continue
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return strings.TrimSpace(b.String())
}
