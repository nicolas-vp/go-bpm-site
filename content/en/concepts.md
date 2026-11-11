# How it works

go-bpm turns a process diagram into Go code before the service starts. The engine uses PostgreSQL to persist queued work, token state, instances and optional history.

## BPMN and tokens

BPMN is a standard notation for business processes. A `.bpmn` file is XML containing executable elements and diagram coordinates.

Think of execution as a token moving through a diagram. A start event creates the token; sequence flows move it to tasks; exclusive gateways select one branch and parallel gateways create several. An end event finishes a token. One run of a process is an **instance**, with its own variables and business key.

![Supported BPMN elements, with Russian labels](images/elements.svg)

| Element | Purpose |
|---|---|
| Task | Run a Go method synchronously |
| Service task | Enqueue a handler with its own retries |
| Script task | Calculate values in a generated Go function |
| User task | Wait for a person |
| Business rule task | Evaluate a DMN table |
| Exclusive gateway | Choose the first matching branch or the default |
| Parallel gateway | Split into all branches or wait for all inputs |
| Intermediate timer | Wait for a duration or date |
| Boundary event | Interrupt a supported task on an error or timeout |
| Subprocess / call activity | Run a child process instance |

See [BPMN modeling](bpmn.md) for supported configurations and restrictions.

## DMN decisions

DMN represents business rules in decision tables. Each row is a rule, input cells are tests, and output cells contain results. All input tests must match for a row to apply. The hit policy determines whether to take the first match, return multiple matches or aggregate them.

| Amount | Customer category | Discount |
|---|---|---|
| `>= 10000` | `"vip"` | `15` |
| `>= 10000` | `-` | `10` |
| `-` | `"vip"` | `5` |
| `-` | `-` | `0` |

A business rule task reads inputs from process variables and writes the result back. See [DMN decisions](dmn.md).

## Generation and deployment

```text
process/*.bpmn and *.dmn
         ↓ go-bpm-gen
Go transitions, conditions, scripts, decisions and method stubs
         ↓ go build
your service + engine + queue → PostgreSQL
```

The generator builds the generated packages, catching invalid expressions, missing model fields and incorrect method signatures. Processes are versioned and reviewed with service code. Changing a process requires rebuilding and deploying the service.

## Queued steps

Workers claim tasks with `SELECT ... FOR UPDATE SKIP LOCKED`. Multiple service replicas can share the same database.

A queue task executes one **step**. A step moves the token through synchronous elements until it must wait: a service task, timer, user task, message, parallel join or process completion. A chain of ordinary methods can run in one step. Parallel branches containing only synchronous methods execute sequentially within that step; service task branches can run concurrently as separate queue tasks.

## Transaction guarantees

Queue task completion, new queued tasks, element and rule history, instance state, user tasks and parallel join counters commit in one database transaction. A step either commits entirely or rolls back entirely.

The queue retries a rolled-back step, including its synchronous methods. External effects such as payments and email need idempotent service task handlers. Service tasks provide at-least-once execution: a handler may finish before its completion transaction fails. An idempotency key can combine `job.InstanceUid` and `job.ElementId`; for a repeated activity, include its logical occurrence as needed by your integration.

## Retries and failures

| Failure | Result |
|---|---|
| Method, condition or DMN evaluation | Roll back and retry the system step |
| Service task handler | Retry its queue task according to BPMN `retries` |
| Matching `bpm.NewError(code, ...)` | Follow the boundary error flow |
| Attempts exhausted | Mark the task, instance and ancestors as failed |

System steps default to 10 attempts. Retry delay starts at one second, doubles up to five minutes and includes jitter. Failed tasks remain available for diagnosis and manual retry.

## Variables and the data model

Instance variables are JSON. Methods receive a typed `*model.DataModel`; JSON tags map fields to variables. The model is shared by all processes in a service.

After a method, all model fields are merged back, including zero values. A zero-valued `omitempty` field leaves its variable unchanged. Variables absent from the model are preserved. Nested structs support paths such as `client.age`.

## Instances and history

Database history is optional and disabled by default. Enable `History.ToDB` for instances and element history, and `RulesHistory.ToDB` for matched DMN rules.

Element events are `activated`, `completed` and `terminated`. Instance states include `running`, `completed`, `failed` and `cancelled`. A new instance is visible as running as soon as its start task is committed, even before the first step executes.

Embedded subprocesses and call activities create child instances with a parent link, copied variables and an inherited business key. The parent waits for completion. Call activities normally propagate child variables back; `propagateAllChildVariables=false` keeps the parent's previous variables. A failed child also fails its ancestors.

## User tasks

An assignee receives a personal task. Candidate users or groups share a queue; one person claims the task, then completes it. Unclaim returns it to the candidate queue. With no assignment restrictions, anyone can claim it. Only the assignee can complete a task; submitted variables merge into the process.

The service supplies user IDs and groups and implements authentication and authorization. The engine does not authenticate users.
