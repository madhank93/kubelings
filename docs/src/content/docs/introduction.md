---
title: Introduction
description: What Kubelings is and how the check-driven lessons work.
---

Kubelings teaches Kubernetes the way *rustlings* teaches Rust: a series of small,
broken-on-purpose scenarios you fix one at a time.

## The loop

1. **Read the situation** — each lesson explains what's broken (or what to build)
   and what "done" looks like.
2. **Fix the cluster** — you get a live, multi-node Kubernetes cluster. Use
   `kubectl` to diagnose and repair.
3. **Watch the check** — every lesson has an automated check. It turns green only
   when the cluster is genuinely in the target state.

Progress is tracked per lesson:

- `◌` not started
- `◐` started (you ran init)
- `✓` solved (verify passed)

## One source of truth

The **course** (`courses/kubelings/`) is the single source of truth and the only
thing published on iximiuz Labs. Each lesson is self-contained: its `index.md`
carries the playground + `init`/`verify` tasks, and its `unit-*.md` files hold the
prose, the interactive check, and the solution.

The local `kind` runner reads those same task blocks, so **local and lab stay in
lockstep** — no duplicated scripts. See [Architecture](/reference/architecture/).

## Where to go next

- [Getting Started](/getting-started/) — install and run your first lesson.
- [The TUI](/guides/tui/) — the interactive terminal app.
- [Lessons](/guides/lessons/) — the catalog and how to add one.
