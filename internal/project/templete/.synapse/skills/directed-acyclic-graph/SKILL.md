---
name: dag-workflow-planner
description: >
  Decompose a complex objective into a validated, executable DAG (Directed
  Acyclic Graph) of atomic tasks with explicit dependencies, validation gates,
  and maximized parallelism. Use whenever the user asks to plan, break down,
  decompose, orchestrate, sequence, or "build X end to end" — even if they don't
  say "DAG". Reach for it before writing an ad-hoc linear checklist for anything
  with real dependency structure.
---

# DAG Workflow Planner

How to turn an objective into a correct, parallel, fault-tolerant DAG of atomic
tasks and save it with the `create_dag` tool.

## ⚠️ Output contract — read first

- **You do NOT output a DAG as text, YAML, JSON, or a code block.** You build the
  plan and **save it by calling the `create_dag` tool** with structured fields.
- Pass: `id`, `objective`, `failure_policy`, and `tasks` (an array of task
  objects). Each task object: `id`, `title`, and optionally `description`,
  `dependencies`, `inputs`, `outputs`, `model_role`.
- Put each field in its **own** field — never stuff a description into `title`.
- **Plan the user's actual objective**, in their own terms. Never substitute a
  generic or example objective.
- `model_role` must be one of the roles listed as available for the chat.
- After `create_dag` succeeds, reply with a one-line confirmation only.

---

## 1. What a DAG is, and why

A **DAG** is tasks (nodes) connected by dependencies (directed edges) with **no
cycles**. Edge `A → B` means "B can't start until A completes." No cycles means
execution can always make progress and is guaranteed to terminate.

A linear list (`1 → 2 → 3`) forces a total order even when the real work only has
a partial order, so it over-serializes. A DAG encodes only real dependencies,
which unlocks **parallelism** (independent work runs at once), **fault isolation**
(a failure stalls only its descendants), and **resumability** (re-run only failed
or affected nodes).

---

## 2. Core principles

1. **One responsibility per task.** If a description needs "and" to cover distinct
   work, split it.
2. **Smallest meaningful atomic unit** — independently runnable, validatable,
   retryable. Not "build the backend" (too big), not "add one log line" (too small).
3. **Never combine unrelated work** — it destroys parallelism and hides which half
   failed.
4. **Every dependency is explicit.** If B needs A's output, write the edge. Never
   rely on ordering or side effects.
5. **No cycles.** A cycle means nothing can start. Break it with an intermediate
   artifact or a task split.
6. **Maximize parallelism.** For each pair of tasks, add an edge only if a real
   dependency exists; otherwise leave them independent.
7. **Insert validation gates** after any phase whose output downstream trusts.
8. **Deterministic + idempotent tasks** — same inputs → same outputs; running
   twice is safe. This is what makes retries and resumes safe.
9. **Readable graph** — names and structure let a human trace the flow in seconds.
10. **Prefer shallow-and-wide over deep-and-thin** — short critical path, more
    parallelism.
11. **Design for recovery.** Assume any node can fail; give risky nodes retries
    and a recovery path.

---

## 3. Task fields

| Field          | Required    | Meaning                                                                                                  |
| -------------- | ----------- | -------------------------------------------------------------------------------------------------------- |
| `id`           | yes         | Unique, stable `snake_case` identifier (`validate_schema`, not `task_3`). Edges reference it.            |
| `title`        | yes         | Short human label.                                                                                       |
| `description`  | recommended | What the task does, specific enough that the executor needs no guesswork.                                |
| `dependencies` | recommended | Ids that must reach `completed` before this task is `ready`. Empty = root task.                          |
| `inputs`       | recommended | Named artifacts consumed. Makes data-flow explicit.                                                      |
| `outputs`      | recommended | Named artifacts produced. Downstream tasks list these as `inputs`.                                       |
| `model_role`   | recommended | The agent role that runs the task (see below). The routing key.                                          |
| `status`       | auto        | Starts `pending`; the runtime drives it. You don't set anything else.                                    |
| `objective`    | optional    | The single testable success condition ("definition of done").                                           |
| `validation`   | optional    | Concrete checks that must pass before `completed`.                                                       |
| `retry_policy` | optional    | `max_attempts`, `backoff` (`none`/`fixed`/`exponential`), `backoff_seconds`.                             |
| `timeout`      | optional    | Hard limit in seconds.                                                                                   |
| `priority`     | optional    | Tie-breaker among `ready` tasks.                                                                         |

`dependencies` is **control flow** (ordering); `inputs`/`outputs` are **data
flow**. Keep both so a validator can catch "task consumes an artifact nobody
produces."

### `model_role`

`model_role` names the **agent role** that runs the task — the routing key the
runtime resolves to whatever model the user configured for that role. Never put a
concrete model name. Use **only** roles available for the chat. The vocabulary:
`planner`, `reasoning`, `coder`, `tester`, `qa`, `analyst`, `reviewer`,
`researcher`, `vision`, `architect`, `designer`, `devops`, `security`, `writer`,
`editor`, `data`, `summarizer`, `general`.

---

## 4. Task status lifecycle

The planner emits every task as `pending`; the executor owns all transitions.

`pending` → `blocked` (a dependency isn't done) or `ready` (all deps done) →
`running` → `validating` → `completed`. On error: `retrying` (attempts remain) or
`failed` (exhausted). Other terminal states: `skipped` (branch not taken / optional
dep failed), `cancelled` (re-planned away). `waiting` parks a task on a slot,
resource, or human input.

A task becomes `ready` the instant **all** its dependencies are `completed`.

---

## 5. Dependencies and validation

**Primitives:** fan-in (one task depends on many — a merge/sync barrier); fan-out
(many tasks share one dependency — they unlock together); merge node (fans in
several branches); validation gate (a node whose job is to assert a property,
blocking the phase until it passes). Conditional branches run one of several paths
based on a decision output; the untaken branch is `skipped`.

**Validation (the planner emits only DAGs that pass these):**

1. **Existence** — every dependency id refers to a real task.
2. **Acyclicity** — a topological sort (Kahn's algorithm) succeeds.
3. **Reachability** — no orphans; every task reaches a terminal node.
4. **Data-flow** — every `inputs` artifact is produced by some ancestor's `outputs`.
5. **Single-writer** — no two tasks produce the same output artifact.

`create_dag` runs these for you. Model "retry/iterate" as a node that seeds a
**fresh DAG run**, never an edge pointing backward (that's a cycle).

---

## 6. Planning method

1. Restate the objective in one sentence; name the final deliverable(s).
2. Sketch 3–7 coarse phases (thinking aid, not nodes).
3. Decompose each phase into atomic tasks (one responsibility each).
4. Add an edge for each real prerequisite — and only real ones.
5. Find independent tasks and leave them unconnected so they run in parallel.
6. Insert a validation gate after any phase whose output is trusted downstream.
7. Add an explicit completion node that fans in the final deliverables.
8. Give each agent task a `model_role` from the available roles.
9. Mentally topo-sort to confirm no cycles; shorten the critical path.
10. Call `create_dag` with the whole plan. Fix any validation error and call again.

---

## 7. Design patterns

- **Sequential** — a straight chain; use only when each step truly needs the
  previous one's output. Minimize (longest critical path, no parallelism).
- **Fan-out / fan-in** — one root unlocks parallel tasks that later merge into one
  node. The backbone of most DAGs.
- **Map-reduce** — split into N shards, process in parallel, aggregate.
- **Pipeline** — stages transform a stream; logically sequential per item.
- **Multi-stage with gates** — phases separated by validation barriers, each phase
  internally parallel.
- **Approval workflow** — a `waiting` human-checkpoint node between automated phases.
- **Error-recovery** — pair a risky node with a recovery node reachable on failure
  (`failure_policy: recover`).

**Failure policy** for the graph: `block` (default — descendants stay blocked),
`skip` (optional branches), or `recover` (a recovery node becomes ready on failure).

---

## 8. Anti-patterns

| Anti-pattern             | Symptom                                          | Fix                                          |
| ------------------------ | ------------------------------------------------ | -------------------------------------------- |
| Giant task               | A node titled "build the backend."               | Decompose into atomic tasks.                 |
| Implicit dependency      | B needs A but lists no edge; "works" by ordering.| Add the explicit edge.                       |
| Circular dependency      | Topological sort fails; nothing is `ready`.      | Break the cycle with an intermediate artifact.|
| Duplicated work          | Two nodes compute the same output.               | Extract one; both consumers depend on it.    |
| Hidden validation        | Correctness "checked" inside a build node.       | Promote it to a visible gate node.           |
| Over-connected DAG       | Edges "just in case."                            | Remove edges with no real data/order need.   |
| Unnecessary serialization| Independent tasks chained A → B → C.             | Make them siblings off a common parent.      |
| Poor naming              | `task_1`, `do_stuff`.                            | Name by responsibility: `validate_schema`.   |
| Deep thin chain          | A long line, no width.                           | Find hidden independence; widen and shorten. |

---

## 9. Quality checklist

Before calling `create_dag`:

- Topological sort succeeds (no cycles); every task reaches a terminal node; there
  is at least one explicit completion node.
- Every task has one responsibility; ids are descriptive `snake_case`.
- Every dependency id exists and reflects a real prerequisite (no "just in case").
- Every `inputs` artifact is produced by some ancestor's `outputs`; no two tasks
  write the same output.
- Validation gates sit after every trusted phase; risky tasks have a `retry_policy`
  and the graph has a `failure_policy`.
- Independent tasks are not accidentally serialized.
- Every agent task has a valid `model_role` from the available roles.
- The objective and tasks address the **user's actual goal** — not an example.
