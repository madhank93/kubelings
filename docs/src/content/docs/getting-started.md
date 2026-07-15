---
title: Getting Started
description: Install the prerequisites and run your first Kubelings lesson locally.
---

There are two ways to run Kubelings. Both drive the **same** lessons from one
source of truth — the checks are identical.

## Option 1 · No setup (fastest) ☁

Run instantly in your browser — zero local install, real clusters, nothing to
clean up:

**→ [Open Kubelings on iximiuz Labs](https://labs.iximiuz.com/courses/kubelings-dbd840c8)**

Pick a lesson, get a live playground, fix the cluster, watch the check turn green.
This is the recommended way to start.

## Option 2 · Run locally 💻

For offline / air-gapped use, or if you prefer your own terminal. Runs on `kind`.

### Prerequisites

A Docker runtime plus a few CLIs. On macOS these come from the dotfiles Brewfile:

- **Docker runtime** — OrbStack or Docker Desktop (must be running)
- **kind**, **kubectl**, **yq**
- **Go ≥ 1.25** (only for the TUI)

### Reproducible toolchain

The repo pins tool versions with `mise.toml` (and `go.sum` locks the Go deps):

```sh
mise install    # fetch the pinned go / kubectl / kind / yq
mise run setup  # install + go build + go test
```

### Dev container / Codespaces

Prefer zero local toolchain setup? The repo ships a
[`.devcontainer/`](https://github.com/madhank93/kubelings/tree/main/.devcontainer)
with Docker-in-Docker (for `kind`), the pinned `mise` toolchain, and `just` — the
full lesson runtime. Open it either way:

- **VS Code** — install the *Dev Containers* extension, then **Reopen in
  Container**.
- **GitHub Codespaces** — **Code → Codespaces → Create codespace** on the repo.

First launch provisions the toolchain automatically; then `just tui`, `just up`,
and the lessons all work inside the container — clusters and all. (The container
targets the lesson runtime; the docs site needs Node, so build it on your host.)

### Clone

```sh
git clone https://github.com/madhank93/kubelings
cd kubelings
```

### Run the TUI

```sh
just tui        # build + launch  (or: go run ./cmd/kubelings)
```

Press **`↵` play** on a lesson: it spins up a local `kind` cluster (if needed),
builds the scenario, and drops you into a shell wired to the cluster — with the
task and helper commands (`task`, `hint`, `verify`, `solution`, `k`=kubectl).

Full walkthrough: [The TUI](/guides/tui/).

### Or use the CLI

No TUI needed:

```sh
scripts/run-challenge-local.sh up                 # 3-node kind cluster
scripts/run-challenge-local.sh rolling-update init
scripts/run-challenge-local.sh rolling-update verify
scripts/run-challenge-local.sh down               # destroy the cluster
```

See [CLI](/guides/cli/).

### Cluster lifecycle

The kind cluster is **created on demand** (`u` / play) and **persists** across
lessons and TUI restarts — quitting does **not** destroy it. It is removed only
when you run `down` (or press `d` in the TUI). This keeps switching between
scenarios fast.
