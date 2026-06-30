# Publishing kubelings on iximiuz Labs

Kubelings content is authored as **self-contained** iximiuz Labs items (`index.md`
with frontmatter + body). Each challenge's `init:`/`verify:` tasks use plain
`kubectl` + bash inline — no external clone, no `curl`, no public repo needed.

## Layout

- `challenges/<slug>/` — one challenge each (`index.md` + `solution.md` + `__static__/`).
- `skill-paths/<slug>/` — composes published challenges into a track.
- `iximiuz/kubelings-svc-selector-75c42c07/` — the original pilot challenge.
- `scripts/validators/k8s.sh` — **optional** local reference lib (NOT sourced at
  runtime; mirror its logic when inlining a check).
- `tools/scaffold.sh` — generate a new challenge from the proven template.
- `tools/publish.sh` — register (first run) + push a challenge; maintains
  `.labctl/slugs.tsv` (challenge id → remote slug).

## Catalog (live)

| id | kind | slug / URL |
|----|------|------------|
| svc-selector | challenge | https://labs.iximiuz.com/challenges/kubelings-svc-selector-75c42c07 |
| kb-wl-01 | challenge | https://labs.iximiuz.com/challenges/kb-wl-01-53e1821a |
| kb-wl-02 | challenge | https://labs.iximiuz.com/challenges/kb-wl-02-6c8af3fb |
| kb-wl-03 | challenge | https://labs.iximiuz.com/challenges/kb-wl-03-e73bdf82 |
| kb-wl-04 | challenge | https://labs.iximiuz.com/challenges/kb-wl-04-a6bb83fd |
| kb-wl-05 | challenge | https://labs.iximiuz.com/challenges/kb-wl-05-723804ee |
| kb-wl-06 | challenge | https://labs.iximiuz.com/challenges/kb-wl-06-6c1df5e8 |
| kb-wl-07 | challenge | https://labs.iximiuz.com/challenges/kb-wl-07-d4d9a2d1 |
| kb-cka-path | skill-path | https://labs.iximiuz.com/skill-paths/kb-cka-path-85a1808a |

All `kb-wl-*` were verified end-to-end on a `k8s-omni` playground
(init builds the scenario, verify fails pre-fix, passes post-fix).

## Workflow

```sh
brew install labctl          # in dotfiles Brewfile
labctl auth login            # one-time

# new challenge:
tools/scaffold.sh kb-wl-08 "My title" Fix-It cka k8s-omni
# edit challenges/kb-wl-08/{index.md,solution.md}
tools/publish.sh kb-wl-08    # first run registers the suffixed slug + renames dir

# re-publish after edits:
tools/publish.sh kb-wl-08
```

Skill-paths are published manually (publish.sh handles `challenge` only):

```sh
labctl content create skill-path <name> --dir /tmp/empty   # once; note the slug
labctl content push  skill-path <slug> --dir skill-paths/<slug> --force
```

## Platform gotchas (learned the hard way)

- **Slug suffix:** `content create` appends a random suffix (`-53e1821a`). Keep the
  local dir name equal to the slug so default `--dir` resolves. `publish.sh` does this.
- **`tagz` ≠ categories:** tagz must not contain category words
  (kubernetes/networking/security/...) — validation 400s. Put those in `categories`.
- **`solution.md` title ≥ 10 chars:** its first H1 (or a frontmatter `title`) must be
  ≥ 10 characters, else push 400s.
- **skill-path has no `difficulty`** attribute (challenges do: `easy|medium|hard`).
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
