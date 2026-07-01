---
title: CLI
description: The bash runner and the kubelings binary subcommands.
---

## `run-challenge-local.sh`

The engine. It extracts a lesson's `init`/`verify` task blocks and runs them on
your local `kind` cluster.

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

## `kubelings` binary

```sh
kubelings              # launch the TUI
kubelings doctor       # env, cluster status, and lessons (no TUI)
kubelings shell <l>    # task + cluster-wired shell for lesson <l>
```

`kubelings doctor` is handy in CI or over SSH — it prints prerequisites, the kind
cluster status, and the lessons with their progress markers, without a TTY.

## `just`

```sh
just tui      # build + launch the TUI
just doctor   # headless status
just up       # cluster up
just down     # cluster down
just run <lesson> <verb>
just test     # go tests
```
