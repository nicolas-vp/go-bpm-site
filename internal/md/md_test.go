package md

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string // подстроки результата
		notWant   []string
	}{
		{
			name: "якоря как у GitHub",
			src:  "# Заголовок\n\n## Task — ваш метод\n\n### `POST /api/v1/tasks/{id}/claim`\n\n## Task — ваш метод",
			want: []string{
				`<h2 id="task--ваш-метод">`,
				`<h3 id="post-apiv1tasksidclaim">`,
				`<h2 id="task--ваш-метод-1">`,
			},
		},
		{
			name: "плотный список без абзацев",
			src:  "- один `код`\n- два\n  продолжение",
			want: []string{"<ul>\n<li>один <code>код</code></li>\n<li>два\nпродолжение</li>\n</ul>"},
		},
		{
			name: "код внутри пункта списка",
			src:  "- PostgreSQL:\n\n  ```sh\n  docker run postgres\n  ```\n\n- Modeler",
			want: []string{`<li><p>PostgreSQL:</p>`, `data-lang="sh"`, "docker</span> run postgres", "<li><p>Modeler</p></li>"},
		},
		{
			name: "нумерованный список",
			src:  "3. три\n4. четыре",
			want: []string{`<ol start="3">`, "<li>четыре</li>"},
		},
		{
			name: "таблица с кодом и экранированной чертой",
			src:  "| Поле | Что |\n|---|:-:|\n| `a|b` | x \\| y |",
			want: []string{"<th>Поле</th>", `<th style="text-align:center">Что</th>`, "<code>a|b</code>", "x | y"},
		},
		{
			name: "цитата",
			src:  "> **Авторизации нет.** API\n> и консоль",
			want: []string{"<blockquote>\n<p><strong>Авторизации нет.</strong> API\nи консоль</p>\n</blockquote>"},
		},
		{
			name: "картинка отдельным абзацем",
			src:  "![Схема «A»](images/a.svg)",
			want: []string{`<figure><img src="images/a.svg" alt="Схема «A»" loading="lazy"></figure>`},
		},
		{
			name: "сырой HTML экранируется, details - нет",
			src:  "<details>\n<summary>XML `x`</summary>\n\nтекст <b>\n</details>",
			want: []string{"<details>\n<summary>XML <code>x</code></summary>", "текст &lt;b&gt;", "</details>"},
		},
		{
			name: "звёздочки в коде и snake_case не курсив",
			src:  "`*services.Services` и method_check_order и *курсив*",
			want: []string{"<code>*services.Services</code>", "method_check_order", "<em>курсив</em>"},
		},
		{
			name:    "ссылка с кодом в тексте",
			src:     "[`TestDecision`](#testdecision) и <https://example.com>",
			want:    []string{`<a href="#testdecision"><code>TestDecision</code></a>`, `<a href="https://example.com" rel="noopener">`},
			notWant: []string{"&lt;https"},
		},
		{
			name: "горизонтальная черта",
			src:  "текст\n\n---\n\n## Дальше",
			want: []string{"<hr>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.src, Options{}).HTML
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("нет %q в\n%s", w, got)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(got, w) {
					t.Errorf("лишнее %q в\n%s", w, got)
				}
			}
		})
	}
}

func TestRenderTitleAndHeadings(t *testing.T) {
	doc := Render("# Быстрый `старт`\n\n## Один\n\n### Два\n\n#### Три", Options{})
	if doc.Title != "Быстрый старт" {
		t.Errorf("Title = %q", doc.Title)
	}
	if len(doc.Headings) != 2 || doc.Headings[0].ID != "один" || doc.Headings[1].Level != 3 {
		t.Errorf("Headings = %+v", doc.Headings)
	}
}

func TestRenderLinks(t *testing.T) {
	var seen []string
	Render("[a](quickstart.md#x) ![b](images/b.png)", Options{Link: func(dest string, image bool) string {
		seen = append(seen, dest)
		return "/" + dest
	}})
	if strings.Join(seen, " ") != "quickstart.md#x images/b.png" {
		t.Errorf("Link вызван с %v", seen)
	}
}

func TestHighlight(t *testing.T) {
	tests := []struct{ lang, code, want string }{
		{"go", `func F() error { return nil } // ok`, `<span class="tok-keyword">func</span> <span class="tok-func">F</span>`},
		{"go", `s := "a // b"`, `<span class="tok-string">&#34;a // b&#34;</span>`},
		{"sh", "git clone git@github.com:x/go-bpm.git", `<span class="tok-builtin">git</span> clone git@github.com:x/go-bpm.git`},
		{"sh", "curl -XPOST x | psql", `<span class="tok-flag">-XPOST</span> x | <span class="tok-builtin">psql</span>`},
		{"json", `{"a": "b"}`, `<span class="tok-key">&#34;a&#34;</span>: <span class="tok-string">&#34;b&#34;</span>`},
		{"sql", "SELECT 1 -- c", `<span class="tok-keyword">SELECT</span> <span class="tok-number">1</span> <span class="tok-comment">-- c</span>`},
		{"xml", `<bpmn:task id="A">`, `&lt;<span class="tok-tag">bpmn:task</span> <span class="tok-attr">id</span>=<span class="tok-string">&#34;A&#34;</span>&gt;`},
		{"", "<x>", "&lt;x&gt;"},
	}
	for _, tt := range tests {
		if got := Highlight(tt.code, tt.lang); !strings.Contains(got, tt.want) {
			t.Errorf("Highlight(%q, %q) = %s\nнет %s", tt.code, tt.lang, got, tt.want)
		}
	}
}
