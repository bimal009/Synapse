---
name: dag-workflow-planner
description: >
  Decompose any complex objective into a validated, executable DAG (Directed
  Acyclic Graph) of atomic tasks with explicit dependencies, status tracking,
  validation gates, and maximized parallelism. Use this skill whenever the user
  asks to plan, break down, decompose, orchestrate, or sequence a multi-step
  project; whenever you need to turn a high-level goal into tasks an executor or
  multi-agent system can run; and whenever words like "workflow", "pipeline",
  "task graph", "DAG", "dependencies", "parallelize", "project plan", or "build
  X end to end" appear — even if the user does not say "DAG" explicitly. Always
  reach for this skill before emitting an ad-hoc linear checklist for any task
  with more than ~4 steps or any real dependency structure.
---

# DAG Workflow Planner

A reference manual for transforming any objective into a correct, parallel,
fault-tolerant Directed Acyclic Graph of atomic tasks. This skill is written so
that an AI planner can consume a user goal and emit a DAG file that an executor
(a scheduler, a multi-agent runtime, a CI system) can run without further human
disambiguation.

---

## 1. Purpose

### What a DAG is

A **DAG** is a set of **nodes** (tasks) connected by **directed edges**
(dependencies) with **no cycles**. An edge `A → B` means "B may not start until A
has completed." Because there are no cycles, the graph always has at least one
task with no unmet dependencies, so execution can always make progress, and the
graph is guaranteed to terminate.

```
        ┌───────┐
        │  A    │
        └───┬───┘
       ┌────┴────┐
       ▼         ▼
   ┌───────┐ ┌───────┐
   │  B    │ │  C    │      B and C depend on A.
   └───┬───┘ └───┬───┘      D depends on both B and C.
       └────┬────┘
            ▼
        ┌───────┐
        │  D    │
        └───────┘
```

### Why DAG planning beats a linear task list

A linear list (`1 → 2 → 3 → 4`) encodes a _total order_ even when the real work
has only a _partial order_. That over-constrains execution: step 3 waits for step
2 even when it never needed step 2's output. A DAG encodes only the dependencies
that actually exist, which unlocks the benefits below.

| Property           | Linear list                          | DAG                                   |
| ------------------ | ------------------------------------ | ------------------------------------- |
| Parallelism        | None — everything serial             | Anything independent runs at once     |
| Dependency clarity | Implicit (order = dependency)        | Explicit edges                        |
| Fault isolation    | A failure stalls everything after it | Only downstream of the failure stalls |
| Resumability       | Restart from a point                 | Re-run only failed/affected nodes     |
| Reuse              | Copy-paste                           | Subgraphs drop in as modules          |

### Core benefits

- **Atomic tasks** — each node does one thing, so it is easy to write, test,
  validate, retry, cache, and reason about. Failure points are small and precise.
- **Parallel execution** — independent branches run simultaneously, collapsing
  wall-clock time to the length of the _critical path_ (the longest dependency
  chain), not the sum of all work.
- **Fault tolerance** — a failed node blocks only its descendants. Siblings keep
  running. Retries and recovery are scoped to the smallest unit.
- **Validation** — explicit validation nodes turn "I hope this worked" into a
  gate the graph cannot pass until a check succeeds.
- **Maintainability** — small, named, single-purpose nodes are self-documenting
  and safe to edit in isolation.
- **Reusability** — a well-bounded subgraph (e.g., "lint → test → build") is a
  reusable module you can paste into many DAGs.

---

## 2. Core Principles

Internalize these. Every later section is an application of them.

1. **One responsibility per task.** If a task's description needs the word "and"
   to describe distinct work, split it. "Write and test the parser" is two tasks.
2. **Smallest meaningful atomic unit.** Atomic does not mean trivial — it means
   independently runnable, validatable, and retryable. "Add one log line" is too
   small to be a node; "implement the auth middleware" is right; "build the
   backend" is too big.
3. **Never combine unrelated work.** Bundling unrelated steps destroys
   parallelism and makes failures ambiguous (which half failed?).
4. **Every dependency is explicit.** If B needs A's output, write the edge. Never
   rely on list ordering, file-system side effects, or "it'll probably finish
   first" to encode a dependency.
5. **No circular dependencies.** A cycle means nothing can start. Cycles are
   always a modeling error — break them by introducing an intermediate artifact
   or splitting a task.
6. **Maximize parallelism.** After dependencies are set, ask of every pair of
   tasks: "does an edge between these actually exist?" If not, leave them
   unconnected so they can run together.
7. **Insert validation between phases.** Put a gate after each phase that
   produces something downstream phases trust (schema validated, tests pass,
   build succeeds) before unlocking the next phase.
8. **Deterministic outputs.** A task given the same inputs should produce the
   same outputs. Non-determinism makes validation and caching meaningless.
9. **Idempotent tasks.** Running a task twice should be safe and yield the same
   end state. This is what makes retries and resumes safe.
10. **Readable DAGs.** Names and structure should let a human trace the flow in
    seconds. If you can't, the executor can't either.
11. **Prefer shallow over deep.** A long thin chain has a long critical path and
    little parallelism. Look for hidden independence that lets you widen and
    shorten the graph.
12. **Design for recovery and retries.** Assume any node can fail. Give nodes a
    retry policy and design the graph so a single failure is recoverable without
    a full restart.

---

## 3. Task Schema

Every task in a DAG file uses this exact structure. Fields marked _required_ must
always be present; others have sensible defaults.

```yaml
- id: build_api_schema # required: unique, stable, snake_case identifier
  title: Define API schema # required: short human label
  description: > # required: what the task does, in prose
    Produce the OpenAPI 3.1 schema covering all CRUD endpoints for the resource.
  objective: > # the single success condition, testable
    A valid openapi.yaml exists describing every endpoint, request, and response.
  inputs: # named artifacts/data this task consumes
    - requirements.md
  outputs: # named artifacts this task produces
    - openapi.yaml
  dependencies: [gather_requirements] # required: list of task ids that must complete first
  status: pending # required: see Section 4 lifecycle
  validation: # checks that must pass for status → completed
    - openapi.yaml parses as valid OpenAPI 3.1
    - every endpoint has a documented response schema
  retry_policy: # how to handle failure
    max_attempts: 3
    backoff: exponential # none | fixed | exponential
    backoff_seconds: 5
  timeout: 600 # seconds before the task is force-failed
  priority: 2 # higher = scheduled first among ready tasks
  tags: [design, backend] # free-form labels for filtering/grouping
  owner: planner # optional: human-readable label for who runs it
  model_role: reasoning # the agent ROLE that runs this task; one of the fixed roles
```

### Field reference

| Field          | Required    | Meaning                                                                                                                                                                                              |
| -------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`           | yes         | Unique, stable identifier. Edges reference this. Never reuse or rename mid-run.                                                                                                                      |
| `title`        | yes         | One-line human-readable name.                                                                                                                                                                        |
| `description`  | yes         | What the task does. Specific enough that the executor needs no guesswork.                                                                                                                            |
| `objective`    | recommended | The single, testable success condition. The "definition of done."                                                                                                                                    |
| `inputs`       | recommended | Named artifacts/data consumed. Makes data-flow (not just control-flow) explicit.                                                                                                                     |
| `outputs`      | recommended | Named artifacts produced. Downstream tasks list these as inputs.                                                                                                                                     |
| `dependencies` | yes         | List of `id`s that must reach `completed` before this task becomes `ready`. Empty list = root task.                                                                                                  |
| `status`       | yes         | Current lifecycle state (Section 4).                                                                                                                                                                 |
| `validation`   | recommended | Concrete checks. All must pass to move `validating → completed`.                                                                                                                                     |
| `retry_policy` | recommended | `max_attempts`, `backoff` strategy, `backoff_seconds`.                                                                                                                                               |
| `timeout`      | recommended | Hard limit in seconds; exceeding it fails the task (subject to retries).                                                                                                                             |
| `priority`     | optional    | Tie-breaker when multiple tasks are `ready` and resources are scarce.                                                                                                                                |
| `tags`         | optional    | Labels for grouping, filtering, routing to agent types.                                                                                                                                              |
| `owner`        | optional    | Free-form human-readable label for who runs the task. Decorative; routing is driven by `model_role`.                                                                                                 |
| `model_role`   | recommended | The agent role that runs this task. Must be one of the fixed roles available for the chat (`planner`, `reasoning`, `coder`, `tester`, `qa`, `analyst`, `reviewer`, `researcher`, `vision`, `architect`, `designer`, `devops`, `security`, `writer`, `editor`, `data`, `summarizer`, `general`). Never a concrete model name. |

**Why `inputs`/`outputs` matter even though `dependencies` exists.** `dependencies`
encodes _control flow_ (ordering). `inputs`/`outputs` encode _data flow_. Keeping
both lets a validator catch a whole class of bugs: if task B lists `inputs:
[schema.json]` but no completed ancestor produces `schema.json`, the DAG is
broken even though the dependency list might look fine.

### Routing: `model_role`

`model_role` names the **agent role** that runs the task. It is the routing key:
the runtime maps the role to whichever concrete model the user has configured for
that role on _this_ machine, at dispatch time. `owner` is just a human-readable
label and does not affect routing.

**Use a role, never a model name.** The planner must not write
`model_role: "qwen2.5-coder:32b"`. It writes a role like `coder`, and the runtime
resolves role → concrete model at dispatch, reading the chat's live model config.
This matters for two reasons:

1. **Grounding/portability.** Every machine has a different model set. A hardcoded
   tag breaks the moment a user hasn't pulled it; a role always resolves to
   _something they actually have configured_.
2. **No snapshot coupling.** Resolving at dispatch (not at planning time) means a
   DAG planned yesterday still runs after the user swaps a model behind a role.

**Only use roles that are available.** Each run, the planner is told which roles
are configured for the chat (under _Available agent roles_). Assign `model_role`
only from that list, using the exact role name. The full role vocabulary is:
`planner` (decomposition, orchestration), `reasoning` (analysis, review),
`coder` (implementation), `tester` (writing/running tests), `qa` (quality),
`analyst`, `reviewer`, `researcher`, `vision`, `architect`, `designer`, `devops`,
`security`, `writer`, `editor`, `data`, `summarizer`, and `general` (cheap,
high-volume steps).

### How the DAG is stored and run (Synapse)

The DAG is **not a file**. The planner builds the whole plan as one JSON object
and saves it with the `create_dag` tool, which validates it and stores it in the
database for the chat. Read the current plan back with `get_dag` (it takes no
arguments and returns the stored JSON) — never read a `dag.json` file.

### Task fields are the prompt source

A task's `description` + `objective` + `inputs` + `outputs` _already are_ its
instructions. The runtime builds the agent's actual prompt at dispatch by
composing the role template for `model_role` + this task's fields + the user's
instruction layers. Because the stored DAG is the single source of truth, there
is nothing to drift out of sync. Practical consequence: **write `description` and
`objective` as if they are the instructions the agent will receive — because,
rendered, they are.**

---

## 4. Task Status Lifecycle

A task moves through a state machine. The executor owns these transitions; the
planner's job is to emit every task as `pending` and let the runtime drive the
rest.

### States

| Status       | Meaning                                                                                               |
| ------------ | ----------------------------------------------------------------------------------------------------- |
| `pending`    | Created but dependencies not yet evaluated. The initial state of every task.                          |
| `blocked`    | At least one dependency is not yet `completed`. Cannot run.                                           |
| `ready`      | All dependencies `completed`; eligible to be scheduled.                                               |
| `waiting`    | Eligible but parked — waiting on a scheduler slot, a resource, or external/human input.               |
| `running`    | Actively executing.                                                                                   |
| `validating` | Execution finished; validation checks are being evaluated.                                            |
| `completed`  | Validation passed; outputs are available to dependents. Terminal (success).                           |
| `failed`     | Execution or validation failed and retries are exhausted. Terminal (failure).                         |
| `retrying`   | Failed an attempt, but a retry is scheduled (attempts remain).                                        |
| `skipped`    | Deliberately not run — a conditional branch wasn't taken, or an optional dependency failed. Terminal. |
| `cancelled`  | Externally stopped or made obsolete (e.g., the planner re-planned). Terminal.                         |

### State transition diagram

```
                         ┌─────────┐
                         │ pending │
                         └────┬────┘
                  deps unmet  │  deps met
                 ┌────────────┴────────────┐
                 ▼                          ▼
            ┌─────────┐                ┌─────────┐
            │ blocked │ ── deps met ─▶ │  ready  │
            └─────────┘                └────┬────┘
                 ▲                          │ picked up
        a dependency fails &               ▼
        propagation = block           ┌─────────┐
                 │                     │ waiting │ (slot / resource / human)
                 │                     └────┬────┘
                 │                          ▼
                 │                     ┌─────────┐
                 │                     │ running │
                 │                     └────┬────┘
                 │              success │   │ error / timeout
                 │                      ▼   ▼
                 │            ┌────────────┐  ┌──────────┐
                 │            │ validating │  │ retrying │◀─ attempts remain
                 │            └─────┬──────┘  └────┬─────┘
                 │        pass │    │ fail        │ re-enter
                 │             ▼    ▼             ▼
                 │      ┌───────────┐ ┌────────┐ (back to running)
                 └──────│ completed │ │ failed │◀─ attempts exhausted
                        └───────────┘ └───┬────┘
                                          │ failure propagates
                                          ▼
                              descendants → skipped / blocked
                                          │
                              (any non-terminal) ──▶ cancelled
```

### Allowed transitions

| From                            | To           | Trigger                                                                                     |
| ------------------------------- | ------------ | ------------------------------------------------------------------------------------------- |
| `pending`                       | `blocked`    | One or more dependencies not yet `completed`.                                               |
| `pending`                       | `ready`      | Task has no dependencies, or all are already `completed`.                                   |
| `blocked`                       | `ready`      | The last outstanding dependency reaches `completed`.                                        |
| `ready`                         | `waiting`    | Picked for execution but parked for a slot/resource/human input.                            |
| `ready` / `waiting`             | `running`    | Scheduler grants a slot and starts execution.                                               |
| `running`                       | `validating` | Execution finished without error.                                                           |
| `running`                       | `retrying`   | Error or timeout, and `attempts < max_attempts`.                                            |
| `running`                       | `failed`     | Error or timeout, and attempts are exhausted (no `validation` retry left).                  |
| `validating`                    | `completed`  | All validation checks pass.                                                                 |
| `validating`                    | `retrying`   | A check failed and attempts remain.                                                         |
| `validating`                    | `failed`     | A check failed and attempts are exhausted.                                                  |
| `retrying`                      | `running`    | Backoff elapsed; new attempt begins.                                                        |
| `blocked` / `ready` / `waiting` | `skipped`    | An upstream dependency `failed` (and policy = skip), or a conditional branch was not taken. |
| any non-terminal                | `cancelled`  | External cancel or re-plan made the task obsolete.                                          |

### Dependency behavior

A task becomes `ready` the instant **all** its `dependencies` are `completed`.
It is `blocked` while any dependency is in a non-`completed`, non-terminal state.

### Failure propagation

When a task reaches `failed`, the executor walks its descendants and applies the
DAG's failure policy:

- **block** (default) — descendants stay `blocked`; a human or recovery node must
  intervene. Use when downstream work is meaningless without this output.
- **skip** — descendants become `skipped`, and _their_ descendants are
  re-evaluated. Use for optional branches.
- **recover** — a designated error-recovery node (Section 7) becomes `ready` to
  attempt a fix or fallback.

### Retry behavior

On failure with attempts remaining, the task enters `retrying`, waits per its
`backoff` strategy, then returns to `running`. Only when `attempts` reach
`max_attempts` does it become terminal `failed`. Because tasks are **idempotent**
(Principle 9), a retry is always safe.

---

## 5. Dependency Management

Dependencies are the edges of the DAG. Model them precisely — most DAG bugs are
dependency bugs.

### Primitives

- **`depends_on` (fan-in)** — a task lists multiple `dependencies`; it waits for
  all of them. This is a **merge / synchronization barrier**.
- **fan-out** — multiple tasks list the _same_ single dependency; they all
  unlock together and run in parallel.
- **parent/child** — informal terms: a "parent" is an upstream dependency, a
  "child" is a downstream dependent. The edge direction is parent → child.
- **merge node** — a task that fans in several branches to combine their outputs
  (e.g., `assemble_report` depending on every research branch).
- **synchronization barrier** — a node whose only job is to wait for a whole
  phase to finish before the next phase starts. Often a validation gate.

### Branching

- **conditional branch** — a task produces a decision; downstream tasks are
  guarded by a condition on that output. The branch not taken is `skipped`.

  ```yaml
  - id: route_by_size
    outputs: [route] # "small" or "large"
    dependencies: [measure_dataset]
  - id: train_small_model
    condition: route == "small" # only becomes ready if condition holds
    dependencies: [route_by_size]
  - id: train_large_model
    condition: route == "large"
    dependencies: [route_by_size]
  ```

- **optional branch** — a branch whose failure should not fail the whole DAG. Mark
  its edge optional so a `failed` optional parent yields `skipped` children, not
  `blocked` ones.

  ```yaml
  - id: generate_nice_to_have_charts
    optional: true
    dependencies: [compute_metrics]
  ```

- **reusable subgraph** — a named, self-contained cluster of tasks with one entry
  dependency and one exit output, designed to be dropped into many DAGs (e.g., a
  `ci_checks` subgraph of lint → typecheck → unit-test → build).

### Validating dependencies

Before executing any DAG, run these checks (the planner should emit only DAGs
that pass them):

1. **Existence** — every id in every `dependencies` list refers to a real task.
2. **Acyclicity** — a topological sort succeeds (no cycles). If it fails, the
   sort reports the nodes in the cycle.
3. **Reachability** — every task is reachable from a root, and every task can
   reach a terminal/output task (no orphans, no dead ends).
4. **Data-flow soundness** — every artifact named in a task's `inputs` is in the
   `outputs` of some completed-before ancestor.
5. **Single-writer (recommended)** — no two tasks claim to produce the same
   output artifact, which would make the source ambiguous.

A standard way to test acyclicity _and_ compute execution order at once is
**Kahn's algorithm**: repeatedly remove tasks with zero unmet dependencies; if
you remove all tasks, the graph is acyclic and the removal order is a valid
execution order; if tasks remain but none have zero unmet dependencies, those
remaining tasks form a cycle.

---

## 6. Planning Methodology

A repeatable, ordered process. Follow it every time; do not skip the validation
steps at the end.

1. **Understand the objective.** Restate the goal in one sentence and write down
   the final deliverable(s). If you can't name the deliverable, you can't build
   the DAG. Surface hidden constraints (deadlines, tech stack, quality bars).
2. **Identify phases.** Group the work into 3–7 coarse phases (e.g., _design →
   build → test → ship_). Phases are thinking aids, not nodes.
3. **Split phases into atomic tasks.** Within each phase, decompose to the
   smallest meaningful unit (Principle 2). Each task gets one responsibility.
4. **Identify dependencies.** For each task ask: "what must be true before this
   can start?" Add an edge for each real prerequisite — and _only_ real ones.
5. **Discover parallel work.** Scan for tasks with no edge between them. Those
   can run together. Actively look for false serialization to remove.
6. **Insert validation tasks.** After any phase whose output the next phase
   trusts, add a gate node that asserts correctness before unlocking downstream.
7. **Add completion tasks.** Add explicit terminal node(s) that fan in the final
   deliverables (e.g., `assemble_release`) so "done" is a real node, not implicit.
8. **Check for cycles.** Run a topological sort / Kahn's algorithm. Fix any cycle
   by introducing an intermediate artifact or splitting the offending task.
9. **Optimize for parallelism.** Shorten the critical path: can a long chain be
   widened? Can a serial pair become independent by precomputing a shared input?
10. **Validate the entire DAG.** Run all dependency checks from Section 5.4. Only
    emit a DAG that passes existence, acyclicity, reachability, and data-flow.

---

## 7. DAG Design Patterns

Reach for a known pattern before inventing structure. Most real workflows are a
composition of these.

### Sequential

A straight chain. Use only when each step genuinely needs the previous one's
output. Minimize these — they have the longest critical path and no parallelism.

```
A → B → C → D
```

### Parallel (fan-out)

One root unlocks many independent tasks at once.

```
        ┌─▶ B
   A ───┼─▶ C
        └─▶ D
```

### Fan-out / Fan-in

Fan out to parallel work, then fan in to a merge node. The backbone of most
real DAGs.

```
        ┌─▶ B ─┐
   A ───┼─▶ C ─┼─▶ E
        └─▶ D ─┘
```

### Map-Reduce

A _map_ node splits work into N shards; N identical _worker_ nodes process them
in parallel; a _reduce_ node aggregates. Same shape as fan-out/fan-in but the
workers are homogeneous and data-parallel.

```
   split ─┬─▶ map_1 ─┐
          ├─▶ map_2 ─┼─▶ reduce
          └─▶ map_3 ─┘
```

### Pipeline

Stages where each transforms the previous stage's output, but multiple items can
be _in flight_ across stages. Logically sequential per item, parallel across
items: `ingest → clean → transform → load`.

### Tree expansion

A node spawns children, each of which spawns its own children — natural for
hierarchical decomposition (an outline → chapters → sections → paragraphs).

### Multi-stage workflow

Phases separated by barriers, each phase internally parallel:

```
[ design phase ] ─▶ [GATE] ─▶ [ build phase ] ─▶ [GATE] ─▶ [ test phase ]
   (parallel)                    (parallel)                  (parallel)
```

### Approval workflow

A `waiting` human-checkpoint node sits between automated phases. The graph cannot
proceed until an external approval flips it to `completed`.

```
build → run_tests → request_approval (waiting:human) → deploy
```

### Validation gate

A node whose sole job is to assert a property. Downstream depends on it, so a
failed gate blocks the phase. Cheap insurance against propagating bad state.

### Error-recovery DAG

Pair a risky node with a recovery node reachable on failure (failure policy =
recover). The recovery node attempts a fix, fallback, or rollback, then rejoins
the main flow.

```
   deploy ──(fail)──▶ rollback ──▶ alert
      │
   (success)──▶ smoke_test
```

---

## 8. Anti-Patterns

Recognize and refactor these. Each one is a violation of a Section 2 principle.

| Anti-pattern                  | Symptom                                                     | Fix                                                                             |
| ----------------------------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **Giant task**                | A node titled "build the backend."                          | Decompose into atomic tasks (Principle 2).                                      |
| **Implicit dependency**       | B needs A but lists no edge; "works" because of list order. | Add the explicit edge (Principle 4).                                            |
| **Circular dependency**       | Topological sort fails; nothing is `ready`.                 | Break the cycle with an intermediate artifact or task split.                    |
| **Duplicated work**           | Two nodes compute the same output.                          | Extract one node; have both consumers depend on it.                             |
| **Hidden validation**         | Correctness "checked" inside a build node.                  | Promote it to a visible validation gate node.                                   |
| **Over-connected DAG**        | Edges "just in case"; everything depends on everything.     | Remove edges that don't reflect a real data/order need — they kill parallelism. |
| **Unnecessary serialization** | Independent tasks chained A → B → C.                        | Make them siblings off a common parent.                                         |
| **Poor naming**               | `task_1`, `step_2`, `do_stuff`.                             | Name by responsibility: `validate_schema`, `render_pdf`.                        |
| **Poor task boundaries**      | A node does design _and_ implementation.                    | Cut at the artifact boundary — one node per produced artifact.                  |
| **Deep thin chain**           | A 12-node line, no width.                                   | Find hidden independence; widen and shorten the critical path.                  |

---

## 9. Best Practices Checklist

Run this before declaring a DAG ready to execute.

**Structure**

- [ ] The topological sort succeeds (no cycles).
- [ ] Every task is reachable from a root and reaches a terminal node.
- [ ] There is at least one explicit completion/terminal node.
- [ ] The critical path is as short as the real dependencies allow.

**Tasks**

- [ ] Every task has exactly one responsibility.
- [ ] No task description needs "and" to list distinct work.
- [ ] Every task is idempotent and (given inputs) deterministic.
- [ ] Names describe responsibility, not order.

**Dependencies & data**

- [ ] Every dependency id exists.
- [ ] Every dependency reflects a real prerequisite (no "just in case" edges).
- [ ] Every `inputs` artifact is produced by some ancestor's `outputs`.
- [ ] No two tasks write the same output artifact.

**Robustness**

- [ ] Validation gates sit after every phase whose output is trusted downstream.
- [ ] Risky tasks have a `retry_policy` and a sane `timeout`.
- [ ] The failure policy (block / skip / recover) is defined for risky branches.
- [ ] Human-approval checkpoints are modeled as explicit `waiting` nodes.

**Parallelism**

- [ ] Independent tasks are not accidentally serialized.
- [ ] Fan-out branches share only the edges they truly need.

---

## 10. Templates

Copy, rename ids, fill in. All use the Section 3 schema.

### Small DAG (linear with a gate)

```yaml
dag:
  id: small_linear
  objective: Produce a validated artifact in three steps.
  failure_policy: block
  tasks:
    - {
        id: step_a,
        title: Gather inputs,
        dependencies: [],
        status: pending,
        outputs: [raw],
      }
    - {
        id: step_b,
        title: Transform,
        dependencies: [step_a],
        status: pending,
        inputs: [raw],
        outputs: [result],
      }
    - {
        id: gate,
        title: Validate result,
        dependencies: [step_b],
        status: pending,
        validation: ["result passes schema check"],
      }
```

### Medium DAG (fan-out / fan-in)

```yaml
dag:
  id: medium_fanout
  objective: Run three independent analyses and merge them.
  failure_policy: block
  tasks:
    - {
        id: prep,
        title: Prepare dataset,
        dependencies: [],
        status: pending,
        outputs: [clean_data],
      }
    - {
        id: an_a,
        title: Analysis A,
        dependencies: [prep],
        status: pending,
        inputs: [clean_data],
        outputs: [a_out],
      }
    - {
        id: an_b,
        title: Analysis B,
        dependencies: [prep],
        status: pending,
        inputs: [clean_data],
        outputs: [b_out],
      }
    - {
        id: an_c,
        title: Analysis C,
        dependencies: [prep],
        status: pending,
        inputs: [clean_data],
        outputs: [c_out],
      }
    - {
        id: merge,
        title: Merge findings,
        dependencies: [an_a, an_b, an_c],
        status: pending,
        inputs: [a_out, b_out, c_out],
        outputs: [report],
      }
```

### Large DAG (multi-stage with barriers)

```yaml
dag:
  id: large_multistage
  objective: Design, build, test, and ship a component.
  failure_policy: block
  tasks:
    # Stage 1: design (parallel)
    - {
        id: spec_api,
        title: Spec API,
        dependencies: [],
        status: pending,
        outputs: [api_spec],
      }
    - {
        id: spec_ui,
        title: Spec UI,
        dependencies: [],
        status: pending,
        outputs: [ui_spec],
      }
    - {
        id: gate_design,
        title: Design review,
        dependencies: [spec_api, spec_ui],
        status: pending,
        validation: ["specs are internally consistent"],
      }
    # Stage 2: build (parallel, gated on design)
    - {
        id: build_api,
        title: Build API,
        dependencies: [gate_design],
        status: pending,
        inputs: [api_spec],
        outputs: [api],
      }
    - {
        id: build_ui,
        title: Build UI,
        dependencies: [gate_design],
        status: pending,
        inputs: [ui_spec],
        outputs: [ui],
      }
    - {
        id: gate_build,
        title: Build passes CI,
        dependencies: [build_api, build_ui],
        status: pending,
        validation: ["lint clean", "unit tests pass"],
      }
    # Stage 3: ship
    - {
        id: integration_test,
        title: Integration tests,
        dependencies: [gate_build],
        status: pending,
      }
    - {
        id: ship,
        title: Release,
        dependencies: [integration_test],
        status: pending,
      }
```

### Parallel DAG (map-reduce)

```yaml
dag:
  id: parallel_mapreduce
  objective: Process N shards in parallel and aggregate.
  failure_policy: skip
  tasks:
    - {
        id: split,
        title: Shard input,
        dependencies: [],
        status: pending,
        outputs: [shards],
      }
    - {
        id: map_1,
        title: Process shard 1,
        dependencies: [split],
        status: pending,
        inputs: [shards],
        outputs: [part_1],
      }
    - {
        id: map_2,
        title: Process shard 2,
        dependencies: [split],
        status: pending,
        inputs: [shards],
        outputs: [part_2],
      }
    - {
        id: map_3,
        title: Process shard 3,
        dependencies: [split],
        status: pending,
        inputs: [shards],
        outputs: [part_3],
      }
    - {
        id: reduce,
        title: Aggregate,
        dependencies: [map_1, map_2, map_3],
        status: pending,
        inputs: [part_1, part_2, part_3],
        outputs: [final],
      }
```

### Multi-stage / Research / Software / AI-Agent templates

These are realized as full worked examples in Section 11 — use the matching
example as your template:

- Multi-stage → _REST API implementation_ (11.1)
- Research → _AI research workflow_ (11.3)
- Software engineering → _Website development_ (11.2) and _Software deployment_ (11.5)
- AI-agent → _Multi-agent workflow_ (11.6)

---

## 11. DAG File Examples

Complete, executable DAG files. Each demonstrates a different structural pattern.
Study the _why_ notes — they explain the design decisions, not just the result.

### 11.1 REST API implementation (multi-stage + validation gates)

**Objective:** Ship a tested, documented CRUD REST API for a `Task` resource.

```
gather_requirements
        │
        ▼
   design_schema ──────────────┐
        │                      │
        ▼                      ▼
  gate_design            scaffold_project
        │                      │
   ┌────┴──────┬───────────────┤
   ▼           ▼               ▼
impl_models  impl_routes   impl_auth
   └────┬──────┴───────┬───────┘
        ▼              ▼
   write_tests    write_openapi_docs
        └──────┬───────┘
               ▼
           gate_tests
               │
               ▼
          assemble_api
```

```yaml
dag:
  id: rest_api_task_resource
  objective: Ship a tested, documented CRUD REST API for a Task resource.
  failure_policy: block
  tasks:
    - id: gather_requirements
      title: Gather requirements
      description: Collect functional requirements and acceptance criteria for the Task API.
      dependencies: []
      status: pending
      outputs: [requirements.md]

    - id: design_schema
      title: Design API schema
      description: Author the OpenAPI 3.1 spec for all CRUD endpoints.
      dependencies: [gather_requirements]
      status: pending
      inputs: [requirements.md]
      outputs: [openapi.yaml]
      validation:
        [
          "openapi.yaml is valid OpenAPI 3.1",
          "every endpoint has request+response schemas",
        ]

    - id: gate_design
      title: Design review gate
      description: Confirm the schema covers all requirements before any code is written.
      dependencies: [design_schema]
      status: pending
      validation: ["each requirement in requirements.md maps to an endpoint"]

    - id: scaffold_project
      title: Scaffold project
      description: Create the project skeleton, dependency manifest, and config.
      dependencies: [gather_requirements]
      status: pending
      outputs: [project_skeleton]

    - id: impl_models
      title: Implement data models
      description: Implement the Task model and persistence layer.
      dependencies: [gate_design, scaffold_project]
      status: pending
      inputs: [openapi.yaml, project_skeleton]
      outputs: [models]
      retry_policy: { max_attempts: 2, backoff: fixed, backoff_seconds: 3 }

    - id: impl_routes
      title: Implement routes/handlers
      description: Implement CRUD handlers per the schema.
      dependencies: [gate_design, scaffold_project]
      status: pending
      inputs: [openapi.yaml, project_skeleton]
      outputs: [routes]

    - id: impl_auth
      title: Implement auth middleware
      description: Add token auth middleware guarding mutating endpoints.
      dependencies: [gate_design, scaffold_project]
      status: pending
      inputs: [openapi.yaml]
      outputs: [auth_middleware]

    - id: write_tests
      title: Write integration tests
      description: Cover every endpoint, including auth failures and edge cases.
      dependencies: [impl_models, impl_routes, impl_auth]
      status: pending
      inputs: [models, routes, auth_middleware]
      outputs: [test_suite]

    - id: write_openapi_docs
      title: Generate API docs
      description: Render human-readable docs from the validated schema.
      dependencies: [impl_routes]
      status: pending
      inputs: [openapi.yaml]
      outputs: [api_docs]
      optional: true

    - id: gate_tests
      title: Tests pass gate
      description: All integration tests must pass before assembly.
      dependencies: [write_tests]
      status: pending
      validation: ["test suite exit code is 0", "coverage >= 80%"]
      retry_policy:
        { max_attempts: 3, backoff: exponential, backoff_seconds: 5 }

    - id: assemble_api
      title: Assemble deliverable
      description: Bundle code, tests, and docs into the release artifact.
      dependencies: [gate_tests, write_openapi_docs]
      status: pending
      inputs: [models, routes, auth_middleware, test_suite, api_docs]
      outputs: [api_release]
```

**Why this design.** `gate_design` is a barrier: writing three implementation
tasks against an unvalidated schema would risk three rewrites, so the gate pays
for itself. The three `impl_*` tasks share no edges with each other, so they run
in parallel — the critical path is `requirements → design → gate → impl →
tests → gate_tests → assemble`, not the sum of all eight build tasks.
`write_openapi_docs` is `optional`, so a docs hiccup degrades the release rather
than blocking it. `gate_tests` gets aggressive retries because flaky tests are
common and idempotent to re-run.

### 11.2 Website development (fan-out / fan-in)

**Objective:** Build and deploy a marketing site with independent page sections.

```
        setup_repo
            │
   ┌────────┼─────────┬──────────┐
   ▼        ▼         ▼          ▼
design   build_hero build_pricing build_footer
 system     │         │          │
   └────────┴────┬────┴──────────┘
                 ▼
            integrate_pages
                 ▼
            run_lighthouse (gate)
                 ▼
              deploy
```

```yaml
dag:
  id: marketing_site
  objective: Build and deploy a marketing site.
  failure_policy: block
  tasks:
    - {
        id: setup_repo,
        title: Initialize repo,
        dependencies: [],
        status: pending,
        outputs: [repo],
      }
    - {
        id: design_system,
        title: Build design system,
        dependencies: [setup_repo],
        status: pending,
        inputs: [repo],
        outputs: [tokens, components],
      }
    - {
        id: build_hero,
        title: Build hero section,
        dependencies: [design_system],
        status: pending,
        inputs: [components],
        outputs: [hero],
      }
    - {
        id: build_pricing,
        title: Build pricing section,
        dependencies: [design_system],
        status: pending,
        inputs: [components],
        outputs: [pricing],
      }
    - {
        id: build_footer,
        title: Build footer,
        dependencies: [design_system],
        status: pending,
        inputs: [components],
        outputs: [footer],
      }
    - {
        id: integrate_pages,
        title: Assemble pages,
        dependencies: [build_hero, build_pricing, build_footer],
        status: pending,
        inputs: [hero, pricing, footer],
        outputs: [site_build],
      }
    - {
        id: run_lighthouse,
        title: Performance + a11y gate,
        dependencies: [integrate_pages],
        status: pending,
        validation: ["Lighthouse perf >= 90", "no critical a11y violations"],
        retry_policy: { max_attempts: 2, backoff: fixed, backoff_seconds: 10 },
      }
    - {
        id: deploy,
        title: Deploy to CDN,
        dependencies: [run_lighthouse],
        status: pending,
        inputs: [site_build],
      }
```

**Why this design.** The three section builds all depend on `design_system` and
nothing else, so they fan out and run together. `integrate_pages` is the fan-in
merge node. `run_lighthouse` is a quality gate — deploying a slow or inaccessible
site is worse than not deploying, so the gate must pass first.

### 11.3 AI research workflow (parallel research → synthesis)

**Objective:** Produce a researched brief answering a question, with cited sources.

```
   decompose_question
   ┌───────┼────────┐
   ▼       ▼        ▼
research research research      (one branch per sub-question)
  _q1     _q2      _q3
   └───────┼────────┘
           ▼
     cross_check_sources (gate)
           ▼
       synthesize
           ▼
       fact_check (gate)
           ▼
      write_brief
```

```yaml
dag:
  id: research_brief
  objective: Produce a cited research brief answering the user's question.
  failure_policy: skip # a dead-end sub-question shouldn't kill the brief
  tasks:
    - {
        id: decompose_question,
        title: Decompose into sub-questions,
        dependencies: [],
        status: pending,
        outputs: [subquestions],
      }
    - {
        id: research_q1,
        title: Research sub-question 1,
        dependencies: [decompose_question],
        status: pending,
        inputs: [subquestions],
        outputs: [findings_1, sources_1],
        owner: researcher-agent,
      }
    - {
        id: research_q2,
        title: Research sub-question 2,
        dependencies: [decompose_question],
        status: pending,
        inputs: [subquestions],
        outputs: [findings_2, sources_2],
        owner: researcher-agent,
      }
    - {
        id: research_q3,
        title: Research sub-question 3,
        dependencies: [decompose_question],
        status: pending,
        inputs: [subquestions],
        outputs: [findings_3, sources_3],
        owner: researcher-agent,
        optional: true,
      }
    - {
        id: cross_check_sources,
        title: Source credibility gate,
        dependencies: [research_q1, research_q2, research_q3],
        status: pending,
        validation:
          [
            "every claim has >=1 credible source",
            "no contradictory sources unresolved",
          ],
      }
    - {
        id: synthesize,
        title: Synthesize findings,
        dependencies: [cross_check_sources],
        status: pending,
        inputs: [findings_1, findings_2, findings_3],
        outputs: [synthesis],
        owner: synthesis-agent,
      }
    - {
        id: fact_check,
        title: Fact-check gate,
        dependencies: [synthesize],
        status: pending,
        validation: ["each synthesized claim traces to a source"],
        retry_policy: { max_attempts: 2, backoff: none },
      }
    - {
        id: write_brief,
        title: Write final brief,
        dependencies: [fact_check],
        status: pending,
        inputs: [synthesis],
        outputs: [brief.md],
        owner: writer-agent,
      }
```

**Why this design.** Research branches are independent and `owner`-tagged to a
`researcher-agent`, so a multi-agent runtime spawns three in parallel. The
`failure_policy: skip` plus the `optional` third branch means a fruitless
sub-question degrades coverage instead of failing the whole brief. Two gates
(`cross_check_sources`, `fact_check`) guard the two places bad information would
otherwise propagate.

### 11.4 Data processing pipeline (staged pipeline)

**Objective:** Turn raw event logs into a cleaned, validated analytics table.

```
ingest → validate_schema(gate) → dedupe → enrich → aggregate → load → verify_load(gate)
```

```yaml
dag:
  id: events_pipeline
  objective: Transform raw event logs into a validated analytics table.
  failure_policy: block
  tasks:
    - {
        id: ingest,
        title: Ingest raw logs,
        dependencies: [],
        status: pending,
        outputs: [raw_events],
        retry_policy:
          { max_attempts: 3, backoff: exponential, backoff_seconds: 10 },
        timeout: 900,
      }
    - {
        id: validate_schema,
        title: Schema gate,
        dependencies: [ingest],
        status: pending,
        inputs: [raw_events],
        validation:
          ["all rows match the event schema", "required fields non-null"],
      }
    - {
        id: dedupe,
        title: Deduplicate,
        dependencies: [validate_schema],
        status: pending,
        inputs: [raw_events],
        outputs: [deduped],
      }
    - {
        id: enrich,
        title: Enrich with dimensions,
        dependencies: [dedupe],
        status: pending,
        inputs: [deduped],
        outputs: [enriched],
      }
    - {
        id: aggregate,
        title: Aggregate metrics,
        dependencies: [enrich],
        status: pending,
        inputs: [enriched],
        outputs: [agg_table],
      }
    - {
        id: load,
        title: Load into warehouse,
        dependencies: [aggregate],
        status: pending,
        inputs: [agg_table],
        outputs: [warehouse_table],
        retry_policy:
          { max_attempts: 3, backoff: exponential, backoff_seconds: 15 },
      }
    - {
        id: verify_load,
        title: Load verification gate,
        dependencies: [load],
        status: pending,
        validation: ["row count matches expected", "no nulls in key columns"],
      }
```

**Why this design.** This is mostly sequential because each stage truly consumes
the previous stage's output — forcing false parallelism here would corrupt data.
The value comes from the two gates and from idempotent, retryable I/O nodes
(`ingest`, `load`) where transient network failure is expected. `validate_schema`
sits right after ingestion so bad data is caught before any expensive transform.

### 11.5 Software deployment (approval + error recovery)

**Objective:** Deploy a release to production with a human gate and auto-rollback.

```
build → test(gate) → stage → request_approval(waiting:human) → deploy_prod
                                                                   │
                                                       ┌──(fail)───┴──(success)─┐
                                                       ▼                        ▼
                                                   rollback                smoke_test(gate)
                                                       ▼                        ▼
                                                    alert                   notify_success
```

```yaml
dag:
  id: prod_deploy
  objective: Deploy a release to production safely, with approval and rollback.
  failure_policy: recover
  tasks:
    - {
        id: build,
        title: Build artifact,
        dependencies: [],
        status: pending,
        outputs: [artifact],
      }
    - {
        id: test,
        title: Test gate,
        dependencies: [build],
        status: pending,
        inputs: [artifact],
        validation: ["unit + integration tests pass"],
      }
    - {
        id: stage,
        title: Deploy to staging,
        dependencies: [test],
        status: pending,
        inputs: [artifact],
        outputs: [staging_url],
      }
    - {
        id: request_approval,
        title: Human approval,
        dependencies: [stage],
        status: pending,
        owner: release-manager,
        tags: [human-checkpoint],
      } # parks in `waiting` until approved
    - {
        id: deploy_prod,
        title: Deploy to production,
        dependencies: [request_approval],
        status: pending,
        inputs: [artifact],
        outputs: [prod_release],
        retry_policy: { max_attempts: 1, backoff: none },
      }
    - {
        id: smoke_test,
        title: Production smoke test gate,
        dependencies: [deploy_prod],
        status: pending,
        validation: ["health endpoint 200", "key user journey succeeds"],
      }
    - {
        id: rollback,
        title: Roll back deploy,
        dependencies: [deploy_prod],
        status: pending,
        condition: "deploy_prod == failed",
        inputs: [artifact],
      } # recovery node
    - {
        id: alert,
        title: Page on-call,
        dependencies: [rollback],
        status: pending,
      }
    - {
        id: notify_success,
        title: Announce release,
        dependencies: [smoke_test],
        status: pending,
      }
```

**Why this design.** `request_approval` is an explicit `waiting` human-checkpoint
— production deploys shouldn't be fully automatic. `failure_policy: recover`
plus the `rollback` node (guarded by a condition on `deploy_prod`'s failure) gives
a defined recovery path instead of leaving prod half-deployed. `deploy_prod` has
`max_attempts: 1` — blindly retrying a production deploy can compound damage, so
failure routes to rollback rather than retry.

### 11.6 Multi-agent workflow (planner-driven, agent-routed)

**Objective:** Given a feature request, have specialized agents implement it.
This is the canonical shape for a planner that emits a DAG for an agent runtime.

```
            plan (planner-agent)
                 │
   ┌─────────────┼──────────────┐
   ▼             ▼              ▼
research      design_api    write_specs
(researcher)  (architect)   (architect)
   └─────────────┼──────────────┘
                 ▼
           code (coder-agent)
                 │
        ┌────────┴────────┐
        ▼                 ▼
   review (critic)   write_tests (tester)
        └────────┬────────┘
                 ▼
            fix_review (coder)
                 ▼
          gate_all_green
                 ▼
           document (docs-agent)
```

```yaml
dag:
  id: feature_multiagent
  objective: Implement a feature request via specialized agents.
  failure_policy: recover
  tasks:
    - {
        id: plan,
        title: Plan the feature,
        dependencies: [],
        status: pending,
        owner: planner-agent,
        model_role: planner,
        outputs: [task_plan],
        priority: 5,
      }
    - {
        id: research,
        title: Research approach,
        dependencies: [plan],
        status: pending,
        owner: researcher-agent,
        model_role: reasoning,
        inputs: [task_plan],
        outputs: [research_notes],
        tags: [reasoning],
      }
    - {
        id: design_api,
        title: Design interfaces,
        dependencies: [plan],
        status: pending,
        owner: architect-agent,
        model_role: reasoning,
        inputs: [task_plan],
        outputs: [interface_spec],
        tags: [design],
      }
    - {
        id: write_specs,
        title: Write acceptance specs,
        dependencies: [plan],
        status: pending,
        owner: architect-agent,
        model_role: reasoning,
        inputs: [task_plan],
        outputs: [acceptance_specs],
      }
    - {
        id: code,
        title: Implement feature,
        dependencies: [research, design_api, write_specs],
        status: pending,
        owner: coder-agent,
        model_role: coder,
        inputs: [research_notes, interface_spec, acceptance_specs],
        outputs: [diff],
        retry_policy: { max_attempts: 2, backoff: fixed, backoff_seconds: 5 },
        priority: 4,
      }
    - {
        id: review,
        title: Critique the diff,
        dependencies: [code],
        status: pending,
        owner: critic-agent,
        model_role: reasoning,
        inputs: [diff],
        outputs: [review_comments],
      }
    - {
        id: write_tests,
        title: Write tests,
        dependencies: [code],
        status: pending,
        owner: tester-agent,
        model_role: coder,
        inputs: [diff, acceptance_specs],
        outputs: [tests],
      }
    - {
        id: fix_review,
        title: Apply review feedback,
        dependencies: [review, write_tests],
        status: pending,
        owner: coder-agent,
        model_role: coder,
        inputs: [diff, review_comments],
        outputs: [final_diff],
      }
    - {
        id: gate_all_green,
        title: All checks green gate,
        dependencies: [fix_review],
        status: pending,
        model_role: general,
        validation:
          ["tests pass on final_diff", "no unresolved review comments"],
        retry_policy:
          { max_attempts: 3, backoff: exponential, backoff_seconds: 5 },
      }
    - {
        id: document,
        title: Update docs,
        dependencies: [gate_all_green],
        status: pending,
        owner: docs-agent,
        model_role: general,
        inputs: [final_diff],
        outputs: [doc_updates],
        optional: true,
      }
```

**Why this design.** Each task carries a `model_role` naming the agent role that
runs it: `code`/`write_tests`/`fix_review` use `coder`, the analysis-heavy
`research`/`design_api`/`review` use `reasoning`, and the cheap
`gate_all_green`/`document` steps use `general` — so a small model handles
high-volume low-stakes work while the expensive model is reserved for where it
matters. The runtime resolves each role to the concrete model the user configured
for it, at dispatch. The planner emits everything as `pending`; the scheduler spawns
`research`, `design_api`, and `write_specs` in parallel because they share only
the `plan` dependency. `review` and `write_tests` also parallelize off `code`.
`priority` nudges the scheduler to favor the critical-path nodes (`plan`, `code`)
when agent slots are scarce. The `gate_all_green` node is the single place "done"
is asserted, and `document` is optional so a docs failure won't block the feature.

### 11.7 Machine learning training pipeline (conditional branch)

**Objective:** Train, evaluate, and conditionally promote a model.

```
prep_data → split → train → evaluate → route_on_metric
                                            │
                              ┌────(good)───┴───(poor)────┐
                              ▼                            ▼
                         register_model              tune_and_retrain
                              ▼                            │
                          deploy_model            (loops back via new DAG run)
```

```yaml
dag:
  id: ml_training
  objective: Train, evaluate, and conditionally promote a model.
  failure_policy: block
  tasks:
    - {
        id: prep_data,
        title: Clean + feature-engineer,
        dependencies: [],
        status: pending,
        outputs: [features],
      }
    - {
        id: split,
        title: Train/val/test split,
        dependencies: [prep_data],
        status: pending,
        inputs: [features],
        outputs: [train_set, val_set, test_set],
      }
    - {
        id: train,
        title: Train model,
        dependencies: [split],
        status: pending,
        inputs: [train_set, val_set],
        outputs: [model],
        timeout: 7200,
        retry_policy: { max_attempts: 2, backoff: fixed, backoff_seconds: 30 },
      }
    - {
        id: evaluate,
        title: Evaluate on test set,
        dependencies: [train],
        status: pending,
        inputs: [model, test_set],
        outputs: [metrics],
      }
    - {
        id: route_on_metric,
        title: Quality decision,
        dependencies: [evaluate],
        status: pending,
        inputs: [metrics],
        outputs: [decision],
      } # "good" or "poor"
    - {
        id: register_model,
        title: Register model,
        dependencies: [route_on_metric],
        status: pending,
        condition: "decision == good",
        inputs: [model],
        outputs: [registered_model],
      }
    - {
        id: deploy_model,
        title: Deploy model,
        dependencies: [register_model],
        status: pending,
        inputs: [registered_model],
      }
    - {
        id: tune_and_retrain,
        title: Tune hyperparameters,
        dependencies: [route_on_metric],
        status: pending,
        condition: "decision == poor",
        inputs: [metrics],
        outputs: [new_config],
      }
```

**Why this design.** `route_on_metric` produces a decision that guards two
mutually exclusive branches via `condition`. Exactly one branch runs; the other
is `skipped`. Note the DAG stays acyclic — "retry training" is modeled as
`tune_and_retrain` producing a `new_config` that seeds a _fresh DAG run_, never an
edge back into `train` (which would create a cycle). `train` gets a long
`timeout` and limited retries because it's expensive.

### 11.8 Book writing (tree expansion)

**Objective:** Draft a non-fiction book from a one-line premise.

```
premise → outline → [ch_1_draft, ch_2_draft, ch_3_draft] (parallel)
   each ch_N_draft → ch_N_revise
   all revisions → consistency_pass(gate) → assemble_manuscript
```

```yaml
dag:
  id: book_draft
  objective: Draft a non-fiction book from a premise.
  failure_policy: block
  tasks:
    - {
        id: premise,
        title: Define premise + audience,
        dependencies: [],
        status: pending,
        outputs: [premise_doc],
      }
    - {
        id: outline,
        title: Build chapter outline,
        dependencies: [premise],
        status: pending,
        inputs: [premise_doc],
        outputs: [outline],
      }
    - {
        id: ch_1_draft,
        title: Draft chapter 1,
        dependencies: [outline],
        status: pending,
        inputs: [outline],
        outputs: [ch1],
      }
    - {
        id: ch_2_draft,
        title: Draft chapter 2,
        dependencies: [outline],
        status: pending,
        inputs: [outline],
        outputs: [ch2],
      }
    - {
        id: ch_3_draft,
        title: Draft chapter 3,
        dependencies: [outline],
        status: pending,
        inputs: [outline],
        outputs: [ch3],
      }
    - {
        id: ch_1_revise,
        title: Revise chapter 1,
        dependencies: [ch_1_draft],
        status: pending,
        inputs: [ch1],
        outputs: [ch1_final],
      }
    - {
        id: ch_2_revise,
        title: Revise chapter 2,
        dependencies: [ch_2_draft],
        status: pending,
        inputs: [ch2],
        outputs: [ch2_final],
      }
    - {
        id: ch_3_revise,
        title: Revise chapter 3,
        dependencies: [ch_3_draft],
        status: pending,
        inputs: [ch3],
        outputs: [ch3_final],
      }
    - {
        id: consistency_pass,
        title: Cross-chapter consistency gate,
        dependencies: [ch_1_revise, ch_2_revise, ch_3_revise],
        status: pending,
        validation:
          ["terminology consistent across chapters", "no contradictions"],
      }
    - {
        id: assemble_manuscript,
        title: Assemble manuscript,
        dependencies: [consistency_pass],
        status: pending,
        outputs: [manuscript],
      }
```

**Why this design.** The outline fans out into per-chapter draft→revise chains
that are fully independent, so all chapters are written in parallel. They fan in
at `consistency_pass`, the one place cross-chapter coherence can be checked. This
tree-expansion shape generalizes to any "outline → sections → subsections" work.

> **Other domains** (documentation generation, code refactoring, video
> production, mobile app development) are compositions of the patterns above:
> documentation = fan-out/fan-in (11.2); refactoring = pipeline with a test gate
> after each step (11.4); video production = multi-stage with an approval gate
> (11.5); mobile app = the multi-stage build of 11.1 plus platform fan-out
> (iOS / Android branches off a shared core).

---

## 12. Progressive Examples

The same project — _"add a contact form to our site"_ — refined in three passes.
Watch what each revision fixes.

### Pass 1 — Simple (naive linear list)

```yaml
dag:
  id: contact_form_v1
  objective: Add a contact form.
  tasks:
    - {
        id: do_it,
        title: Build the contact form feature,
        dependencies: [],
        status: pending,
      }
```

**Problems.** One giant task (anti-pattern: giant task). No parallelism, no
validation, no recovery. If it fails, you know nothing about _where_. This is a
to-do item, not a plan.

### Pass 2 — Better (decomposed and ordered)

```yaml
dag:
  id: contact_form_v2
  objective: Add a contact form.
  tasks:
    - {
        id: design_form,
        title: Design form fields,
        dependencies: [],
        status: pending,
      }
    - {
        id: build_frontend,
        title: Build form UI,
        dependencies: [design_form],
        status: pending,
      }
    - {
        id: build_backend,
        title: Build submit endpoint,
        dependencies: [build_frontend],
        status: pending,
      }
    - {
        id: test,
        title: Test it,
        dependencies: [build_backend],
        status: pending,
      }
```

**Improvements.** Atomic tasks with names that describe responsibility.
**Remaining problems.** `build_backend` depends on `build_frontend`, but the
endpoint doesn't actually need the UI — that's _unnecessary serialization_. There
are no validation gates, no retry policy, and "test" isn't a gate that blocks
anything. Data flow is implicit.

### Pass 3 — Production-quality

```yaml
dag:
  id: contact_form_v3
  objective: Add a working, spam-protected contact form with tests.
  failure_policy: block
  tasks:
    - {
        id: design_form,
        title: Design form + validation rules,
        dependencies: [],
        status: pending,
        outputs: [form_spec],
      }
    - {
        id: build_frontend,
        title: Build form UI,
        dependencies: [design_form],
        status: pending,
        inputs: [form_spec],
        outputs: [form_ui],
      }
    - {
        id: build_backend,
        title: Build submit endpoint,
        dependencies: [design_form],
        status: pending,
        inputs: [form_spec],
        outputs: [submit_endpoint],
        retry_policy: { max_attempts: 2, backoff: fixed, backoff_seconds: 3 },
      }
    - {
        id: add_spam_protection,
        title: Add rate-limit + captcha,
        dependencies: [build_backend],
        status: pending,
        inputs: [submit_endpoint],
        outputs: [protected_endpoint],
      }
    - {
        id: integration_test,
        title: End-to-end test gate,
        dependencies: [build_frontend, add_spam_protection],
        status: pending,
        inputs: [form_ui, protected_endpoint],
        validation:
          [
            "valid submission succeeds",
            "spam submission is blocked",
            "invalid input shows errors",
          ],
        retry_policy:
          { max_attempts: 3, backoff: exponential, backoff_seconds: 5 },
      }
    - {
        id: ship,
        title: Deploy behind flag,
        dependencies: [integration_test],
        status: pending,
      }
```

**Improvements over Pass 2.**

- `build_frontend` and `build_backend` now both depend only on `design_form`, so
  they run in parallel — the false serialization is gone.
- `inputs`/`outputs` make data flow explicit and validatable.
- A real `integration_test` **gate** with concrete checks blocks shipping;
  "spam is blocked" is now an asserted property, not a hope.
- Retry policies cover the flaky network-touching nodes.
- `add_spam_protection` was surfaced as its own concern instead of hiding inside
  the backend task.

The lesson: decomposition (Pass 2) is necessary but not sufficient. Production
quality comes from _removing false edges_, _making data flow explicit_, and
_adding gates and recovery_.

---

## 13. Execution Rules

How an executor (or an AI acting as one) should run a DAG. The planner emits the
graph; these rules drive it to completion.

### The execution loop

```
1.  Initialize: set every task to `pending`. Validate the DAG (Section 5.4); abort if invalid.
2.  Resolve states: any task whose dependencies are all `completed` → `ready`;
    any task with an unmet dependency → `blocked`.
3.  Select: from `ready` tasks, pick the highest-`priority` ones that fit available
    resources/slots. Ties broken by priority, then by shortest remaining critical path.
4.  Run: move selected tasks to `running` (via `waiting` if they must queue for a slot).
5.  On finish:
      success → `validating`; run checks → pass = `completed`, fail = retry-or-fail.
      error/timeout → retry-or-fail.
6.  Propagate: when a task `completed`, re-resolve its dependents (some become `ready`).
                when a task `failed`, apply the failure policy to its descendants.
7.  Repeat from step 3 until no task is `ready`, `running`, `waiting`, or `retrying`.
8.  Detect completion (below).
```

### Selecting ready tasks

A task is selectable only when its status is `ready`. Never start a `blocked`,
`waiting`-on-human, or terminal task. Among `ready` tasks, prefer those on the
critical path (longest downstream chain) so the overall finish time shrinks;
`priority` is the explicit override.

### Respecting dependencies

Never start a task until **every** id in its `dependencies` is `completed`
(not merely `running` or `validating`). This invariant is what makes the graph
correct; violating it for speed corrupts results.

### Updating status

The executor is the single writer of task status. Every transition must be one
allowed by the Section 4 table. Persist each transition (with timestamp and
attempt count) so the run is observable and resumable.

### Handling failures

On `failed`, apply the DAG's `failure_policy`:

- **block** → descendants stay `blocked`; surface for human/recovery.
- **skip** → descendants become `skipped`; re-resolve _their_ descendants.
- **recover** → mark the designated recovery node `ready`.

### Retries

On a failed attempt with `attempts < max_attempts`: status → `retrying`, wait per
`backoff`, then → `running`. Idempotency (Principle 9) guarantees a retry can't
double-apply side effects.

### Validation

A task is not `completed` until it is `validating` and **all** its `validation`
checks pass. A failed check is treated like an execution failure for retry
purposes. Gate nodes are just tasks whose entire job is validation.

### Completion detection

The DAG is **complete** when every task is in a terminal state (`completed`,
`failed`, `skipped`, or `cancelled`) and no task is `ready`/`running`/`waiting`/
`retrying`. The run **succeeded** if all _required_ (non-optional) terminal tasks
are `completed`; it **failed** if any required task is `failed`.

### Cancellation

To cancel, move all non-terminal tasks to `cancelled` and signal any `running`
tasks to stop (via the executor's cancellation mechanism). Already-`completed`
tasks keep their outputs.

### Resuming a partially completed DAG

On resume, load persisted state. Keep all `completed` tasks as-is (their outputs
are valid because tasks are deterministic). Reset any task left mid-flight
(`running`/`validating`/`retrying` at the time of interruption) back to `ready`
if its dependencies are still satisfied, then re-enter the loop at step 2. Because
tasks are idempotent, re-running a reset task is safe. Only failed/affected and
never-run nodes do real work — completed work is never repeated.

---

## 14. Quality Standards

Hold every emitted DAG to these bars.

### Naming

- `id`s are `snake_case`, unique, stable, and describe the task's responsibility
  (`validate_schema`, not `task_3`).
- `title`s are short imperative human labels.
- Output artifact names are nouns reused verbatim as downstream `inputs`.

### Routing

- Every task that an agent runs sets a `model_role` naming the agent role that
  runs it (one of the fixed roles available for the chat).
- `model_role` is always a role name (`planner`, `coder`, `reasoning`, …), never a
  concrete model name or tag — the runtime resolves it at dispatch.
- Reserve expensive roles for genuinely hard tasks; push high-volume, low-stakes
  steps (routing, short validations) to a cheap role like `general`.

### Readability

- A reader can trace the flow and find the critical path within seconds.
- Phases/stages are visually grouped (comments or `tags`).
- No node has more inbound or outbound edges than it genuinely needs.

### Maintainability

- One responsibility per task, so edits are local.
- Editing one task never silently changes another's behavior (no hidden coupling
  beyond declared `inputs`/`outputs`).

### Modularity

- Reusable clusters (CI checks, research-and-verify) are factored as subgraphs
  with a single entry dependency and a single exit output.

### Observability

- Every status transition is persisted with timestamp and attempt count.
- Validation checks are explicit and named, so a failure says _what_ failed.
- The DAG can be rendered/inspected at any time to show live status per node.

### Scalability

- The graph is as shallow and wide as real dependencies permit (short critical
  path), so adding parallel resources actually reduces wall-clock time.
- Map-reduce shapes are used for data-parallel work rather than long chains.

### Testability

- Each task's `objective` is a single testable success condition.
- `validation` checks are objective and machine-checkable wherever possible.
- Determinism + idempotency mean a task can be run in isolation and verified.

---

## Quick-reference summary

When given any objective:

1. Name the deliverable.
2. Sketch phases.
3. Decompose to atomic tasks.
4. Draw only the real dependency edges.
5. Find what can run in parallel.
6. Insert validation gates between trusted phases.
7. Add an explicit completion node.
8. Tag each agent task with a `model_role` (the agent role that runs it) — a role
   name from the available set, never a model name.
9. Topologically sort to prove it's acyclic; shorten the critical path.
10. Run the Section 9 checklist, then emit every task as `pending`.
11. Save the whole plan with `create_dag` (it validates and stores it); read it
    back with `get_dag`.

A good DAG is **atomic** (one job per node), **explicit** (every edge real and
declared), **parallel** (no false serialization), **guarded** (gates where trust
is transferred), **recoverable** (retries, failure policy, idempotent nodes), and
**routed** (a valid `model_role` on every agent task).
