package main

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Страница «Насколько быстрый go-bpm»: результаты bench/results/*.json рядом с таблицами статьи
// content/performance-article.json. В content/performance.md - подстановки:
//
//	@@chart:false@@ / @@chart:true@@ - панель столбцов без истории / с историей
//	@@table@@                        - таблица по отметкам
//	@@v:<файл результата без .json>@@ - экземпляров в секунду прогона go-bpm
//	@@stand@@                        - процессор, ядра, PostgreSQL, Go прогонов go-bpm

type article struct {
	Source      string          `json:"source"`
	Stand       string          `json:"stand"`
	Checkpoints []int           `json:"checkpoints"`
	Systems     []articleSystem `json:"systems"`
}

type articleSystem struct {
	Name      string    `json:"name"`
	History   bool      `json:"history"`
	PerSecond []float64 `json:"per_second"`
}

// benchResult - часть результата bench/cmd/bench, нужная странице.
type benchResult struct {
	Process    string  `json:"process"`
	History    bool    `json:"history"`
	N          int     `json:"n"`
	PerSecond  float64 `json:"per_second"`
	TotalMs    int64   `json:"total_ms"`
	SyncCommit string  `json:"synchronous_commit"`
	CPU        string  `json:"cpu"`
	NumCPU     int     `json:"num_cpu"`
	GoVersion  string  `json:"go_version"`
	Postgres   string  `json:"postgres"`
	Date       string  `json:"date"`
	// коммитов в секунду, которые база выдерживала до и после прогона
	CommitsBefore float64 `json:"commits_per_second_before"`
	CommitsAfter  float64 `json:"commits_per_second_after"`
	Checkpoints   []struct {
		Completed int     `json:"completed"`
		PerSecond float64 `json:"per_second"`
	} `json:"checkpoints"`
}

// perfRow - строка графика и таблицы.
type perfRow struct {
	Name    string
	History bool
	GoBpm   bool
	Points  map[int]float64 // отметка -> экземпляров в секунду
	Final   float64
	// прогонов go-bpm и разброс их итогов; Final - медиана
	Runs     int
	Min, Max float64
}

type perfData struct {
	article article
	results map[string]benchResult // медианный прогон варианта
	runs    []benchResult          // все прогоны
	rows    []perfRow
}

func loadPerf(articleFile, resultsDir string) (*perfData, error) {
	data, err := os.ReadFile(articleFile)
	if err != nil {
		return nil, err
	}
	p := &perfData{results: map[string]benchResult{}}
	if err := json.Unmarshal(data, &p.article); err != nil {
		return nil, fmt.Errorf("%s: %w", articleFile, err)
	}
	for _, s := range p.article.Systems {
		if len(s.PerSecond) != len(p.article.Checkpoints) {
			return nil, fmt.Errorf("%s: у %s %d значений на %d отметок", articleFile, s.Name, len(s.PerSecond), len(p.article.Checkpoints))
		}
		row := perfRow{Name: s.Name, History: s.History, Points: map[int]float64{}, Final: s.PerSecond[len(s.PerSecond)-1]}
		for i, cp := range p.article.Checkpoints {
			row.Points[cp] = s.PerSecond[i]
		}
		p.rows = append(p.rows, row)
	}

	// results/<вариант>/<номер прогона>.json: на странице медиана прогонов варианта, в подсказке - разброс
	dirs, err := filepath.Glob(filepath.Join(resultsDir, "*", "*.json"))
	if err != nil {
		return nil, err
	}
	runs := map[string][]benchResult{}
	for _, f := range dirs {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var r benchResult
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		key := filepath.Base(filepath.Dir(f))
		runs[key] = append(runs[key], r)
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("нет результатов в %s: запустите bench/run.sh", resultsDir)
	}
	for key, rs := range runs {
		sort.Slice(rs, func(i, j int) bool { return rs[i].PerSecond < rs[j].PerSecond })
		r := rs[len(rs)/2]
		p.results[key] = r
		p.runs = append(p.runs, rs...)
		row := perfRow{Name: goBpmName(r), History: r.History, GoBpm: true, Points: map[int]float64{}, Final: r.PerSecond,
			Runs: len(rs), Min: rs[0].PerSecond, Max: rs[len(rs)-1].PerSecond}
		for _, cp := range r.Checkpoints {
			row.Points[cp.Completed] = cp.PerSecond
		}
		// итог - с учётом завершающих шагов после последней задачи
		row.Points[r.N] = r.PerSecond
		p.rows = append(p.rows, row)
	}
	sort.SliceStable(p.rows, func(i, j int) bool { return p.rows[i].Final > p.rows[j].Final })
	return p, nil
}

func goBpmName(r benchResult) string {
	kind := "service task"
	if r.Process == "BenchMethods" {
		kind = "методы"
	}
	name := "go-bpm, " + kind
	if r.History {
		name += ", история"
	} else {
		name += ", без истории"
	}
	if r.SyncCommit == "off" {
		name += ", async commit"
	}
	return name
}

var perfRe = regexp.MustCompile(`@@(chart:(?:true|false)|table|stand|v:[a-z0-9-]+)@@`)

// renderPerf заменяет подстановки в HTML страницы, отрендеренной из Markdown.
func (p *perfData) render(page string) (string, error) {
	var firstErr error
	out := perfRe.ReplaceAllStringFunc(page, func(m string) string {
		key := strings.Trim(m, "@")
		switch {
		case key == "chart:true" || key == "chart:false":
			return p.chart(key == "chart:true")
		case key == "table":
			return p.table()
		case key == "stand":
			return html.EscapeString(p.stand())
		default:
			r, ok := p.results[strings.TrimPrefix(key, "v:")]
			if !ok {
				if firstErr == nil {
					firstErr = fmt.Errorf("нет результата %s", key)
				}
				return m
			}
			return formatInt(r.PerSecond)
		}
	})
	// подстановка-блок стоит отдельным абзацем: <p> вокруг панели не нужен
	out = regexp.MustCompile(`<p>(<div class="perf-[^"]+"[\s\S]*?</div><!--perf-->)</p>`).ReplaceAllString(out, "$1")
	return out, firstErr
}

func (p *perfData) stand() string {
	lo, hi := math.Inf(1), 0.0
	var r benchResult
	for _, r = range p.runs {
		// с synchronous_commit=off замер не ждёт диска и про диск ничего не говорит
		if r.SyncCommit == "off" {
			continue
		}
		for _, c := range []float64{r.CommitsBefore, r.CommitsAfter} {
			if c > 0 {
				lo, hi = math.Min(lo, c), math.Max(hi, c)
			}
		}
	}
	s := fmt.Sprintf("%s (%d потоков), PostgreSQL %s, %s, %s", r.CPU, r.NumCPU, r.Postgres, r.GoVersion, r.Date)
	if hi > 0 {
		s += fmt.Sprintf("; до и после прогонов база выдерживала от %s до %s коммитов в секунду с ожиданием диска", formatInt(lo), formatInt(hi))
	}
	return s
}

// scaleMax - общий масштаб обеих панелей, чтобы их можно было сравнивать на глаз.
func (p *perfData) scaleMax() float64 {
	m := 0.0
	for _, r := range p.rows {
		m = math.Max(m, r.Final)
	}
	return math.Ceil(m/1000) * 1000
}

func (p *perfData) chart(history bool) string {
	max := p.scaleMax()
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="perf-chart" style="--perf-steps:%d" role="figure" aria-label="`, int(max/1000))
	if history {
		b.WriteString(`Экземпляров в секунду с историей или экспортом`)
	} else {
		b.WriteString(`Экземпляров в секунду без истории`)
	}
	b.WriteString(`"><div class="perf-axis" aria-hidden="true">`)
	for v := 0.0; v <= max; v += 1000 {
		fmt.Fprintf(&b, `<span style="left:%.2f%%">%s</span>`, v/max*100, compact(v))
	}
	b.WriteString(`</div><ul class="perf-bars">`)
	for _, r := range p.rows {
		if r.History != history {
			continue
		}
		class := "perf-row"
		if r.GoBpm {
			class += " perf-gobpm"
		}
		tip := fmt.Sprintf("%s: %s экз./с на 100 тыс.", r.Name, formatInt(r.Final))
		if r.Runs > 1 {
			tip += fmt.Sprintf(", медиана %d прогонов, от %s до %s", r.Runs, formatInt(r.Min), formatInt(r.Max))
		} else if lo, hi, ok := r.span(); ok {
			tip += fmt.Sprintf("; от %s до %s после прогрева", formatInt(lo), formatInt(hi))
		}
		fmt.Fprintf(&b, `<li class="%s" data-tip="%s"><span class="perf-name">%s</span>`+
			`<span class="perf-track"><span class="perf-bar" style="width:%.2f%%"></span>`+
			`<span class="perf-value">%s</span></span></li>`,
			class, html.EscapeString(tip), html.EscapeString(r.Name), r.Final/max*100, formatInt(r.Final))
	}
	b.WriteString(`</ul></div><!--perf-->`)
	return b.String()
}

// span - разброс скорости на отметках от 10 тыс.: виден ли спад по мере роста базы.
func (r perfRow) span() (lo, hi float64, ok bool) {
	lo = math.Inf(1)
	for cp, v := range r.Points {
		if cp < 10000 {
			continue
		}
		lo, hi, ok = math.Min(lo, v), math.Max(hi, v), true
	}
	return lo, hi, ok
}

func (p *perfData) table() string {
	cps := p.article.Checkpoints
	var b strings.Builder
	b.WriteString(`<div class="perf-table table"><table><thead><tr><th>Система</th>`)
	for _, cp := range cps {
		fmt.Fprintf(&b, `<th style="text-align:right">%s</th>`, compact(float64(cp)))
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, history := range []bool{false, true} {
		title := "Без истории"
		if history {
			title = "С историей или экспортом"
		}
		fmt.Fprintf(&b, `<tr class="perf-group"><td colspan="%d">%s</td></tr>`, len(cps)+1, title)
		for _, r := range p.rows {
			if r.History != history {
				continue
			}
			class := ""
			if r.GoBpm {
				class = ` class="perf-gobpm"`
			}
			fmt.Fprintf(&b, `<tr%s><td>%s</td>`, class, html.EscapeString(r.Name))
			for _, cp := range cps {
				v, ok := r.Points[cp]
				cell := "—"
				if ok {
					cell = formatInt(v)
				}
				fmt.Fprintf(&b, `<td style="text-align:right">%s</td>`, cell)
			}
			b.WriteString(`</tr>`)
		}
	}
	b.WriteString(`</tbody></table></div><!--perf-->`)
	return b.String()
}

// formatInt - целое с узким неразрывным пробелом между разрядами: 4 154.
func formatInt(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(" ")
		}
		b.WriteRune(c)
	}
	return b.String()
}

func compact(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.0f тыс.", v/1000)
	}
	return fmt.Sprintf("%.0f", v)
}
