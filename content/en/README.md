# go-bpm documentation

go-bpm is an open source BPMN engine for Go services backed by PostgreSQL. BPMN diagrams and DMN decision tables become typed Go code at build time; the engine runs that code on a task queue inside your service.

## Getting started

- [Quick start](quickstart.md): run the demo or build your own service.
- [How it works](concepts.md): tokens, steps, transactions, retries and variables.
- [BPMN modeling](bpmn.md): supported elements and their configuration.
- [DMN decisions](dmn.md): decision tables, hit policies and rule history.
- [Code generator](generator.md): generated files, data models and custom methods.

## Model and test processes

- [AI agent in the modeler](modeler-mcp.md): connect an MCP client to an open diagram.
- [Demo walkthrough](examples.md): user tasks, decisions, timers, messages and AI tasks.
- [Account opening](go-bpm-bank.md): a teaching example with subprocesses and staff tasks.
- [LLM orchestration](go-bpm-ai-orchestrator.md): execution, independent review, revision and human escalation.
- [Testing processes](testing.md): ordinary Go unit tests and PostgreSQL integration tests.

## Operate the service

- [Running and operating](running.md): configuration, history, retries, metrics and retention.
- [HTTP API and console](api.md): start instances, complete tasks and investigate failures.
- [Performance](performance.md): benchmark methodology and limitations.
- [Astra Linux compatibility](astra-linux.md): the tested environment and results.

## About the project

- [Roadmap](roadmap.md)
- [About us](about.md)
- [Source code](https://github.com/nicolas-vp/go-bpm)

The English guides cover the same subjects as the Russian documentation. Use the RU link to open the corresponding Russian reference with additional examples. Diagram labels, screenshots, the video and the shared demo console retain their original language.
