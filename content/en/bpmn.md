# BPMN modeling

Use go-bpm modeler for a palette and linter tailored to the engine. Camunda Modeler with the Camunda 8 platform also works. Save BPMN and DMN files under `process/`, including subdirectories.

## Namespaces and identifiers

The generator reads equivalent `gobpm:*` and `zeebe:*` extension names. When both are present, `gobpm:*` takes precedence. User tasks also accept Camunda 7 assignment attributes.

```xml
xmlns:gobpm="https://github.com/nicolas-vp/go-bpm/schema/1.0"
xmlns:zeebe="http://camunda.org/schema/zeebe/1.0"
```

Process IDs are unique across input files and become API names and Go package names. Element IDs are unique within a process and become Go function names. Prefer `CheckOrder` to random IDs. Reserved process package names include `handlers`, `model` and `services`.

Sequence flows use `sourceRef` and `targetRef`. Explicit incoming/outgoing declarations are optional, but must agree with the sequence flows. Gateway outgoing order determines condition evaluation order.

## Supported elements

![Supported BPMN elements, Russian labels](images/elements.svg)

| Element | Execution |
|---|---|
| Plain start/end events | Start or complete a process |
| Sequence flow | Move the token |
| Exclusive gateway | First matching condition or default flow |
| Parallel gateway | Split into every branch or join all inputs |
| Task | Synchronous Go method |
| Script task | Generated synchronous Go code |
| Service task | Queued handler with retries |
| AI task | Queued model request through your `engine.AIClient` |
| Business rule task | Generated DMN decision |
| User task | Wait for a person |
| Receive task / intermediate message catch | Wait for a correlated message |
| Intermediate timer catch | Wait for a duration or date |
| Interrupting boundary timer | Supported task, script, service and user activities |
| Interrupting boundary error | Task, script, service and AI activities |
| Embedded subprocess | Child instance |
| Call activity | Another process in the same service |
| Sequential multi-instance | Repeat an activity with a count and/or completion condition |

## Start and end events

Each process and embedded subprocess needs exactly one plain start event. Multiple end events are allowed, but the instance completes at the first reached end event. Join parallel branches before an end: the engine does not count outstanding tokens at separate end events.

Timer/message/signal starts and terminate/error/message ends are unsupported.

## Conditions and gateways

Conditions belong only on outgoing exclusive gateway flows. A task or event has a single outgoing sequence flow; insert a gateway to branch.

```xml
<bpmn:sequenceFlow id="Flow_big" sourceRef="IsBig" targetRef="Approve">
  <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">=amount &gt; 1000</bpmn:conditionExpression>
</bpmn:sequenceFlow>
```

| Expression | Generated Go equivalent |
|---|---|
| `amount > 1000` | `data.Amount > 1000` |
| `amount = 55` | `data.Amount == 55` |
| `approved` | `data.Approved` |
| `a > 1 and b < 2` | `data.A > 1 && data.B < 2` |
| `not(approved)` | `!(data.Approved)` |
| `client.age >= 18` | `data.Client.Age >= 18` |
| `len(items) > 0` | `len(data.Items) > 0` |

A leading `=` is removed. Go operators and supported FEEL-style equality/logical operators are translated; general FEEL functions are not. Model types are checked at compilation.

Exclusive gateways test conditions in order and take the first true result. Every output must have a condition or be the default flow. Without a matching condition or default, the step fails. A merging gateway with one unconditional output is a pass-through.

Parallel gateways copy variables into all branches and wait for all incoming flows at a join. Branches should write different variables: competing writes to the same name have no useful deterministic merge. Synchronous branches run sequentially within a step; service tasks permit concurrent queue execution.

## Tasks and scripts

An ordinary task calls a handwritten method with `context.Context`, `*services.Services` and `*model.DataModel`. It runs synchronously inside a step. Returning an error rolls back and retries the step. Use a service task for long operations and external effects.

A script with a result variable is an expression:

```xml
<zeebe:script expression="=name + &#34;!&#34;" resultVariable="greeting" />
```

This generates `data.Greeting = data.Name + "!"`. Without a result variable, the script is Go code inserted into a generated method; known model identifiers are rewritten to fields and needed packages are imported. Do not add a handwritten method with the same name as a generated script.

## Service tasks

```xml
<bpmn:serviceTask id="Notify" name="Notify customer">
  <bpmn:extensionElements>
    <zeebe:taskDefinition type="notifyClient" retries="3" />
  </bpmn:extensionElements>
</bpmn:serviceTask>
```

Arrival enqueues `bpm.task.notifyClient` and ends the current step. A worker calls your handler, writes updated variables and advances the process transactionally with completion.

`retries` is the attempt count; omitted or zero means one attempt. One job type has one handler across the service. `bpm.Job` provides instance, process, business key, element, job type and last-attempt information. Task headers are not read from BPMN. Default concurrency is one worker per job type per service replica.

Handlers must be idempotent: successful work can be repeated if the completion transaction fails. See [how it works](concepts.md).

## AI tasks

An AI task is a service task with `gobpm:aiTask`. The engine queues the request, calls your `engine.AIClient` and writes its response into the result variable. No handwritten task handler is needed.

```xml
<bpmn:serviceTask id="Classify" name="Classify request">
  <bpmn:extensionElements>
    <gobpm:taskDefinition type="ai" retries="3" />
    <gobpm:aiTask resultVariable="category"
      systemPrompt="Reply with one category: billing, login or other."
      prompt="Request: ${request.text}" />
  </bpmn:extensionElements>
</bpmn:serviceTask>
```

`prompt` and `resultVariable` are required. An optional `model` is passed through to your client; without it, the client chooses its default. `systemPrompt` is optional. Prompts can reference a whole variable with `=variable` or interpolate `${variable}` into text. Result fields use the data model's type: text for strings or JSON for structured results. A model refusal can be routed through an `ai_refusal` boundary error. The demo uses a stub client.

## Business rule tasks

Set the DMN decision ID and result variable using `calledDecision`. Decision tables run synchronously as generated Go code. See [DMN decisions](dmn.md).

## User tasks

```xml
<bpmn:userTask id="Approve" name="Approve request">
  <bpmn:extensionElements>
    <zeebe:assignmentDefinition candidateGroups="managers, admins" />
    <zeebe:formDefinition formKey="approve-form" />
  </bpmn:extensionElements>
</bpmn:userTask>
```

Assignments support `assignee`, `candidateUsers` and `candidateGroups`. Use literals, comma-separated candidate lists or variable references such as `=initiator`, `${initiator}` or `#{initiator}`. Nested paths are supported. Arbitrary assignment expressions and FEEL lists are rejected.

Camunda 7 `camunda:assignee`, `candidateUsers`, `candidateGroups` and `formKey` attributes take precedence when supplied alongside their Camunda 8 equivalents.

An invalid or missing assignment variable is skipped with a warning. Without any assignment or candidates, anyone can claim the task. `formKey` is a label for your frontend; the engine does not interpret it. See the [task API](api.md).

## Timers

Use ISO 8601 durations such as `PT30S`, `PT5M`, `P1D` or `P1DT2H`. Years and months are unsupported. Go duration syntax such as `2s` is invalid in BPMN.

For a date timer, use an RFC 3339 timestamp with a zone or a variable reference:

```xml
<bpmn:timerEventDefinition>
  <bpmn:timeDate xsi:type="bpmn:tFormalExpression">=dueDate</bpmn:timeDate>
</bpmn:timerEventDefinition>
```

The deadline is evaluated on arrival. Past dates fire immediately; invalid dates fail the step. Timers are persisted queue tasks, survive restarts and do not occupy a worker while waiting. Accuracy is roughly the queue polling interval, about one second. Repeating `timeCycle` timers are unsupported.

## Boundary events

Only interrupting timer and error boundary events are supported. Timer events can attach to tasks, script tasks, service tasks and user tasks. Error events attach to tasks, script tasks, service tasks and AI tasks. They do not attach to user tasks, subprocesses or call activities.

For a service task timeout, handler completion and timer compete. The winner advances the process; the loser's result is ignored. A handler already executing is not forcibly stopped. For synchronous methods, the timeout uses context cancellation, so the method must observe `ctx`. A user task timer cancels the task and follows the escalation flow; subsequent completion returns HTTP 409.

A business error can be caught by code:

```go
return bpm.NewError("LIMIT_EXCEEDED", "customer limit exceeded")
```

A boundary error with `errorRef` matches that code; one without a reference catches any error eligible for routing. Error details become `errorCode` and `errorMessage` variables.

For task/script methods, uncaught errors retry the step. For service handlers, an uncaught `bpm.Error` fails immediately. Ordinary handler errors retry until attempts run out; a catch-all boundary error can then route the final failure.

## Messages

A receive task or intermediate message catch waits for a constant message name plus a correlation key:

```xml
<bpmn:receiveTask id="WaitPayment" messageRef="Message_payment" />
<bpmn:message id="Message_payment" name="paymentReceived">
  <bpmn:extensionElements>
    <zeebe:subscription correlationKey="=orderId" />
  </bpmn:extensionElements>
</bpmn:message>
```

Without a subscription override, the instance business key is used. A variable key is evaluated when the token starts waiting and can be a string or number. Empty keys fail the step.

Send with `engine.CorrelateMessage` or `POST /api/v1/messages`. Variables merge into every currently waiting matching token. An optional TTL buffers an unmatched message for the first matching future subscriber; without TTL it returns 404. `CorrelateMessageTx` correlates within a service transaction. Message starts and message boundary events are unsupported.

## Subprocesses and call activities

Embedded subprocesses create child instances named `<process>.<subprocess ID>`. They receive a copy of parent variables and return variables at completion. Nested embedded subprocesses and boundary events on subprocesses are unsupported.

A call activity invokes another process from the same service:

```xml
<zeebe:calledElement processId="testSubProcess" propagateAllChildVariables="false" />
```

The child inherits the business key and a copy of variables. Propagation defaults to true; false preserves the parent's previous variables. Input/output mappings are not supported.

## Sequential multi-instance

Activities, embedded subprocesses and call activities can repeat sequentially:

```xml
<bpmn:multiInstanceLoopCharacteristics isSequential="true">
  <bpmn:loopCardinality xsi:type="bpmn:tFormalExpression">3</bpmn:loopCardinality>
  <bpmn:completionCondition xsi:type="bpmn:tFormalExpression">=approved</bpmn:completionCondition>
</bpmn:multiInstanceLoopCharacteristics>
```

A positive integer cardinality and/or a completion condition is required. With both, the first limit reached ends the loop. Conditions are evaluated after each execution. `loopCounter` starts at one; use an `omitempty` model field so methods outside loops do not overwrite it with zero. The variable is removed after the loop.

Variables pass from one iteration to the next. Task and script iterations are separate queued steps, so a failed iteration retries independently. A boundary error interrupts the entire loop. Boundary timers on multi-instance activities are unsupported.

## Limitations

Unsupported elements are rejected during generation: inclusive/event-based/complex gateways, manual/send tasks, throw and signal events, event or nested embedded subprocesses, transactions, ad-hoc subprocesses, standard loops, parallel or collection-based multi-instance, and non-plain starts/ends.

Also rejected are non-interrupting boundaries, unsupported boundary attachments or types, invalid references, missing required task properties and unsupported assignment/timer expressions.

Some extensions are ignored rather than rejected: `zeebe:ioMapping`, `zeebe:taskHeaders` and execution listeners. Documentation, lanes, annotations and data objects do not execute. General FEEL functions are not translated and can fail compilation.
