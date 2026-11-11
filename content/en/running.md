# Running and operating

Run go-bpm inside your Go service with PostgreSQL. Apply migrations, register generated processes and queues, enable the history you need, and start queue workers after initialization.

## Database and migrations

Use PostgreSQL 13+ and `pgx/v5`. `migrations.Up(ctx, pool)` applies missing schema migrations and supports concurrent service starts. For an existing migration system, use SQL from `migrations.List()`.

The engine uses tables including `bpm_queue`, `bpm_process_instance`, `bpm_history`, `bpm_rules_history`, `bpm_user_task` and message storage. Keep database backups and access management within the service's normal operations.

## Engine configuration

| `engine.Config` field | Default | Purpose |
|---|---|---|
| `KindPrefix` | `bpm` | Queue kind prefix |
| `SystemWorkers` | 4 | Concurrent process steps per replica |
| `SystemMaxAttempts` | 10 | Attempts before a system step fails |
| `AttemptTimeout` | 30s | Per-attempt context timeout |
| `RetryDelay` | 1s | Initial retry delay |
| `TaskWorkers` | 1 | Workers per service job type per replica |
| `History` | Disabled | Element and instance history |
| `RulesHistory` | Disabled | Matched DMN rule history |
| `Hooks` | None | Service integration callbacks |
| `Logger` | `slog.Default()` | Engine logger |

| `queue.Options` field | Default | Purpose |
|---|---|---|
| `PollInterval` | 1s with jitter | Poll for delayed work |
| `Retention` | 7 days | Retain completed/cancelled queue tasks; negative disables cleanup |
| `OnAttempt` | nil | Attempt outcome and duration callback |
| `Logger` | `slog.Default()` | Queue logger |

Retry delays double with jitter up to five minutes. Immediate tasks wake workers with `LISTEN/NOTIFY`; polling is primarily for timers and retries.

## History

```go
on := func() bool { return true }
e, err := engine.New(q, engine.Config{
    History:      engine.HistoryConfig{ToDB: on},
    RulesHistory: engine.HistoryConfig{ToDB: on},
}, bpmgen.Processes(svc), bpmgen.Queues(svc))
```

`ToLog` and `ToDB` are functions evaluated during execution, so configuration can switch them dynamically. Instance rows and element history require `History.ToDB`; without it, instance search and the console are empty. User tasks are persisted independently.

Variables are recorded at starts and tasks, not on every gateway and flow. Database history commits with the step; logs can contain repeated lines when a transaction retries.

## Hooks and tracing

`engine.Hooks` supports:

- `OnElement(ctx, HistoryRecord)` for activation, completion and termination events, independent of history flags.
- `WrapCall(ctx, Call, next)` around task methods, scripts, decisions and service handlers. Start a tracing span, invoke `next(ctx)` and return its error.
- `OnInstanceFailed(ctx, InstanceFailure)` when attempts are exhausted.

Hooks run synchronously in workers. Element and failure callbacks can repeat after rollback; make external notifications idempotent or record an outbox in the same database transaction.

## History retention

`engine.Retention` deletes old completed and cancelled root instances with their children, element/rule history and user tasks. It preserves running and failed instances. Children remain while their root is active.

Multiple replicas can clean concurrently using row locks. `DeleteExpired(ctx)` allows cleanup from an external schedule. Cleanup also removes expired buffered messages. Queue task cleanup is controlled separately by `queue.Options.Retention`.

## Scaling and shutdown

Multiple replicas can share one database. Workers claim tasks with `FOR UPDATE SKIP LOCKED`. A task lease lasts `AttemptTimeout + 10s`; another worker can recover it after a crashed replica's lease expires.

Increase system workers, per-job-type task workers and replicas according to load. Size `pgxpool` above the total worker count, with an additional connection for `LISTEN`. Long synchronous methods occupy system workers; network and slow operations usually belong in service tasks.

On shutdown, stop HTTP traffic, stop and drain the queue with `q.Stop()`, then close the database pool. In-flight handlers have their attempt timeout rather than immediate signal cancellation.

## Process versions

A token stores its current element ID. Preserve elements with waiting instances when deploying a changed diagram. To remove one, first prevent new arrivals, let existing instances pass, then remove it.

For incompatible versions, run the new service with a different `KindPrefix`, such as `bpm2`, while the old version drains `bpm`. History and instance tables remain shared.

## Prometheus metrics

```go
collector := metrics.NewCollector(pool, metrics.Options{Logger: log})
prometheus.MustRegister(collector)
q := queue.New(pool, queue.Options{OnAttempt: collector.ObserveAttempt})
mux.Handle("GET /metrics", promhttp.Handler())
```

| Metric | Meaning |
|---|---|
| `bpm_queue_tasks` | Pending and failed tasks by kind/status |
| `bpm_queue_lag_seconds` | Age of the oldest ready task |
| `bpm_queue_attempt_duration_seconds` | Per-replica attempt duration histogram |
| `bpm_instances` | Running and failed instances by process/status |
| `bpm_user_tasks_open` | Open user tasks by process |
| `bpm_scrape_success` | Whether database collection succeeded |

Database gauges are shared across replicas: aggregate them with `max`, rather than summing duplicate observations. Attempt histograms are local to each replica and can be summed. Collector options include namespace, query timeout and histogram buckets.

```promql
max by (kind) (bpm_queue_tasks{status="failed"}) > 0
max by (kind) (bpm_queue_lag_seconds) > 60
max(bpm_scrape_success) == 0
```

## Diagnose and recover failures

An instance becomes failed when a queue task exhausts attempts. Open failed instances in the console, inspect the error and deploy a fix or correct variables. Retry returns the task to the queue and the instance and its ancestors to running.

![Failed instance view in the Russian console](images/console-failed.png)

Use the console's variable editor to retry with corrected token data. Cancellation is available for root instances and includes their children, queued tasks, timers and open user tasks. The same actions are exposed by [HTTP and Go operations APIs](api.md#failures-and-cancellation).

A queue diagnostic query:

```sql
select id, kind, attempt, run_at, left(last_error, 200) as error
from bpm_queue
where status = 'pending' and attempt > 0
order by run_at;
```

## Demo configuration and Docker

Run `go run ./examples/demo/cmd/demo -help` for demo flags. `GO_BPM_DSN` supplies the database connection, `GO_BPM_ADDR` the listener, and `GO_BPM_RETENTION` the optional history retention period. Demo history flags default to enabled; the engine library defaults to disabled.

The repository also supplies container deployment configuration. Protect the API and console with your service's authentication middleware; user task identity fields are supplied by the caller and are not authentication.

## Tests

Use ordinary unit tests for methods and decisions, and a real PostgreSQL database for full process tests. Repository commands include `make test-unit`, `make test`, `make lint`, `make vulncheck` and `make generate`.

Integration tests clear their database. Set `GO_BPM_TEST_DSN` to a dedicated test database. See [testing processes](testing.md).
