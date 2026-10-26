// Команда bench - нагрузочный тест go-bpm по методике статьи «Zeebe и Camunda: сравниваем известные BPM-системы
// под высокими нагрузками» (Хабр, 2021): запустить N экземпляров процесса из двух REST-вызовов и засечь, за сколько
// завершились первые 1 тыс., 10 тыс., 20 тыс. ... экземпляров.
//
//	go run ./cmd/bench -dsn postgres://.../go_bpm_bench -process service -history=true -n 100000
//
// Таблицы bpm_* базы очищаются перед прогоном: нужна отдельная база.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolas-vp/go-bpm/engine"
	"github.com/nicolas-vp/go-bpm/migrations"
	"github.com/nicolas-vp/go-bpm/queue"

	"example.com/bench/internal/bpmgen"
	"example.com/bench/internal/bpmgen/services"
)

// processes - варианты процесса: две задачи-метода в одном шаге или две service task через очередь
var processes = map[string]string{
	"methods": "BenchMethods",
	"service": "BenchServiceTasks",
}

type config struct {
	dsn, process, url, out     string
	n, step, starters          int
	systemWorkers, taskWorkers int
	poolSize                   int
	history, truncate, verbose bool
}

// Checkpoint - сколько экземпляров завершилось к моменту Elapsed от первого старта.
type Checkpoint struct {
	Completed int     `json:"completed"`
	ElapsedMs int64   `json:"elapsed_ms"`
	PerSecond float64 `json:"per_second"`
}

// Result - итог прогона, пишется в JSON для страницы сайта.
type Result struct {
	Process       string       `json:"process"`
	History       bool         `json:"history"`
	N             int          `json:"n"`
	Starters      int          `json:"starters"`
	SystemWorkers int          `json:"system_workers"`
	TaskWorkers   int          `json:"task_workers"`
	PoolSize      int          `json:"pool_size"`
	StartedInMs   int64        `json:"started_in_ms"`
	Checkpoints   []Checkpoint `json:"checkpoints"`
	TotalMs       int64        `json:"total_ms"`
	PerSecond     float64      `json:"per_second"`
	RestCalls     int64        `json:"rest_calls"`
	DBSizeMB      float64      `json:"db_size_mb"`
	CPU           string       `json:"cpu"`
	NumCPU        int          `json:"num_cpu"`
	GoVersion     string       `json:"go_version"`
	Postgres      string       `json:"postgres"`
	SyncCommit    string       `json:"synchronous_commit"`
	// CommitsBefore, CommitsAfter - сколько коммитов в секунду база выдерживала до и после прогона:
	// потолок для go-bpm, по нему видно, не мешало ли что-то стенду
	CommitsBefore float64 `json:"commits_per_second_before"`
	CommitsAfter  float64 `json:"commits_per_second_after"`
	Date          string  `json:"date"`
}

func main() {
	var c config
	flag.StringVar(&c.dsn, "dsn", os.Getenv("GO_BPM_BENCH_DSN"), "PostgreSQL DSN отдельной базы, или GO_BPM_BENCH_DSN")
	flag.StringVar(&c.process, "process", "service", "вариант процесса: service (две service task) или methods (два метода)")
	flag.StringVar(&c.url, "url", "", "адрес REST-заглушки; пусто - поднять свою, отдаёт {}")
	flag.StringVar(&c.out, "out", "", "куда записать результат в JSON")
	flag.IntVar(&c.n, "n", 100000, "сколько экземпляров запустить")
	flag.IntVar(&c.step, "step", 10000, "шаг замеров")
	flag.IntVar(&c.starters, "starters", 8, "сколько горутин запускают экземпляры")
	flag.IntVar(&c.systemWorkers, "system-workers", 16, "engine.Config.SystemWorkers")
	flag.IntVar(&c.taskWorkers, "task-workers", 16, "engine.Config.TaskWorkers")
	flag.IntVar(&c.poolSize, "pool", 48, "размер пула соединений")
	flag.BoolVar(&c.history, "history", true, "история и экземпляры в базе (History.ToDB)")
	flag.BoolVar(&c.verbose, "v", false, "писать предупреждения и ошибки движка и очереди в stderr")
	flag.BoolVar(&c.truncate, "truncate", false, "подтверждение: очистить таблицы bpm_* базы перед прогоном")
	flag.Parse()
	if err := run(c); err != nil {
		log.Fatal(err)
	}
}

func run(c config) error {
	name, ok := processes[c.process]
	if !ok {
		return fmt.Errorf("неизвестный вариант процесса %q", c.process)
	}
	if c.dsn == "" || !c.truncate {
		return errors.New("нужны -dsn отдельной базы и -truncate: таблицы bpm_* будут очищены")
	}
	ctx := context.Background()

	url := c.url
	if url == "" {
		var err error
		if url, err = restStub(); err != nil {
			return err
		}
	}

	poolConfig, err := pgxpool.ParseConfig(c.dsn)
	if err != nil {
		return err
	}
	poolConfig.MaxConns = int32(c.poolSize)
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.Up(ctx, pool); err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `truncate bpm_queue, bpm_parallel_sync, bpm_wait, bpm_history, bpm_rules_history,
		bpm_process_instance, bpm_user_task restart identity`)
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `vacuum analyze`); err != nil {
		return err
	}

	var finished, calls atomic.Int64
	svc := &services.Services{
		Rest:     &restClient{url: url, calls: &calls, http: &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 256}}},
		Finished: func() { finished.Add(1) },
	}
	log := slog.New(slog.DiscardHandler)
	if c.verbose {
		log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	history := func() bool { return c.history }
	q := queue.New(pool, queue.Options{Logger: log})
	e, err := engine.New(q, engine.Config{
		SystemWorkers: c.systemWorkers,
		TaskWorkers:   c.taskWorkers,
		History:       engine.HistoryConfig{ToDB: history},
		Logger:        log,
	}, bpmgen.Processes(svc), bpmgen.Queues(svc))
	if err != nil {
		return err
	}
	q.Start(ctx)
	defer q.Stop()

	res := Result{
		Process: name, History: c.history, N: c.n, Starters: c.starters,
		SystemWorkers: c.systemWorkers, TaskWorkers: c.taskWorkers, PoolSize: c.poolSize,
		CPU: cpuModel(), NumCPU: runtime.NumCPU(), GoVersion: runtime.Version(),
		Date: time.Now().Format(time.DateOnly),
	}
	_ = pool.QueryRow(ctx, `show server_version`).Scan(&res.Postgres)
	_ = pool.QueryRow(ctx, `show synchronous_commit`).Scan(&res.SyncCommit)
	if res.CommitsBefore, err = commitRate(ctx, pool, c.taskWorkers); err != nil {
		return err
	}
	fmt.Printf("%s, история %v, synchronous_commit %s, коммитов/с %.0f: %d экземпляров, стартеров %d, воркеров %d/%d, пул %d\n",
		name, c.history, res.SyncCommit, res.CommitsBefore, c.n, c.starters, c.systemWorkers, c.taskWorkers, c.poolSize)

	// стартеры берут номера из общего счётчика, пока не запустят N экземпляров
	begin := time.Now()
	var launched atomic.Int64
	var startErr error
	var once sync.Once
	var wg sync.WaitGroup
	for range c.starters {
		wg.Go(func() {
			for launched.Add(1) <= int64(c.n) {
				if _, err := e.StartProcess(ctx, name, nil); err != nil {
					once.Do(func() { startErr = err })
					return
				}
			}
		})
	}
	startDone := make(chan struct{})
	go func() {
		wg.Wait()
		res.StartedInMs = time.Since(begin).Milliseconds()
		close(startDone)
	}()

	// замеры: 1 тыс., затем каждые step экземпляров, как в статье
	checkpoints := []int{min(1000, c.n)}
	for k := c.step; k <= c.n; k += c.step {
		if k > checkpoints[len(checkpoints)-1] {
			checkpoints = append(checkpoints, k)
		}
	}
	next := 0
	for next < len(checkpoints) {
		if startErr != nil {
			return fmt.Errorf("старт экземпляра: %w", startErr)
		}
		done := finished.Load()
		for next < len(checkpoints) && done >= int64(checkpoints[next]) {
			elapsed := time.Since(begin)
			cp := Checkpoint{Completed: checkpoints[next], ElapsedMs: elapsed.Milliseconds(),
				PerSecond: float64(checkpoints[next]) / elapsed.Seconds()}
			res.Checkpoints = append(res.Checkpoints, cp)
			fmt.Printf("%7d завершено за %8d мс: %6.0f экз./с\n", cp.Completed, cp.ElapsedMs, cp.PerSecond)
			next++
		}
		time.Sleep(2 * time.Millisecond)
	}
	<-startDone

	// последняя задача отработала; ждём, пока очередь доделает завершающие шаги
	for {
		var pending, failed int
		err := pool.QueryRow(ctx, `select count(*) filter (where status = 'pending'),
			count(*) filter (where status = 'failed') from bpm_queue`).Scan(&pending, &failed)
		if err != nil {
			return err
		}
		if failed > 0 {
			return fmt.Errorf("%d задач упало", failed)
		}
		if pending == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	total := time.Since(begin)
	res.TotalMs = total.Milliseconds()
	res.PerSecond = float64(c.n) / total.Seconds()
	res.RestCalls = calls.Load()
	var size int64
	_ = pool.QueryRow(ctx, `select pg_database_size(current_database())`).Scan(&size)
	res.DBSizeMB = float64(size) / (1 << 20)
	if res.CommitsAfter, err = commitRate(ctx, pool, c.taskWorkers); err != nil {
		return err
	}
	fmt.Printf("итого: %d экземпляров за %d мс, %.0f экз./с; запуск занял %d мс; REST-вызовов %d; база %.0f МБ; коммитов/с после %.0f\n",
		c.n, res.TotalMs, res.PerSecond, res.StartedInMs, res.RestCalls, res.DBSizeMB, res.CommitsAfter)

	if c.out != "" {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(c.out, append(data, '\n'), 0o644)
	}
	return nil
}

// commitRate - сколько однострочных insert с коммитом в секунду база выдерживает с clients соединений за 5 с.
func commitRate(ctx context.Context, pool *pgxpool.Pool, clients int) (float64, error) {
	if _, err := pool.Exec(ctx, `create table if not exists bench_commit_probe(v int)`); err != nil {
		return 0, err
	}
	defer pool.Exec(ctx, `drop table bench_commit_probe`)
	const d = 5 * time.Second
	end := time.Now().Add(d)
	var n atomic.Int64
	var probeErr error
	var once sync.Once
	var wg sync.WaitGroup
	for range clients {
		wg.Go(func() {
			for time.Now().Before(end) {
				if _, err := pool.Exec(ctx, `insert into bench_commit_probe values (1)`); err != nil {
					once.Do(func() { probeErr = err })
					return
				}
				n.Add(1)
			}
		})
	}
	wg.Wait()
	return float64(n.Load()) / d.Seconds(), probeErr
}

// restClient - GET к заглушке со случайным параметром, как в статье (против кэширования).
type restClient struct {
	url   string
	http  *http.Client
	calls *atomic.Int64
}

func (r *restClient) Call(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?v=%d", r.url, rand.Int64()), nil)
	if err != nil {
		return err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("REST: %s", resp.Status)
	}
	r.calls.Add(1)
	return nil
}

// restStub поднимает заглушку на свободном порту localhost: отдаёт {} как статический test.json из статьи.
func restStub() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	return "http://" + ln.Addr().String() + "/test.json", nil
}

func cpuModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	for line := range strings.Lines(string(data)) {
		if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "model name" {
			return strings.TrimSpace(v)
		}
	}
	return runtime.GOARCH
}
