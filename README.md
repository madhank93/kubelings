# kubelings

Learn Kubernetes the **rustlings** way — fix small, broken-on-purpose cluster
scenarios one at a time until an automated check turns green.

Everything lives in one place: the [iximiuz Labs](https://labs.iximiuz.com)
**Course**. Each lesson carries its own `init`/`verify` tasks, so the same files
run on iximiuz Labs *and* locally on `kind` — single source of truth, no
duplicated scripts.

▶ **Course:** https://labs.iximiuz.com/courses/kubelings-dbd840c8
📖 **Docs:** https://kubelings.madhan.app (source in [`docs/`](docs/), Astro Starlight)

## Repo layout

```
courses/kubelings/                       # THE source of truth (and the published Course)
  index.md                               #   course meta
  module-N/0.index.md                    #   module definition (numeric prefix orders modules)
  module-N/<n>.<lesson>/
    index.md                             #   lesson: playground + init/verify tasks
    unit-1.md                            #   prose: the task, hints, ::simple-task check, solution
scripts/
  run-challenge-local.sh                 # THE engine: run any lesson on local kind
  validators/k8s.sh                      # optional local reference helpers (lessons inline their checks)
cmd/kubelings/                           # the TUI entrypoint (Go)
internal/{course,progress,runner,preflight,ui}/  # TUI internals (UI-only; execs the runner)
tools/
  scaffold-lesson.sh                     # new lesson from template
justfile                                 # just tui | doctor | up | down | run <lesson> <verb>
.labctl/slugs.tsv                        # local id -> remote slug (the course)
```

There is no separate `challenges/` directory — the course lessons **are** the
scenarios. The local runner reads the lesson `index.md` task blocks directly.

## Run locally — TUI

The fastest way to practice locally is the interactive TUI (`cmd/kubelings`): a
list of scenarios grouped by module with progress markers, one-key
init/verify/reset, hint/solution, a cluster-wired shell, and live cluster status.

```sh
just tui          # build + launch  (or: go run ./cmd/kubelings)
just doctor       # headless: env, cluster status, lessons (no TUI)
```

Keys: **`↵`/`space` play** (cluster up if needed → init → drop into shell) ·
`i` init · `v` verify · `r` reset · `h` hint · `s` solution (confirms first) ·
`t` shell · `u`/`d` cluster up/down · `g` refresh · `?` help · `q` quit. Markers:
`◌` not started · `◐` started · `✓` solved (in `.labctl/progress.tsv`, shared with
the CLI). Starting a different scenario while one is still active prompts
**destroy / keep / cancel**.

The cluster shell (`t`, or the play key) opens pre-wired to `kind-kubelings` / ns
`kubelings` and **prints the task**, with helper commands: `task`, `hint`,
`verify`, `solution`, `klreset`, `k`=kubectl. Open it standalone too:

```sh
kubelings shell rolling-update      # task + cluster-wired shell for one lesson
```

The TUI is UI-only — it delegates every action to `run-challenge-local.sh`, so the
CLI and TUI stay in lockstep. Build prereqs: Go ≥ 1.25 (TUI), plus the runner
prereqs below.

> **Security:** lesson task scripts are treated as untrusted code — they run
> inside the kind node container (not on your host), confined to the course tree
> and the kind cluster, with Pod Security `baseline` enforced on the lesson
> namespace. See [`SECURITY.md`](SECURITY.md).

## Run a lesson locally — CLI

`scripts/run-challenge-local.sh` extracts the `init`/`verify` task scripts from a
lesson's `index.md` and runs them on a local `kind` cluster — the exact scripts
iximiuz Labs runs.

**Prerequisites** (macOS via the dotfiles Brewfile): a Docker runtime
(OrbStack/Docker), `kind`, `kubectl`, `yq`.

**Reproducible toolchain:** `mise.toml` pins Go + the CLIs; `go.sum` locks the Go
deps. A fresh clone gets the exact environment with:

```sh
mise install   # fetch the pinned go/kubectl/kind/yq  (Docker still needed)
mise run setup # install + go build + go test
```

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
