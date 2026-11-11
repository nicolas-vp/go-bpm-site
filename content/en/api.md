# HTTP API and console

The JSON API and console are ordinary handlers on your service's `http.ServeMux`. Register only the parts you need.

```go
httpapi.Register(mux, e, history)
httpapi.RegisterUserTasks(mux, engine.NewUserTaskService(pool, e))
httpapi.RegisterMessages(mux, e)
httpapi.RegisterOperations(mux, engine.NewInstanceOperations(pool, e))
webui.Register(mux, history)
```

The console uses the API on the same origin. Authentication and authorization belong to your service. Derive task user IDs and groups from authenticated identity rather than trusting client request bodies.

Errors have the form `{"error":"message"}`. IDs are UUIDs; timestamps are RFC 3339. Common response codes are 400 for invalid input, 403 for forbidden task claims, 404 for missing resources and 409 for state conflicts.

## Processes

| Method and path | Purpose |
|---|---|
| `GET /api/v1/processes` | Root processes and instance counts by status |
| `GET /api/v1/processes/{name}` | Process ID, name and BPMN XML |
| `POST /api/v1/processes/{name}/instances` | Start an instance |
| `GET /api/v1/processes/{name}/heatmap` | Element counts and duration aggregates |
| `GET /api/v1/processes/{name}/stats` | Instance outcomes and durations by start date |

Start request:

```json
{"variables":{"amount":5000,"category":"vip"},"businessKey":"order-2"}
```

Both fields are optional. A successful start returns `201 {"id":"..."}` after enqueueing, before process execution. With database history enabled, the instance is immediately visible as running.

Counts and analytics require `History.ToDB`. Heatmap accepts RFC 3339 `from` (inclusive), `to` (exclusive) and `limit` (default 1000, maximum 10000). It returns execution counts, active waits, total/average/minimum/maximum milliseconds and a `truncated` flag when the selected sample is limited.

Statistics accepts `from`, `to` and an IANA `tz`, defaulting to the last 30 days in UTC, with a maximum period of 366 days. Outcomes are grouped by instance start date and reflect current status. Median and p95 use completed instances; period totals are calculated from instances, not daily averages.

## Instances

| Method and path | Purpose |
|---|---|
| `GET /api/v1/instances` | Filtered newest-first list |
| `GET /api/v1/instances/{id}` | Instance details |
| `GET /api/v1/instances/{id}/history` | Element history |

List filters include `processName`, `status`, exact `businessKey`, RFC 3339 `from`/`to`, `limit` (default 100, maximum 1000) and `before` (the last instance ID on the previous page). States are `running`, `completed`, `failed` and `cancelled`. Embedded subprocess names use `<process>.<subprocess ID>`.

```json
{"instances":[{"instance_uid":"...","process_name":"OrderApproval","business_key":"order-2","status":"running","created_at":"...","updated_at":"..."}]}
```

History returns `{"records":[...]}`. Use `?withChildren=true` to include subprocesses and call activities. Records contain event, instance, process, element and timestamp; start/task events include variables. Events are `activated`, `completed` or `terminated`. Child records include their parent instance and element IDs. No history records returns 404.

## User tasks

`GET /api/v1/tasks` supports `assignee`, `candidateUser`, repeatable or comma-separated `candidateGroup`, `processName`, `instanceId`, `businessKey`, `status`, `limit` and `before`. The default task status is `created`; `completed` selects completed tasks.

Assignment and candidate filters combine as OR: request personal tasks and tasks the user can claim in one query. Candidate results include unclaimed matching tasks and tasks without assignment restrictions.

`GET /api/v1/tasks/{id}` includes variables captured when the task was created, suitable for a form.

| Operation | Request body | Success |
|---|---|---|
| `POST /api/v1/tasks/{id}/claim` | `{"userId":"bob","groups":["managers"]}` | 204 |
| `POST /api/v1/tasks/{id}/unclaim` | `{"userId":"bob"}` | 204 |
| `POST /api/v1/tasks/{id}/complete` | `{"userId":"bob","variables":{"approved":true}}` | 204 |

Only candidates can claim restricted tasks. Claiming your own task again is idempotent. Only the assignee can unclaim or complete. Completed, cancelled or differently assigned tasks produce state conflicts. Completion merges variables and enqueues continuation; errors in later process steps are handled asynchronously.

Equivalent Go operations are available through `engine.UserTaskService`. Compare its sentinel errors with `errors.Is`.

## Failures and cancellation

`GET /api/v1/instances/{id}/failures` lists exhausted queue tasks for the instance and its children. Each entry includes `task_id`, kind, instance, process, element, attempts, failure time, error and token variables. The element ID is where the queued step started; the error may identify a later synchronous method.

`POST /api/v1/failures/{taskId}/retry` resets attempts and returns the instance and its ancestors to running. An optional body replaces the **entire** token variable object:

```json
{"variables":{"amount":55}}
```

Without variables, retry preserves the previous data, useful after deploying a code fix. Success returns 204; a missing or no-longer-failed task returns 404.

`POST /api/v1/instances/{id}/cancel` cancels a root instance and its children, queued/failed work, timers and open user tasks. It returns 204, 404 if missing, or 409 for a child or finished instance. Cancelled user tasks cannot be completed.

Go equivalents are `ListFailures`, `Retry` and `Cancel` on `engine.InstanceOperations`.

## Messages

`POST /api/v1/messages` correlates a message with waiting receive tasks and intermediate message catches:

```json
{"name":"paymentReceived","correlationKey":"order-1","variables":{"paid":true},"ttl":"10m"}
```

Name and key are required. Variables merge into each receiving token. Optional `ttl` uses Go duration syntax and buffers an unmatched message for a future subscriber. The response is `200 {"correlated":1}`; zero means it was buffered. Without a TTL, no matching subscriber returns 404. Invalid bodies or TTLs return 400.

The Go equivalents are `CorrelateMessage` and `CorrelateMessageTx`, with `engine.WithMessageTTL` as needed. See [message modeling](bpmn.md#messages).

## Console

The console lives at `/ui/`, with instance diagrams at `/ui/history/`. Its JavaScript, bpmn-js and dmn-js assets are embedded in the service binary. The shared public demo console is in Russian.

- Processes: diagrams and status counters.
- Start: process selection, business key and typed variables.
- Instances: filters, parents, children and history links.
- Tasks: personal tasks and candidate queues, claim/unclaim/complete.
- Heatmap: time spent and execution counts by diagram element.
- Statistics: outcomes by day, median and p95 duration.
- Instance history: token path, task variables, matched DMN rules, failures, variable correction, retries and cancellation.

![Console process view](images/console-processes.png)
