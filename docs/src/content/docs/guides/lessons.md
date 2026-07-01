---
title: Lessons
description: The Kubelings lesson catalog and how to add a new one.
---

Lessons live under `courses/kubelings/module-N/<n>.<name>/`. Each has:

- `index.md` — frontmatter with the `playground` and `init`/`verify` **tasks**.
- `unit-1.md` — the prose: situation, task, hint, the interactive check, and the
  solution (in a collapsible block).

## Catalog — Workloads & Scheduling

| Lesson | Scenario | Type |
|--------|----------|------|
| `rolling-update` | fix unsafe `maxSurge`/`maxUnavailable` | Fix-It |
| `daemonset` | build a node-level log collector | Build-It |
| `statefulset` | StatefulSet + headless Service (stable identity) | Build-It |
| `jobs` | make a never-finishing Job complete | Fix-It |
| `cronjobs` | stop CronJob pileup via `concurrencyPolicy` | Fix-It |
| `hpa` | autoscale a Deployment with an HPA (1→5) | Build-It |
| `oomkill` | right-size memory to stop an OOMKill loop | Debug-It |

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
tools/scaffold-lesson.sh module-2 8 ingress "Fix the broken Ingress" k8s-omni
# edit courses/kubelings/module-2/8.ingress/{index.md,unit-1.md}

# test locally
scripts/run-challenge-local.sh ingress init && scripts/run-challenge-local.sh ingress verify

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
