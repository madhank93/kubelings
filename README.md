# kubelings

Learn Kubernetes the **rustlings** way — fix small, broken-on-purpose cluster
scenarios one at a time until an automated check turns green.

Each scenario is a self-contained [iximiuz Labs](https://labs.iximiuz.com)
**challenge** (`index.md` with `init`/`verify` tasks in its frontmatter). The same
files run on iximiuz Labs *and* locally on `kind` — one source of truth, no
duplicated scripts.

▶ **Course on iximiuz Labs:** https://labs.iximiuz.com/courses/kubelings-dbd840c8

## Repo layout

```
challenges/<slug>/            # one challenge each (authored unit)
  index.md                    #   frontmatter: playground + init/verify tasks; body: the task + hints
  solution.md                 #   reference solution
courses/kubelings/            # the published Course, composed from the challenges
  index.md                    #   course meta
  module-N/0.index.md         #   module definition
  module-N/<n>.<lesson>/      #   lesson = challenge: index.md (tasks) + unit-1.md (prose + check)
skill-paths/kb-cka-path/      # a track that lists published challenge slugs
scripts/
  run-challenge-local.sh      # run any challenge on local kind
  validators/k8s.sh           # optional local reference helpers (challenges inline their checks)
tools/
  scaffold.sh                 # new challenge from template
  publish.sh                  # register + push a challenge (tracks slugs in .labctl/slugs.tsv)
  challenge-to-lesson.sh      # convert a challenge into a course lesson
.labctl/slugs.tsv             # local id -> remote slug map
```

## Run a challenge locally

Scenarios are portable: `scripts/run-challenge-local.sh` extracts the `init`/`verify`
task scripts from `challenges/<slug>/index.md` and runs them on a local `kind`
cluster — the exact scripts iximiuz Labs runs.

**Prerequisites** (macOS via the dotfiles Brewfile): a Docker runtime
(OrbStack/Docker), `kind`, `kubectl`, `yq`.

```sh
scripts/run-challenge-local.sh up                # one-time: 3-node kind cluster
scripts/run-challenge-local.sh list              # list challenges
scripts/run-challenge-local.sh kb-wl-01 init     # build the scenario
scripts/run-challenge-local.sh kb-wl-01 verify   # check your fix (re-run after each change)
scripts/run-challenge-local.sh kb-wl-01 reset    # wipe ns + re-init
scripts/run-challenge-local.sh kb-wl-01 solution # print the reference solution
scripts/run-challenge-local.sh down              # delete the cluster
```

The challenge arg accepts an id (`kb-wl-01`), a full slug, or a dir path. The
iximiuz-only `machine:` field is ignored locally (everything runs against your
current kube-context). Override the cluster with `KIND_WORKERS=N` /
`KUBELINGS_CLUSTER=name`.

### Typical loop

```sh
scripts/run-challenge-local.sh up
scripts/run-challenge-local.sh kb-wl-07 init      # OOMKilled CrashLoop scenario
kubectl -n kubelings get pods -l app=cache        # diagnose
kubectl -n kubelings set resources deploy/cache --requests=memory=64Mi --limits=memory=128Mi
scripts/run-challenge-local.sh kb-wl-07 verify    # ✅ PASS
```

## Catalog (Workloads & Scheduling)

| id | scenario | type |
|----|----------|------|
| kb-wl-01 | Rolling update — fix unsafe `maxSurge`/`maxUnavailable` | Fix-It |
| kb-wl-02 | Build a node-level log collector DaemonSet | Build-It |
| kb-wl-03 | StatefulSet + headless Service (stable identity) | Build-It |
| kb-wl-04 | Make a never-finishing Job complete | Fix-It |
| kb-wl-05 | Stop CronJob pileup via `concurrencyPolicy` | Fix-It |
| kb-wl-06 | Autoscale a Deployment with an HPA (1→5) | Build-It |
| kb-wl-07 | Right-size memory to stop an OOMKill loop | Debug-It |

## Authoring & publishing (iximiuz Labs)

Requires `labctl` (`brew install labctl`) and `labctl auth login`.

```sh
# new challenge
tools/scaffold.sh kb-wl-08 "My title" Fix-It cka k8s-omni
# edit challenges/kb-wl-08/{index.md,solution.md}, test locally, then:
tools/publish.sh kb-wl-08            # first run registers the suffixed slug + renames dir
tools/publish.sh kb-wl-08            # re-push after edits

# fold a challenge into the course as a lesson
tools/challenge-to-lesson.sh challenges/<slug> courses/kubelings/module-2/8.mylesson mylesson mylesson
labctl content push course kubelings-dbd840c8 --dir courses/kubelings --force
```

See [`iximiuz/README.md`](iximiuz/README.md) for the full publishing workflow and
platform gotchas (slug suffixes, `tagz` vs `categories`, playground limits, etc.).
