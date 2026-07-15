---
title: Authoring lessons
description: How Kubelings lessons are structured, run, and added. Browse all lessons in the Catalog.
---

Looking for the list of lessons? Browse and filter all 107 in the
**[Catalog](/catalog)**. This page is for *authoring* — how a lesson is built.

Lessons live under `courses/kubelings/module-N/<n>.<name>/`. Each has:

- `index.md` — frontmatter with the `playground` and `init`/`verify` **tasks**.
- `unit-1.md` — the prose: situation, task, hint, the interactive check, and the
  solution (in a collapsible block).

Four lesson types (the TUI and Catalog show a badge for each):

| Type | Meaning |
|---|---|
| **lab** | hands-on concept lesson, verify-gated |
| **incident** | a real, cited production incident, reproduced (hands-on) |
| **drill** | a synthetic failure pattern seen across many companies (hands-on) |
| **read** | guided reading (runbooks) — no tasks, read-only |

## Running a lesson

```sh
scripts/run-challenge-local.sh <lesson> init     # build it
scripts/run-challenge-local.sh <lesson> verify   # check your fix
scripts/run-challenge-local.sh <lesson> reset    # wipe + re-init
scripts/run-challenge-local.sh <lesson> solution # print the lesson (incl. solution)
```

`<lesson>` accepts a lesson name (e.g. `rolling-update`), its slug, or a dir path
(confined to the course tree).

## Adding a lesson

```sh
tools/scaffold-lesson.sh module-2 14 my-lesson "Fix the broken thing" k8s-omni
# edit courses/kubelings/module-2/14.my-lesson/{index.md,unit-1.md}

# test locally
scripts/run-challenge-local.sh my-lesson init && scripts/run-challenge-local.sh my-lesson verify

# publish the whole course
labctl content push course kubelings-dbd840c8 --dir courses/kubelings --force
```

### Task authoring rules

- `init` tasks build the broken/baseline state; `verify` exits `0` when solved,
  non-zero otherwise.
- Keep checks plain `kubectl` + bash.
- Lesson scripts run **inside the kind node**, not on your host — see
  [Security](/reference/security/). Avoid `hostPath`/privileged pods; the lesson
  namespace enforces Pod Security `baseline`.
- Incident replays carry a verified `source:` URL, surfaced on the lesson's
  [Catalog](/catalog) row.
