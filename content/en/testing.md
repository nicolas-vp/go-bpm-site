# Testing processes

Task methods, gateway conditions, DMN decisions and service handlers are ordinary Go functions. Test them with `go test`. For an entire process, use the real queue and engine with a dedicated PostgreSQL test database.

| Level | Checks | Database |
|---|---|---|
| Generation | Diagram, conditions, model types and signatures | No |
| Unit tests | Methods, conditions, decisions, stubbed handlers | No |
| End-to-end | Transitions, tasks, retries and instance outcomes | PostgreSQL |

Examples below are included from the site's [orders example](https://github.com/nicolas-vp/go-bpm-site/tree/main/examples/orders), which CI builds and tests. Original source comments and assertion messages may be in Russian.

## Data model and dependencies

The order process validates an order, sends large orders to manager approval, then notifies the customer. A mailer interface makes the external dependency replaceable.

```go file=examples/orders/internal/bpmgen/model/data_model.go decl=DataModel
```

```go file=examples/orders/internal/bpmgen/services/services.go decl=Services,Mailer
```

## Test a task method

Call the method with a model and inspect its fields; no engine or database is needed.

```go file=examples/orders/internal/bpmgen/order_approval/method_check_order.go decl=CheckOrder
```

```go file=examples/orders/internal/bpmgen/order_approval/method_check_order_test.go decl=TestCheckOrder
```

## Test a gateway condition

The expression `=amount > 1000` becomes a Go function. Test boundary cases from the same package because the function is unexported.

```go file=examples/orders/internal/bpmgen/order_approval/cond_is_big.go decl=isBig
```

```go file=examples/orders/internal/bpmgen/order_approval/method_check_order_test.go decl=TestIsBig
```

## Test a service handler

Use a fake mailer that records calls or returns an error. Construct `bpm.Job` with the fields the handler needs.

```go file=examples/orders/internal/bpmgen/handlers/notify_client.go decl=NotifyClient
```

```go file=examples/orders/internal/bpmgen/handlers/notify_client_test.go
```

## Test a DMN decision

Test both the output and matched rule IDs, which later appear in console history. Business rule methods can also be called directly: `bpm.RecordDecision` skips history outside an engine execution context.

```go file=go-bpm/examples/demo/bpmgen/test_decision/decision_decision_1.go decl=decideDecision1
```

```go file=examples/demo/decision_test.go
```

## Test the whole process

Unit tests cannot establish that gateway routes, assignment groups and retries work together. End-to-end tests start the real engine and queue on PostgreSQL, with stubbed external services.

`StartProcess` only enqueues a start, so poll instance status or user tasks with a timeout rather than assuming immediate completion.

```go file=examples/orders/internal/e2e/order_test.go decl=TestSmallOrder,TestBigOrderNeedsApproval,TestMailerDown,TestInvalidOrder
```

The environment uses two attempts and a 50 ms retry delay to make failure tests fast:

```go file=examples/orders/internal/e2e/order_test.go decl=testEnv,start,testPool
```

The mailer fake uses a mutex because handlers execute in worker goroutines:

```go file=examples/orders/internal/e2e/order_test.go decl=testEnv.waitStatus,testEnv.waitTask,fakeMailer,fakeMailer.Send,fakeMailer.Sent
```

## Run the example

Without `GO_BPM_TEST_DSN`, database tests skip. From `examples/orders`:

```sh
docker run -d --name orders-test-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=orders_test -p 5433:5432 postgres:18
GO_BPM_TEST_DSN='postgres://postgres:postgres@localhost:5433/orders_test?sslmode=disable' go test ./...
```

**The database is cleared before tests.** Use a dedicated test database. Tests share tables, so avoid `t.Parallel()` and use `go test -p 1` if database tests span packages.

## CI and engine tests

In CI, provide a PostgreSQL service and set `GO_BPM_TEST_DSN`, then run generation, `git diff --exit-code` and tests. The diff check catches diagrams changed without regenerating code.

The engine repository includes integration coverage for demo processes, user task claim/unclaim/complete, transactional starts, boundary timers and errors, decisions, call activities and generator validation. See [running and operating](running.md#tests) for commands.
