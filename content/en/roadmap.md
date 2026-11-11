# Roadmap

Work is grouped by importance and complexity, from documentation to changes in the execution model. Completed tasks remain visible, crossed out and linked to documentation. Colors indicate relative effort.

@@roadmap:stages@@

This is a plan, without promised dates. See [documentation](README.md) for current behavior and [BPMN limitations](bpmn.md#limitations) for unsupported features. Discuss priorities in [GitHub issues](https://github.com/nicolas-vp/go-bpm/issues).

## Stage 1. Documentation

Answers to the questions architects ask first, without changing the engine.

@@roadmap:docs@@

## Stage 2. Operations and security

Engine improvements that reduce custom production infrastructure.

@@roadmap:ops@@

## Stage 3. Missing BPMN features

Features built on waiting points, queued tasks and boundary timer races.

@@roadmap:bpmn@@

## Stage 4. Advanced BPMN

Changes that require richer execution semantics and compensation.

@@roadmap:advanced@@

## Design decisions

- Processes are deployed with service code and validated by compilation, tests and review.
- PostgreSQL remains the storage and queue foundation.
- Go remains the implementation language; call other services through service tasks.
