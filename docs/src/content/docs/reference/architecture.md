---
title: Architecture
description: How the course, the runner, and the TUI fit together — and how to author a lesson.
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

## Authoring a lesson

Browse the lessons themselves in the **[Catalog](/catalog/)**; this is how one is
built. Every lesson is one of four types, badged in both the TUI and the catalog:

| Type | Meaning |
|---|---|
| **lab** | hands-on concept lesson, verify-gated |
| **incident** | a real, cited production incident, reproduced (hands-on) |
| **drill** | a synthetic failure pattern seen across many companies (hands-on) |
| **read** | guided reading (runbooks) — no tasks, read-only |

### Running a lesson

```sh
scripts/run-challenge-local.sh <lesson> init     # build it
scripts/run-challenge-local.sh <lesson> verify   # check your fix
scripts/run-challenge-local.sh <lesson> reset    # wipe + re-init
scripts/run-challenge-local.sh <lesson> solution # print the lesson (incl. solution)
```

`<lesson>` accepts a lesson name (e.g. `rolling-update`), its slug, or a dir path
(confined to the course tree).

### Adding a lesson

```sh
tools/scaffold-lesson.sh module-2 14 my-lesson "Fix the broken thing" k8s-omni
# edit courses/kubelings/module-2/14.my-lesson/{index.md,unit-1.md}

# test locally
scripts/run-challenge-local.sh my-lesson init && scripts/run-challenge-local.sh my-lesson verify

# publish the whole course
labctl content push course kubelings-dbd840c8 --dir courses/kubelings --force
```

The catalog is derived from the course, so regenerate it after adding or renaming
a lesson — it re-extracts titles, descriptions and each lesson's problem prose:

```sh
python3 docs/scripts/gen-catalog.py
```

### Task authoring rules

- `init` tasks build the broken/baseline state; `verify` exits `0` when solved,
  non-zero otherwise.
- Keep checks plain `kubectl` + bash.
- Lesson scripts run **inside the kind node**, not on your host — see
  [Security](/reference/security/). Avoid `hostPath`/privileged pods; the lesson
  namespace enforces Pod Security `baseline`.
- Incident replays carry a verified `source:` URL, surfaced on the lesson's
  [Catalog](/catalog/) row.
