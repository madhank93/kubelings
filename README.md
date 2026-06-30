# kubelings

Learn Kubernetes the **rustlings** way — fix small, broken-on-purpose cluster
scenarios one at a time until an automated check turns green.

Everything lives in one place: the [iximiuz Labs](https://labs.iximiuz.com)
**Course**. Each lesson carries its own `init`/`verify` tasks, so the same files
run on iximiuz Labs *and* locally on `kind` — single source of truth, no
duplicated scripts.

▶ **Course:** https://labs.iximiuz.com/courses/kubelings-dbd840c8

## Repo layout

```
courses/kubelings/                       # THE source of truth (and the published Course)
  index.md                               #   course meta
  module-N/0.index.md                    #   module definition (numeric prefix orders modules)
  module-N/<n>.<lesson>/
    index.md                             #   lesson: playground + init/verify tasks
    unit-1.md                            #   prose: the task, hints, ::simple-task check, solution
scripts/
  run-challenge-local.sh                 # run any lesson on local kind
  validators/k8s.sh                      # optional local reference helpers (lessons inline their checks)
tools/
  scaffold-lesson.sh                     # new lesson from template
.labctl/slugs.tsv                        # local id -> remote slug (the course)
```

There is no separate `challenges/` directory — the course lessons **are** the
scenarios. The local runner reads the lesson `index.md` task blocks directly.

## Run a lesson locally

`scripts/run-challenge-local.sh` extracts the `init`/`verify` task scripts from a
lesson's `index.md` and runs them on a local `kind` cluster — the exact scripts
iximiuz Labs runs.

**Prerequisites** (macOS via the dotfiles Brewfile): a Docker runtime
(OrbStack/Docker), `kind`, `kubectl`, `yq`.

```sh
scripts/run-challenge-local.sh up                 # one-time: 3-node kind cluster
scripts/run-challenge-local.sh list               # list runnable lessons
scripts/run-challenge-local.sh rolling-update init # build the scenario
scripts/run-challenge-local.sh rolling-update verify   # check your fix (re-run after each change)
scripts/run-challenge-local.sh rolling-update reset     # wipe ns + re-init
scripts/run-challenge-local.sh rolling-update solution  # print the lesson (incl. solution)
scripts/run-challenge-local.sh down               # delete the cluster
```

`<lesson>` accepts a lesson name (e.g. `rolling-update`), its slug, or a dir path.
The iximiuz-only `machine:` field is ignored locally (everything runs against your
current kube-context). Override with `KIND_WORKERS=N` / `KUBELINGS_CLUSTER=name`.

### Typical loop

```sh
scripts/run-challenge-local.sh up
scripts/run-challenge-local.sh oomkill init       # OOMKilled CrashLoop scenario
kubectl -n kubelings get pods -l app=cache        # diagnose
kubectl -n kubelings set resources deploy/cache --requests=memory=64Mi --limits=memory=128Mi
scripts/run-challenge-local.sh oomkill verify     # ✅ PASS
```

## Lessons (Module 2 — Workloads & Scheduling)

| lesson | scenario | type |
|--------|----------|------|
| rolling-update | fix unsafe `maxSurge`/`maxUnavailable` | Fix-It |
| daemonset | build a node-level log collector | Build-It |
| statefulset | StatefulSet + headless Service (stable identity) | Build-It |
| jobs | make a never-finishing Job complete | Fix-It |
| cronjobs | stop CronJob pileup via `concurrencyPolicy` | Fix-It |
| hpa | autoscale a Deployment with an HPA (1→5) | Build-It |
| oomkill | right-size memory to stop an OOMKill loop | Debug-It |

## Add a lesson & publish

Requires `labctl` (`brew install labctl`) and `labctl auth login`.

```sh
# scaffold a lesson under a module
tools/scaffold-lesson.sh module-2 8 ingress "Fix the broken Ingress" k8s-omni
# edit courses/kubelings/module-2/8.ingress/{index.md,unit-1.md}

# test locally
scripts/run-challenge-local.sh ingress init && scripts/run-challenge-local.sh ingress verify

# publish the whole course
labctl content push course kubelings-dbd840c8 --dir courses/kubelings --force
```

See [`iximiuz/README.md`](iximiuz/README.md) for the course schema and platform
gotchas (lesson frontmatter, `tagz` vs `categories`, playground limits, etc.).
