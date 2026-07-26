# learnkit — shared platform for the interactive-course repos

**Date:** 2026-07-26
**Scope:** kubelings, golings, learn-client-go (kubeclientlings), build-your-own-x (byox)
**Goal:** collapse duplication, raise every repo to best-of-breed, and make course #5 (ELK) and #6 (security) cost a manifest + content instead of a fork.

---

## Status (2026-07-26)

**Phase 0 is done and merged** — see §8. Everything below it is a proposal, not a
commitment.

| Repo | Phase 0 | Landing page |
|---|---|---|
| golings | merged (#1) | PR #2 |
| kubelings | merged (#1) | PR #2 |
| byox | merged (#14) | PR #15 |
| kubeclientlings | PR #1 open | PR #2, stacked on #1 |

Landed beyond the original Phase 0 list: byox's run log was quadratic and
unbounded (fixed, capped at 5000 lines); byox and kubelings had no `node` pin,
so their Astro site builds inherited the global mise version and failed; byox
went from **0 tests to 9**.

**The recommendation on the rest still stands, and it is deliberately partial:**

- **Phase 2 (learnsite) — do it.** Best ratio in the plan: ~2,650 duplicated
  LOC in one artifact, and the seam is data (`catalog.json`), not code.
- **Phases 1, 3, 4 (Go lib, abstractions, TUI kit) — defer** until a fifth
  course is actually being built, and let that course drive the interface
  shapes. Designing them now means abstracting for two courses that do not
  exist, from four that have already diverged.

The four demo recordings added in the landing-page PRs are a small preview of
Phase 2: one `AsciinemaPlayer.astro` with `cast`/`caption` props, shared by all
four sites instead of forked four ways.

---

## 1. What actually varies

All four are the same product: *pick an item from a curriculum tree → read instructions → do work → a verifier says pass/fail → progress persists → a static site mirrors the tree.*

| Axis | golings | kubeclientlings | kubelings | byox | ELK (future) | Security (future) |
|---|---|---|---|---|---|---|
| Item | exercise (`.go`) | exercise (`.go` + cluster) | lesson (md + bash tasks) | stage (repo + tester) | lesson (md + queries) | lab (md + vuln target) |
| Manifest | `info.toml` | `info.toml` | `courses/**` frontmatter | `courses.yml` + vendored `course-definition.yml` | TBD | TBD |
| Verifier | `go run`/`go test` + golangci | same + kind | bash script exit code | external tester binary | ES query assertion | flag match / script |
| Environment | none | kind | kind | none | docker compose | docker + network |
| Content home | in-repo | in-repo | in-repo | **cloned external repos** | in-repo | in-repo + external targets |
| Progress | JSON, streak derived | JSON, streak derived | TSV tri-state | JSON, streak stored | — | — |
| CLI | cobra | cobra | hand-rolled `os.Args` | hand-rolled | — | — |
| Site gen | Go 299 LOC | Go 245 LOC | **Python 404 LOC** | Go 338 LOC | — | — |
| Tests | 1.2k LOC | 1.2k LOC | 942 LOC | **0** | — | — |
| CI | lint+test | **none** | catalog-drift + vet+test | **none** | — | — |

**Invariants** (→ the library): curriculum tree, item metadata, verifier contract, environment lifecycle, progress store, two-pane TUI, static-site catalog.

**Variants** (→ per-repo plugins): manifest parser, verifier impl, environment impl, content layout.

---

## 2. Duplication, measured

**Go**
- `kubeclientlings` is a hard fork of `golings`. `exercises/{describe,list,reset,security}.go` byte-identical modulo the product name; `state.go` differs by one `const`; `tui/watch.go` byte-identical; `cmd/*.go` differ by import path + 3 lines; `tui/{model,update,view,keys}.go` ~85% identical. **≈1,700 duplicated lines**, now drifting apart with non-overlapping fixes in each.
- Four independent glamour wrappers, four watchers, four progress stores, four `findRoot`, four two-pane layout engines, two preflight packages (lcg's header literally says *"Ported from kubelings' internal/preflight"*).

**Web**
- `astro.config.mjs`, `package.json`, `tsconfig.json`, `content.config.ts`, `[slug].json.ts`, `hero.css`: identical modulo strings across 3 repos.
- `catalog.astro`: **forked 4×** — 610 / 567 / 580 / 892 LOC ≈ **2,650 LOC of copy-paste**.
- `pages.yml`: 4 near-copies, already drifting (byox uses `go-version-file`, others hardcode `'1.26'`).
- Catalog generators: 882 Go LOC + 404 Python LOC doing the same job four ways.

**Total addressable: ~5,600 LOC → ~1,800 shared.**

---

## 3. Best-of-breed matrix — port these regardless of learnkit

| Solved best in | What | Missing from |
|---|---|---|
| kubelings `runner.go` | `Setpgid` + `SIGKILL(-pid)` — kills the whole child tree | lcg, golings |
| kubelings `markdown.go` | `WithColorProfile(ANSI256)` (glamour falls back to raw `##` off-TTY) + output cache | byox, golings |
| kubelings `main.go` | `findRoot()` from cwd **and** exe dir | golings/lcg watcher hardcodes `os.Getwd()` |
| kubelings | cloud-only gating, scenario-switch guard, `esc` to cancel, `m` to release mouse capture, d2 diagram pipeline | all |
| golings `view.go` | fixed glamour style (never `WithAutoStyle` — it issues a terminal bg query that blocks and sprays bytes in altscreen) + `GLAMOUR_STYLE` override | kubelings, byox |
| golings `state.go` | streak **derived** from timestamps, never stored → cannot drift | byox stores it |
| golings `security.go` | `validatePath` — rejects abs/non-canonical/escaping paths at load | kubelings, byox |
| golings | `notes.md` post-solve walkthrough (`x`) — best learning feature in the set; lint gate; `update` task (stash→pull→pop) | all / byox+kubelings |
| byox `watch.go` | 400ms debounce + ext filter + `Errors` drain + closeable | golings, lcg |
| byox `runner/run.go` | `cmd.WaitDelay` — bounds `Wait` when grandchildren hold the stdout pipe | all |
| byox | regression re-run (stage N reruns 1..N); per-stage reference solutions + prev-stage diff | all |
| kubelings CI | catalog-drift job (regen, fail on `git diff`) — doubles as manifest validation | golings, lcg, byox |
| lcg `preflight` | cheapest-first short-circuit + `/readyz` probe | kubelings |
| lcg `internal/exkit` | shared "always-correct" helper lib so bugs live in exercises only — right instinct, wrong scope | model for the rest |

---

## 4. Bugs found

**P0**
1. `golings/go.mod` = `module github.com/mauricioabreu/golings`, remote = `madhank93/golings`. `go install` impossible.
2. `byox` = `module github.com/madhan/byox` (no such user). Same.
3. golings + lcg `tui/watch.go`: never drains `watcher.Errors` → one error wedges the loop and watch mode dies silently. Also no debounce (editors write 2–4× per save → duplicate runs), no ext filter, only pre-existing dirs watched (`mkdir` a new exercise → invisible), never `Close`d, root is `os.Getwd()` so watch breaks from any subdir.
4. golings: no timeout **and** no cancel key — an infinite-loop exercise wedges the TUI; only exit is killing the terminal.
5. lcg: `exec.CommandContext` kills the `go test` driver, not the test binary it spawned. Against live kind that leaks processes holding namespaces.

**P1**
6. byox engine: 0 tests across 1,544 LOC, no CI at all. `resolveTemplate`'s hand-rolled mustache parser is the highest-risk untested code.
7. lcg + byox: no lint/test workflow whatsoever.
8. byox `tui.go` = 1,041 LOC, 49 top-level decls, one file.
9. No repo caps runner output size — a runaway exercise printing GB fills memory (`strings.Builder` / `CombinedOutput`).

**Security**
10. kubelings `buildRC` interpolates `repoRoot`/`lesson`/`title` into a bash rcfile with Go's `%q`, which is **Go** quoting, not shell quoting. Titles come from your own frontmatter so it's not exploitable today, but `$(...)` in a title reaches the shell. *Fix regardless of the rest of this plan.*
11. byox `repoBase()` = `filepath.Base` of an arbitrary URL from `courses.yml`; slugs feed `filepath.Join` unvalidated. `pathguard` closes it.
12. byox `Setup` clones `--depth=1` of a default branch, builds it, and executes the binary. That's the CodeCrafters model, but an upstream tester-repo compromise = silent RCE on next `just setup`. Pin tester repos to commit SHAs; document the trust boundary.
13. kubelings `shellEnv` writes a cluster-admin kubeconfig at 0600 (good, and cleanup on exit was clearly a fix) but doesn't force `0700` on the temp dir.
14. No dependabot/renovate anywhere. 4 repos × (npm + go), unattended. No `gosec`. `.golangci.yml` in 2 of 4.

---

## 5. learnkit — pros and cons

### Pros
1. **Fixes land once.** Proof this matters: the four watcher bugs live in two repos; the exec-kill fix exists in exactly one; the glamour off-TTY fix in exactly one. Each was fixed where it hurt and never propagated.
2. **Best-of-breed becomes the floor.** lcg has no markdown rendering purely because it forked before golings added it. New repos start with pgroup-kill, debounced watch, cached glamour, path guard, derived streak.
3. **Course #5 is a manifest, not a fork.**
4. **Test leverage.** Shared exec/watch/md/progress tested once covers four consumers — cheaper than 4× suites, and it fixes byox's 0%.
5. **Security class closure.** One audited `pathguard` + one audited `exec` + one audited `shellquote`.
6. **UX consistency.** Today `v`=verify (kubelings), `enter`=run (golings), `t`=test (byox); `n`=next in golings but "no" in kubelings' confirm. Your own muscle memory doesn't transfer between your own tools.
7. **Site: 2,650 forked LOC → one component.** Every site feature (search, keyboard nav, OG images, deep links) currently costs 4×.
8. **Bubbletea v2 migration becomes one project, not four.**

### Cons — and mitigations
1. **Premature abstraction on the TUI.** The four TUIs look alike but differ meaningfully (cluster chrome + switch guards + splash; course-header rows + regression semantics; notes/Explain). A single `learnkit/tui.Model` becomes a 30-flag config monster.
   → **Mitigation, and the single most important decision here: do NOT share the Model. Share the pieces** — list-with-headers widget, filter state machine, keybar, viewport pane, markdown cache, layout math — and let each app own its `Update`. Widgets, not framework.
2. **Coupling four independently-releasing repos.** One bad learnkit bump breaks four things.
   → semver + `go.mod` pins; each repo upgrades on its own schedule; learnkit CI builds all four consumers against `main` nightly, so breakage surfaces upstream.
3. **Contribution friction** — "fix the watcher" becomes 2 PRs + tag + 4 bumps.
   → `go.work` at `/Volumes/work/git-repos` so local edits are instant; tag only when publishing.
4. **Version skew / diamond.** golings on v0.3, kubelings on v0.5 → fixes exist but aren't everywhere. Same failure as today, just visible.
   → renovate + a `mise run bump-learnkit` task.
5. **Over-generalizing the verifier.** ELK/security results may not fit `(Status, string)`.
   → keep it minimal: `Verify(ctx) Result` where `Result{Status, Output, Checks []Check}`; apps extend `Checks`.
6. **byox's vendored-external-content model genuinely doesn't fit** the in-repo model of the other three.
   → `course.Provider` interface with two impls (`fsprovider`, `vendorprovider`); byox keeps its clone/build logic behind its own provider. Don't force it.
7. **A 5th repo to maintain.**
   → it's ~1,800 LOC, leaf-heavy, low-churn once settled.
8. **Bubbletea v1 lock-in across all consumers at once.** Accepted; the pro outweighs it.

### Monorepo — considered, rejected
Maximum dedup and atomic refactors, but: each repo has its own public identity, domain, GH Pages site, and install story; learners clone the *course* repo and edit exercises inside it (a monorepo forces cloning everything — golings alone is 8.4 MB, byox vendors GBs); release cadences differ. **`go.work` + a shared-libs repo gets ~80% of monorepo ergonomics at 0% of the packaging cost.**

---

## 6. Improvements by dimension

### UX (learner-facing)
- **Unify the keymap.** Publish one table; every app uses the same verbs: `↑↓/jk` move · `/` filter · `↵` primary action · `r` reset · `h` hint · `x` explain · `s` solution (gated) · `n` next-unsolved · `?` help · `q` quit. Env-bearing apps add `u`/`d`/`t` in a documented second tier.
- **`n` next-unsolved**: only kubelings + golings → port to lcg, byox.
- **`/` filter**: byox has none, with 10 courses × ~50 stages = ~500 rows. Port.
- **Mouse**: kubelings + golings only. Port wheel + hit-test to lcg, byox. Also port kubelings' `m` release-capture toggle — without it you **cannot copy an error message out of golings**, which is a real daily bug.
- **Cancel a running action** (`esc`): only kubelings. Everywhere.
- **Progress bar / percent complete**: nowhere. Add to the shared header.
- **Streak**: golings + byox only.
- **Resume where I was**: `Tracker.Current` exists in golings/lcg, used by golings only.
- **`notes.md` post-solve walkthrough**: golings only, and it's the best learning feature in the set. Port to all four.
- **Failure output ergonomics**: kubelings renders markdown + centers diagrams; golings dumps raw `go test`. Add `learnkit/outfmt` — surface the first failing assertion, fold the rest.
- **Accessibility**: no `NO_COLOR` handling anywhere; glamour hardcoded dark in kubelings/byox. Add `NO_COLOR`/`--no-color` + a high-contrast style; port golings' `GLAMOUR_STYLE`.
- **First-run**: splash in kubelings + golings; byox drops you into a blank list before `just setup` with no explanation.

### DevEx
- Fix both broken module paths (P0 #1, #2).
- Add lint+test CI to lcg and byox; add golangci-lint to kubelings; add the catalog-drift job to all four.
- **One task runner.** Today: mise (golings, lcg), just (byox), *both* (kubelings). Pick **mise** — it also pins toolchains (go/node/kind), which just doesn't.
- `go.work` at the repos root for cross-repo dev.
- One reusable `workflow_call` pages workflow; one shared `.goreleaser.yaml` (only golings ships binaries today).
- Coverage floor in CI (start at whatever byox reaches, ratchet).
- **Naming alignment** — free, permanent clarity: `docs/` (kubelings) vs `web/` (×3); dir `learn-client-go` vs module `kubeclientlings` vs remote `kubeclientlings`; dir `build-your-own-x` vs remote `byox`.
- Shared `AGENTS.md` conventions (only `golings/web` has one).
- devcontainer in 3 of 4 — standardize.

### Security
Actions: `shellquote` helper for kubelings' rcfile; `pathguard` into byox + kubelings; pin byox tester repos to SHAs + document the RCE trust boundary; `0700` on kubelings' shell temp dir; output-size cap in shared exec; `gosec` + `.golangci.yml` in all four; renovate everywhere; `_headers` CSP on the four sites.

### Features to cross-port
kubelings' **cloud-only gating** (generalizes to "this lab needs X you don't have" — ELK and security both need it) · kubelings' **scenario-switch guard** (any stateful-env course) · kubelings' **d2 → ASCII + SVG diagram pipeline** (generalizes perfectly to ELK/network/security topologies) · byox's **regression re-run** (any progressive-build course) · byox's **per-stage reference solutions + prev-stage diff** · golings' **lint gate** and **`update` task**.

**Nothing has**: cross-content search, bookmarks, time-spent tracking, progress export, a multi-course home (byox is closest).

### What ELK / security need that doesn't exist yet
1. **Environment provider abstraction** — kind vs docker-compose vs docker+network. Hardcoded per repo today.
2. **Verifier abstraction** — `go test` vs bash exit vs external binary vs *ES query assertion* vs *flag string*.
3. **Non-Go exercise languages** — golings/lcg hardcode `exec.Command("go", ...)`. ELK is queries/JSON/configs; security is Python/bash. Needs a command template per item mode.
4. **Flag/answer verification** (CTF style) — submit a string, hash-compare. New primitive.
5. **Reset-to-clean-vulnerable-state** for security labs.
6. **Courses as data** — a new course should be addable without writing Go: `course.yaml` + content dir + optionally a small verifier plugin.

---

## 7. Target architecture

```
github.com/madhank93/learnkit          (Go module, ~1,800 LOC)
├── exec/        pgroup runner, ctx cancel, WaitDelay, output cap     [from kubelings + byox]
├── watch/       debounced fsnotify, ext filter, Errors drain, dir-create  [from byox]
├── md/          glamour: fixed style, explicit color profile, renderer
│                cache per width + output cache per content, NO_COLOR   [kubelings + golings]
├── pathguard/   canonical/relative/rooted path validation            [from golings]
├── shellquote/  POSIX shell quoting                                  [new — fixes kubelings]
├── root/        FindRoot(markers...) from cwd and exe dir            [from kubelings]
├── preflight/   binary + daemon + readiness checks, Issue{Msg,Fix}   [from lcg]
├── progress/    Store iface; derived streak; TSV + JSON backends     [from golings]
├── course/      Item/Group tree + Provider iface (fs, vendor)        [new]
├── verify/      Verifier iface + impls: cmd, script, binary, flag    [new]
├── env/         Provider iface {Up,Down,Status,Reset,Doctor}
│                + impls: none, kind, compose                         [new]
├── catalog/     tree → catalog.json (the site contract)              [replaces 4 generators]
├── outfmt/      failure-output formatting, first-failure surfacing   [new]
└── tui/         WIDGETS ONLY — list-with-headers, filter, keybar,
                 pane layout, splash, confirm, progress bar, keymap   [not a Model]

github.com/madhank93/learnsite         (npm package / template)
├── Catalog.astro          one component, facet-configurable
├── starlightConfig.ts     factory: {title, site, themeColor, keywords, repo}
├── hero.css
└── catalog.schema.json    the contract learnkit/catalog emits
```

**Contract between halves:** `catalog.json`. Go emits it, Astro renders it. Neither knows the other's internals.

---

## 8. Phased plan

### Phase 0 — stop the bleeding (no new repo) · ~1 day
Independent, mechanical, each valuable alone.

| # | Task | Repo | Accept |
|---|---|---|---|
| 0.1 | Fix module path → `github.com/madhank93/golings`; update all imports | golings | `go install github.com/madhank93/golings/golings@latest` works |
| 0.2 | Fix module path → `github.com/madhank93/byox` (both go.mod files) | byox | same |
| 0.3 | Port byox's watcher (debounce, ext filter, Errors drain, Close, dir-create) | golings, lcg | watch survives an fsnotify error; one run per save |
| 0.4 | Port kubelings' pgroup exec | golings, lcg | `esc` kills the whole tree; no orphan test binaries |
| 0.5 | Add `esc` cancel + run timeout | golings | infinite-loop exercise no longer wedges the TUI |
| 0.6 | Shell-quote the rcfile interpolation | kubelings | title containing `$(id)` does not execute |
| 0.7 | `pathguard` copied in (pre-lib) | byox, kubelings | traversal slug rejected at load with a clear error |
| 0.8 | Add lint+test workflow | lcg, byox | CI red on a deliberate vet failure |
| 0.9 | Add catalog-drift job | golings, lcg, byox | CI red on a stale catalog |
| 0.10 | `0700` temp dir; output-size cap | kubelings; all | — |

**Exit:** four repos installable, no known P0, CI on every repo.

### Phase 1 — learnkit leaves · ~2–3 days
Leaf packages only, zero design debate: `exec`, `watch`, `md`, `pathguard`, `shellquote`, `root`, `preflight`, `progress`, `outfmt`.

- Create repo; `go.work` at `/Volumes/work/git-repos`.
- Move Phase-0 fixes in; **write the tests here** (this is where byox's 0% gets fixed by proxy).
- Migrate consumers one at a time, in this order: **lcg → golings → kubelings → byox** (least to most divergent).
- learnkit CI: unit tests + a nightly matrix building all four consumers against `main`.

**Exit:** ~700 LOC deleted across consumers; one place to fix a watcher bug.

### Phase 2 — learnsite · ~3–4 days
- Define `catalog.schema.json`: `{meta, groups[], items[], facets[]}`.
- Build `learnkit/catalog` emitting it; each repo keeps only a small manifest→items adapter. **Deletes kubelings' 404-line Python outlier.**
- Extract `Catalog.astro` (facet-configurable: golings needs Mode, lcg doesn't, byox needs difficulty) + `starlightConfig()` factory + `hero.css`.
- One reusable `workflow_call` pages workflow.

**Exit:** ~2,650 forked LOC → one component; a site feature ships once.

### Phase 3 — abstractions · ~4–5 days
The part that makes ELK cheap.
- `course.Provider` — `fsprovider` (golings/lcg/kubelings) + `vendorprovider` (byox keeps its clone/build logic behind it).
- `verify.Verifier` — `cmd` (templated, **unblocks non-Go languages**), `script`, `binary`, `flag`.
- `env.Provider` — `none`, `kind`, `compose`.
- Retrofit all four. byox stays on `vendorprovider`; if it fights the interface, **let byox keep its own** — that's the documented escape hatch, not a failure.

**Exit:** a course is describable as data + a verifier choice.

### Phase 4 — TUI widget kit · ~4–5 days
Last and highest-risk. **Widgets, not a Model.**
- Extract: list-with-headers + scroll clamp, filter state machine, keybar/footer, pane layout math, splash, confirm dialog, progress bar, shared keymap.
- Drive the design from kubelings' and byox's versions (furthest along).
- Each app keeps its own `Update`. Split byox's 1,041-line `tui.go` while migrating.
- Then port the UX gaps: `n`, `/`, mouse + `m`, `esc`, `notes.md`/`x`, streak, progress bar, `NO_COLOR`.

**Exit:** consistent keymap and behavior across four tools; ~800 LOC deleted.

### Phase 5 — new-course template + ELK pilot · ~3 days + content
- `learnkit-template` repo: manifest + content dir + provider/verifier/env choice + site + CI, wired.
- Build **ELK** from it as the real test of the abstraction: `env: compose`, `verify: cmd` (curl/ES query assertion), content in-repo.
- Anything ELK needs that the template can't express is a Phase-3 gap — fix it in learnkit, not in ELK.

**Exit:** course #5 shipped without forking. Time-to-new-course measured — that's the number that justifies the whole plan.

### Phase 6 — hardening & ops · ongoing
- Security course adds two primitives: **flag verification** and **reset-to-clean-vulnerable-state**. Both land in learnkit.
- renovate across all repos; `gosec` + shared `.golangci.yml`; coverage ratchet.
- Pin byox tester repos to SHAs; document the RCE trust boundary in its README.
- Shared goreleaser + release workflow.
- Naming alignment (`docs/`→`web/`, dir↔module↔remote).
- Cross-content search, bookmarks, progress export.

---

## 9. Sequencing rules

1. **Phase 0 before anything.** Independent value, no lib dependency, removes the excuse to rush the abstraction.
2. **Leaves before trees.** `exec`/`watch`/`md` have obvious right answers. `tui` does not — do it last, once the leaves proved the workflow.
3. **Migrate least-divergent first.** lcg → golings → kubelings → byox.
4. **byox is the escape-hatch canary.** If it fights an interface, the interface is wrong or byox opts out. Do not bend byox to fit.
5. **ELK is the acceptance test for Phase 3**, not a Phase 6 afterthought. If ELK needs a fork, the abstraction failed and should be revised before the security course.

## 10. Kill criteria

Abandon or shrink learnkit if, at Phase 3 review: fewer than two of the four consumers actually adopt an abstraction; or `course.Provider` needs >2 impls for 4 repos; or the TUI kit's config surface exceeds ~10 knobs. In that case keep Phases 0–2 (leaves + site, unambiguous wins) and stop.

## 11. Effort

| Phase | Est. | Risk |
|---|---|---|
| 0 — bleeding | 1 d | none — **done, merged** |
| 1 — leaves | 2–3 d | low |
| 2 — site | 3–4 d | low |
| 3 — abstractions | 4–5 d | **medium** — the real design work |
| 4 — TUI kit | 4–5 d | **medium-high** — over-abstraction trap |
| 5 — template + ELK | 3 d + content | low if 3 is right |
| 6 — hardening | ongoing | low |

**~3.5 weeks of focused work** to go from four forks to a platform. Phases 0–2 (~1.5 weeks) capture most of the value and are safe to stop after.
