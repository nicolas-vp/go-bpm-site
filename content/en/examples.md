# Demo walkthrough

The demo service in `examples/demo` includes processes for user tasks, decisions, gateways, subprocesses, timers, messages, multi-instance and AI requests. Try them on the [live demo](/en/demo/) or run them locally using the [quick start](quickstart.md).

## TestUserTask

A manager from `managers` approves a request. Start with `{"initiator":"alice"}`. Claim the approval as `bob` in group `managers`, then complete with `{"approved":true}` to finish. With `false`, the process creates a correction task for `alice`; completing it returns to approval.

```sh
curl -XPOST localhost:8080/api/v1/processes/TestUserTask/instances \
  -d '{"variables":{"initiator":"alice"},"businessKey":"request-1"}'
```

![User task approval and correction loop](images/demo-user-task.svg)

## TestDecision

Two business rule tasks call `Decision_1` in sequence. Inputs `a=foo`, `b=bar` produce `c="correct foobar"`. Other combinations exercise different rows. Decision functions compile into Go; the console shows matched rules.

![Two business rule tasks](images/demo-decision.svg)

## TestBpmProcess

A synchronous method, service task and call activity lead to an exclusive gateway on `amount`. With `55`, the process follows one branch. With `1`, it takes the script and parallel branches. Use a string such as `"abc"` to exercise type failure and recovery: after system retries are exhausted, fix token variables and retry in instance history.

The default system retry budget is 10 attempts with increasing delay, so failure takes several minutes rather than appearing immediately.

![Gateways, script, call activity and parallel branches](images/demo-bpm-process.svg)

## testSubProcess

Three service task branches run through separate queue tasks and join before completion. This demonstrates concurrent handlers rather than synchronous methods within a single step. It can be started directly or invoked by `TestBpmProcess`.

![Parallel service task branches](images/demo-sub-process.svg)

## TestTimerAndException

An intermediate timer waits three seconds. Boundary timers and errors provide alternate paths. Demo methods normally act as stubs and follow the main path; to exercise errors or timeouts locally, make the relevant method return a matching `bpm.NewError` or wait while observing context cancellation. The [BPMN guide](bpmn.md#boundary-events) explains routing and retry behavior.

![Timers and error events](images/demo-timer-and-exception.svg)

## TestMultiInstance

A notification method runs three times. A sequential review task repeats until `approved` is true, up to three rounds. `loopCounter` exposes the current round. The live demo offers approval and another-round actions as `bob` from `managers`.

![Sequential multi-instance activities](images/demo-multi-instance.svg)

## TestMessage

A receive task waits for `paymentReceived`, correlated by the instance business key. Start with a non-empty business key, then send:

```sh
curl -XPOST localhost:8080/api/v1/messages \
  -d '{"name":"paymentReceived","correlationKey":"order-1","variables":{"paid":true}}'
```

The message advances matching waiting tokens and merges its variables. An optional `ttl` can buffer a message that arrives before the token starts waiting.

![Payment receive task](images/demo-message.svg)

## TestAiTask

Two AI tasks classify a request and assess urgency. The first writes text into `category`; the second writes JSON into `triage`. A refusal follows the `ai_refusal` boundary error. The demo uses `aistub`; your service supplies an implementation of `engine.AIClient`.

![AI classification, urgency and refusal handling](images/demo-ai-task.svg)

## Follow an instance

Open instance history to see the token path, task variables, child instances and DMN rule results. Root instances can be cancelled. Failed tasks can be retried with previous or corrected variables. These actions are also available through the [HTTP API](api.md).
