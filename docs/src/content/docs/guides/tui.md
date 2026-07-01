---
title: The TUI
description: The interactive Kubelings terminal UI — keys, panes, and the play flow.
---

`kubelings` (in `cmd/kubelings`) is a [bubbletea](https://github.com/charmbracelet/bubbletea)
terminal UI. It is **UI-only** — every action delegates to the bash runner, so the
TUI and CLI stay in lockstep.

```sh
just tui          # build + launch
# or
go run ./cmd/kubelings
```

## Layout

- **Welcome splash** on launch (project, author, how-it-works, cluster lifecycle).
  Any key enters; `a` reopens it.
- **Left** — lessons grouped by module with progress markers `◌ / ◐ / ✓`.
- **Right** — the selected lesson: description, status, and a **Cluster** block
  (local kind context, node count, k8s version, namespace `kubelings`, and the
  iximiuz playground it mirrors).
- **Footer** — the key bar and a spinner while a task runs.

## Keys

| Key | Action |
|-----|--------|
| `↑/↓` `j/k` | navigate lessons |
| `↵` / `space` | **play** — cluster up (if needed) → init → drop into shell |
| `i` | init the scenario |
| `v` | verify your fix |
| `r` | reset (wipe namespace + re-init) |
| `h` | show hint |
| `s` | show solution (asks to confirm) |
| `t` | shell wired to the cluster |
| `u` / `d` | cluster up / down |
| `g` | refresh status & progress |
| `a` | about / welcome |
| `?` | help |
| `q` | quit (cluster stays up) |

## The play flow

Pressing **`↵`** on a lesson does the whole setup in one keystroke:

1. brings the `kind` cluster up if it isn't already,
2. runs the lesson's init tasks (builds the broken scenario),
3. drops you into a shell **wired to the cluster** (context `kind-kubelings`,
   namespace `kubelings`) that prints the task.

Inside that shell you have helper commands:

```text
task · hint · verify · solution · klreset · k=kubectl
```

Type `verify` to run the check without leaving the shell; `exit` returns to the TUI
and the marker updates.

## Switching scenarios

Starting a different scenario while one is still active prompts **destroy / keep /
cancel** — so you don't clobber in-progress work by accident.

## Progress markers

Markers are persisted in `.labctl/progress.tsv` and shared with the CLI, so
whether you run a lesson from the TUI or `run-challenge-local.sh`, the state stays
in sync.
