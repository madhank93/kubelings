# Publishing kubelings on iximiuz Labs

Kubelings content is authored as **self-contained** iximiuz Labs items (`index.md`
with frontmatter + body). Each challenge's `init:`/`verify:` tasks use plain
`kubectl` + bash inline — no external clone, no `curl`, no public repo needed.

## Layout

- `challenges/<id>/` — **source of truth**: one scenario each (`index.md` +
  `solution.md`). Feeds the local `kind` runner and the course lessons. Not
  published standalone — the Course is the only published product.
- `courses/kubelings/` — the **Course**, composed from the challenges as lessons.
- `scripts/validators/k8s.sh` — **optional** local reference lib (NOT sourced at
  runtime; mirror its logic when inlining a check).
- `tools/scaffold.sh` — generate a new challenge from the proven template.
- `tools/challenge-to-lesson.sh` — fold a challenge into the course as a lesson.
- `tools/publish.sh` — (optional) publish a single standalone challenge; maintains
  `.labctl/slugs.tsv` (id → remote slug).

## Catalog (live)

| id | kind | slug / URL |
|----|------|------------|
| kubelings | course | https://labs.iximiuz.com/courses/kubelings-dbd840c8 |

All 7 Workloads scenarios were verified end-to-end on a `k8s-omni` playground
(init builds the scenario, verify fails pre-fix, passes post-fix).

## Workflow (add a scenario to the Course)

```sh
brew install labctl          # in dotfiles Brewfile
labctl auth login            # one-time

# 1. author the scenario (source of truth)
tools/scaffold.sh kb-wl-08 "My title" Fix-It cka k8s-omni
# edit challenges/kb-wl-08/{index.md,solution.md}; test locally:
scripts/run-challenge-local.sh kb-wl-08 init && scripts/run-challenge-local.sh kb-wl-08 verify

# 2. fold it into the course as a lesson, then push the whole course
tools/challenge-to-lesson.sh challenges/kb-wl-08 courses/kubelings/module-2/8.foo foo foo
labctl content push course kubelings-dbd840c8 --dir courses/kubelings --force
```

> Publishing a one-off standalone challenge is still possible with
> `tools/publish.sh <id>` (registers a suffixed slug), but the Course is the
> shipped product.

### Course structure

```
courses/kubelings/
  index.md                       # kind: course
  module-N/0.index.md            # kind: module  (numeric prefix orders modules)
  module-N/<n>.<lesson>/index.md # kind: lesson  (playground + tasks; numeric prefix orders lessons)
  module-N/<n>.<lesson>/unit-K.md# kind: unit    (prose + ::simple-task bound to a task)
```

A lesson carries the `playground` + `tasks`; its units render `::simple-task`
blocks that turn green when a task passes. `challenge-to-lesson.sh` generates a
lesson (`index.md` + `unit-1.md`) straight from a challenge — keep the challenge
as the source of truth and regenerate.

## Platform gotchas (learned the hard way)

- **Slug suffix:** `content create` appends a random suffix (`-53e1821a`). Keep the
  local dir name equal to the slug so default `--dir` resolves. `publish.sh` does this.
- **`tagz` ≠ categories:** tagz must not contain category words
  (kubernetes/networking/security/...) — validation 400s. Put those in `categories`.
- **`solution.md` title ≥ 10 chars:** its first H1 (or a frontmatter `title`) must be
  ≥ 10 characters, else push 400s.
- **skill-path has no `difficulty`** attribute (challenges do: `easy|medium|hard`).
- **Course lessons drop `categories`/`tagz`/`difficulty`** — a lesson takes
  `kind/title/description/name/slug/createdAt/playground/tasks`. `content create`
  scaffolds sample modules/lessons; `push` deletes any remote files not present
  locally, so the sample is cleaned automatically on first real push.
- **Custom playgrounds can't run install scripts.** The manifest schema
  (`labctl playground manifest <name>`) only tweaks topology/resources/tabs/networks;
  there is no `initScript`/`image`. Install software (metrics-server, Cilium…) via the
  challenge `init:` task instead. `labctl playground save`/`exec` do **not** exist —
  run ad-hoc commands with `labctl ssh <id> -m <machine> -- '<cmd>'`.
- **`init:` tasks block the loading screen** — keep heavy setup reasonable
  (metrics-server install is ~60–90s; bump `timeout_seconds`).
- **Images:** prefer `ghcr.io/iximiuz/labs/*`; Docker Hub images (e.g. `polinux/stress`)
  work but are rate-limited.
- **k8s-omni machines:** `cplane-01`, `node-01`, `node-02`, `dev-machine`. Run tasks on
  `cplane-01` (admin kubeconfig present).
