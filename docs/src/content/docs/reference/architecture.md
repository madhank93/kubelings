---
title: Architecture
description: How the course, the runner, and the TUI fit together.
---

Kubelings has three layers with a strict separation of concerns.

## 1. The course — source of truth

`courses/kubelings/` is the only authored artifact and the only thing published
on iximiuz Labs.

```text
courses/kubelings/
  index.md                        # course
  module-N/0.index.md             # module
  module-N/<n>.<lesson>/index.md  # lesson: playground + init/verify tasks
  module-N/<n>.<lesson>/unit-1.md # unit: prose + ::simple-task + solution
```

A lesson's `index.md` holds the `playground` and the `tasks` (init/verify) that
define the scenario and its check.

## 2. The runner — the engine

`scripts/run-challenge-local.sh` is the single executor. It:

- resolves a lesson (confined to the course tree),
- extracts the `init`/`verify` `run:` blocks with `yq`,
- executes them **inside the kind control-plane node** (never on the host),
- tracks progress in `.labctl/progress.tsv`.

Because the platform and the local runner execute the *same* task blocks, "works
locally" and "works on iximiuz Labs" never drift.

## 3. The TUI — UI only

`cmd/kubelings` is a bubbletea app. It **does not** reimplement lesson logic — it
shells out to the runner for every action. Its Go packages:

- `internal/course` — read-only discovery of modules/lessons + hint/solution/task
  extraction.
- `internal/progress` — read/write the shared progress markers.
- `internal/runner` — thin wrapper over the bash runner + cluster status.
- `internal/preflight` — dependency / Docker checks.
- `internal/ui` — the bubbletea model, splash, and the cluster-wired shell.

## Local cluster

The local cluster is a **3-node `kind`** cluster (`kind-kubelings`) that mirrors
the iximiuz **k8s-omni** playground. It is created on demand, shared across
lessons, and destroyed only on `down`. Each scenario lives in the `kubelings`
namespace.
