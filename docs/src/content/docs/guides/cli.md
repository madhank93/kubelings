---
title: CLI
description: The kubelings binary, the bash engine underneath it, and the just shortcuts.
---

Two layers, one engine:

- **`kubelings`** — the TUI/CLI binary. This is the interface you normally use.
- **`scripts/run-challenge-local.sh`** — the bash engine the binary delegates
  every cluster/lesson action to. Still very much in use: it's the single
  place that extracts a lesson's `init`/`verify` task blocks and runs them —
  **confined inside the kind node**, never on your host (see
  [Security](/reference/security/)). Call it directly for scripting/CI; the
  TUI calls the exact same code paths.

## `kubelings` binary

```sh
kubelings              # launch the TUI
kubelings doctor       # env, cluster status, and lessons (no TTY needed)
kubelings shell <l>    # task + cluster-wired shell for lesson <l>
```

In the TUI: `↵` plays a lesson (cluster up → init → shell), `v` verifies,
`r` resets, `↵` on a reading marks it read. `kubelings doctor` is handy in CI
or over SSH — prerequisites, kind cluster status, and per-lesson progress
markers, no TTY required.

## The engine: `run-challenge-local.sh`

Same verbs the TUI uses, callable directly:

```sh
scripts/run-challenge-local.sh up                 # create the 3-node kind cluster
scripts/run-challenge-local.sh list               # list runnable lessons
scripts/run-challenge-local.sh <lesson> init      # build the scenario
scripts/run-challenge-local.sh <lesson> verify    # run the check
scripts/run-challenge-local.sh <lesson> reset     # wipe ns + re-init
scripts/run-challenge-local.sh <lesson> solution  # print the lesson content
scripts/run-challenge-local.sh <lesson> progress  # print .labctl/progress.tsv
scripts/run-challenge-local.sh down               # delete the cluster
```

Override the cluster with `KIND_WORKERS=N` and `KUBELINGS_CLUSTER=name`.

When to reach for it instead of the TUI:

- CI pipelines (init + verify as a lesson smoke test)
- automation/scripting around lessons
- debugging a lesson you're authoring (its output is the raw task output)

## `just`

```sh
just tui      # build + launch the TUI
just doctor   # headless status
just up       # cluster up
just down     # cluster down
just run <lesson> <verb>
just test     # go tests
```
