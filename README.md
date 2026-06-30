# kubelings

Learn Kubernetes the rustlings way — fix small, broken-on-purpose cluster
scenarios one at a time until an automated check passes.

Each exercise is **portable data**, not code, so the same files run on a hosted
platform (iximiuz Labs, killercoda) or locally on `kind`:

```
exercises/<NN-topic>/<name>/
  task.md        # what's broken and what "done" looks like
  init.sh        # creates the broken scenario in the cluster
  verify.sh      # polls cluster state; exit 0 = solved, non-zero = not yet
  hint.md        # progressive hints
  solution.yaml  # the fix (kept on a separate `solution` branch later)
  meta.yaml      # title, tier, topic, concepts (for curriculum generation)
```

`init.sh` and `verify.sh` are plain `kubectl` + `bash`, so a platform's
"initialize" / "verify" hooks can call them directly, and the local runner uses
the same scripts.

## Run a pilot locally

Needs `kind` + `kubectl` + Docker.

```sh
scripts/run-local.sh exercises/01-services/svc-selector up      # one-time: create the kind cluster
scripts/run-local.sh exercises/01-services/svc-selector init    # apply the broken scenario
scripts/run-local.sh exercises/01-services/svc-selector verify  # check your fix (re-run after editing)
scripts/run-local.sh exercises/01-services/svc-selector reset   # wipe + re-apply the broken baseline
scripts/run-local.sh exercises/01-services/svc-selector solve   # apply the answer (to see it pass)
```

## Run an iximiuz-format challenge locally

The `challenges/<slug>/` items are authored for iximiuz Labs (one `index.md` with
`init`/`verify` tasks in the frontmatter). `scripts/run-challenge-local.sh` runs
those same task scripts on a local `kind` cluster — no separate `init.sh`/`verify.sh`
needed, so local and lab stay in sync.

Needs `kind` + `kubectl` + `yq` + a Docker runtime (OrbStack/Docker).

```sh
scripts/run-challenge-local.sh up                # one-time: 3-node kind cluster
scripts/run-challenge-local.sh list              # list challenges
scripts/run-challenge-local.sh kb-wl-01 init     # build the scenario
scripts/run-challenge-local.sh kb-wl-01 verify   # check your fix (re-run after editing)
scripts/run-challenge-local.sh kb-wl-01 reset    # wipe ns + re-init
scripts/run-challenge-local.sh kb-wl-01 solution # print solution.md
scripts/run-challenge-local.sh down              # delete the cluster
```

The challenge arg accepts an id (`kb-wl-01`), full slug, or dir path. The
iximiuz-only `machine:` field is ignored locally (everything runs against your
kind context).

## On iximiuz Labs / killercoda

Point the challenge's **init** step at `init.sh` and its **verify/check** step at
`verify.sh`. The platform provides an isolated cluster per learner — so a broken
exercise can't affect anyone else, and "reset" is just re-running `init.sh`.
