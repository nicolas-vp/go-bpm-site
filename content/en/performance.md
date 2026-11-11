# Performance

The project repeats the simple process benchmark from the 2021 article [Zeebe and Camunda under high load](https://habr.com/ru/companies/raiffeisenbank/articles/594117/) on the same machine: Intel Core i7-4770K, four cores, 24 GB RAM and SSD storage.

## Results at a glance

- With PostgreSQL history, synchronous methods reach **@@v:methods-history-true@@ instances/s**, and service tasks reach **@@v:service-history-true@@ instances/s**.
- Without history, methods reach **@@v:methods-history-false@@ instances/s**. Every process step still commits to PostgreSQL.
- History adds roughly 10–15% overhead in these runs because its rows share the step transaction.
- With `synchronous_commit=off`, service tasks with history reach **@@v:service-history-true-async-commit@@ instances/s**, with the durability tradeoff explained below.

## Comparison charts

Throughput is completed instances divided by elapsed time from the first start, for 100,000 starts. go-bpm bars are the median of three runs; hover or focus a bar to see the range. Other systems' values come from the 2021 article, rather than new runs.

### Without history

@@chart:false@@

### With history or export

@@chart:true@@

Camel and Spring Integration do not persist BPM instance state and do not resume failed runs after restart. They serve as a throughput reference for this hardware.

### Checkpoints

Average throughput since the beginning at each completion checkpoint. go-bpm uses the median run.

@@table@@

## Methodology

The process is start → two REST calls → end. Each call requests static `{}` with a random `?v=` parameter to avoid caching. The article used nginx; these runs use an HTTP stub inside the Go process.

- **Methods:** both calls run synchronously in one system queue step. A failure in the second call repeats the entire step.
- **Service tasks:** each call has a separate queued handler and retry budget, adding transaction overhead.

The actual benchmark methods and handlers:

```go file=bench/internal/bpmgen/bench_methods/method_rest2.go decl=Rest2
```

```go file=bench/internal/bpmgen/handlers/rest_call.go decl=RestCall
```

Eight goroutines start 100,000 instances through `StartProcess`; 16 system workers and 16 service task workers execute them. Each variant runs three times, clearing the dedicated benchmark database between runs.

```sh
GO_BPM_BENCH_DSN='postgres://postgres:postgres@localhost:5432/go_bpm_bench?sslmode=disable' ./run.sh
```

Environment: @@stand@@. PostgreSQL 18 runs in Docker with host networking, `shared_buffers=2GB` and `max_wal_size=4GB`; remaining settings use defaults. `synchronous_commit=on` except where explicitly marked.

## Interpretation and limitations

The comparison systems were measured in 2021 with different Java, Camunda and Zeebe versions. Treat this as an order-of-magnitude comparison on the same hardware, not a current product ranking.

Disk commit throughput varied substantially between runs. Four CPU cores share starters, workers, the stub and PostgreSQL. Throughput changes after the starters finish enqueueing; charts therefore show medians and ranges.

The article used default PostgreSQL settings; go-bpm used the tuning above. With defaults and Docker port forwarding on this machine, method throughput was about 1,000–1,900 instances/s and service tasks about 350–500.

`synchronous_commit=off` acknowledges commits before WAL is flushed to disk. A database server crash can lose recent acknowledged transactions. Use that configuration only when the application's durability requirements permit it.

## A bug found by the benchmark

Early runs sometimes stalled for about 40 seconds at the end: a queue claim leased several tasks but delivered only one to a worker, leaving others waiting for lease expiry. This was fixed in [d7d82c6](https://github.com/nicolas-vp/go-bpm/commit/d7d82c6). The results above include the fix.
