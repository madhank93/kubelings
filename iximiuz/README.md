# Publishing kubelings on iximiuz Labs

The **Course** (`courses/kubelings/`) is the single source of truth and the only
published product. Each lesson is self-contained: its `index.md` carries the
`playground` + `init`/`verify` tasks (plain `kubectl` + bash), and its `unit-*.md`
files hold the prose, the `::simple-task` check, and the solution.

## Layout

- `courses/kubelings/` — the Course (source of truth + published).
- `scripts/run-challenge-local.sh` — run any lesson on local `kind`.
- `scripts/validators/k8s.sh` — **optional** local reference lib (NOT sourced at
  runtime; mirror its logic when inlining a check).
- `tools/scaffold-lesson.sh` — scaffold a new lesson.

## Catalog (live)

| id | kind | slug / URL |
|----|------|------------|
| kubelings | course | https://labs.iximiuz.com/courses/kubelings-dbd840c8 |

All 7 Workloads lessons were verified end-to-end on local `kind` and a `k8s-omni`
playground (init builds the scenario, verify fails pre-fix, passes post-fix).

## Course structure

```
courses/kubelings/
  index.md                        # kind: course
  module-N/0.index.md             # kind: module  (numeric prefix orders modules)
  module-N/<n>.<lesson>/index.md  # kind: lesson  (playground + tasks; numeric prefix orders lessons)
  module-N/<n>.<lesson>/unit-K.md # kind: unit    (prose + ::simple-task bound to a task + solution)
```

A lesson carries the `playground` + `tasks`; its units render `::simple-task`
blocks that turn green when a task passes.

## Workflow

```sh
brew install labctl          # in dotfiles Brewfile
labctl auth login            # one-time

# scaffold a lesson, edit it, test locally
tools/scaffold-lesson.sh module-2 8 ingress "Fix the broken Ingress" k8s-omni
scripts/run-challenge-local.sh ingress init && scripts/run-challenge-local.sh ingress verify

# publish the whole course
labctl content push course kubelings-dbd840c8 --dir courses/kubelings --force
```

`push` deletes any remote files not present locally, so renames/removals sync
cleanly. First-time course registration was a one-off
`labctl content create course kubelings`.

## Platform gotchas (learned the hard way)

- **Slug suffix:** `content create` appends a random suffix (e.g. `-dbd840c8`).
  The course slug is recorded in `.labctl/slugs.tsv`.
- **`tagz` ≠ categories:** tagz must not contain category words
  (kubernetes/networking/security/...) — validation 400s. Put those in `categories`.
- **Title length ≥ 10 chars** for course/module/lesson titles, else push 400s.
- **Lesson frontmatter** takes `kind/title/description/name/slug/createdAt/playground/tasks`
  only — no `categories`/`tagz`/`difficulty` (those belong on the course).
- **Custom playgrounds can't run install scripts.** The manifest schema
  (`labctl playground manifest <name>`) only tweaks topology/resources/tabs/networks;
  there is no `initScript`/`image`. Install software (metrics-server, Cilium…) via a
  lesson `init:` task instead. `labctl playground save`/`exec` do **not** exist —
  run ad-hoc commands with `labctl ssh <id> -m <machine> -- '<cmd>'`.
- **`init:` tasks block the loading screen** — keep heavy setup reasonable
  (metrics-server install is ~60–90s; bump `timeout_seconds`).
- **Images:** prefer `ghcr.io/iximiuz/labs/*`; Docker Hub images (e.g. `polinux/stress`)
  work but are rate-limited.
- **k8s-omni machines:** `cplane-01`, `node-01`, `node-02`, `dev-machine`. Run tasks on
  `cplane-01` (admin kubeconfig present).
