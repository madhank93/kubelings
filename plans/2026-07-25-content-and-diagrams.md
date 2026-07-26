# Content quality + shared diagrams (TUI & site)

Date: 2026-07-25
Scope: `courses/kubelings/**` (116 lessons), `internal/ui`, `internal/course`, `docs/`

## Goal

1. TUI shows the lesson body as rendered markdown, not a metadata card.
2. Lesson prose is short, scannable, and consistent across all 116 lessons.
3. Diagrams come from **one source file per diagram** and render in both the terminal
   (ASCII/Unicode) and the website (SVG).

---

## 1. Diagram toolchain — decision: **D2**

`.d2` source → build step emits both targets:

| target | command | consumer |
|---|---|---|
| terminal | `d2 topology.d2 topology.txt` (`.txt` ⇒ ASCII renderer) | TUI + `task` in the lesson shell |
| site | `d2 --theme 0 --dark-theme 200 topology.d2 topology.svg` | Starlight lesson page |

Why D2 over the alternatives:

- **D2 0.7.1 ships an ASCII renderer** — any `.txt` output path uses it; default charset is
  Unicode box-drawing, `--ascii-mode=standard` forces 7-bit ASCII. No other mainstream
  diagram tool renders the *same source* to terminal and SVG.
- **Written in Go.** Same toolchain as the TUI; installable as a pinned CLI (mise) and, if
  ever needed, embeddable as a library (`oss.terrastruct.com/d2/d2lib`). No Node, no
  Playwright, no headless Chrome in the docs build.
- Mermaid was the obvious candidate and loses on both ends: SVG at build time needs
  `rehype-mermaid` + Playwright (heavy, flaky in CI), and terminal rendering has no
  maintained renderer (`mermaid-ascii` supports a small subset of flowcharts only).
- Graphviz + `graph-easy` means two toolchains and a Perl dependency for the ASCII half.
- Hand-drawn ASCII is zero-dependency but has no SVG path and drifts the moment a lesson
  changes.

Known constraint: **the D2 ASCII renderer is alpha** and has no width control. Mitigation is
a house style — see §1.2 — plus a lint gate on rendered width.

### 1.1 Single-source flow

Source of truth stays `courses/kubelings/<module>/<NN.lesson>/`. Add per lesson:

```
diagrams/topology.d2        # hand-written, the only file a human edits
diagrams/topology.txt       # generated, committed
diagrams/topology.svg       # generated, committed
```

Generated artifacts are **committed** so the TUI has zero runtime dependencies and the
Astro build never shells out to `d2`.

`unit-1.md` embeds the ASCII inside marked fences:

```markdown
<!-- d2:topology -->
```text
 ┌──────────┐        ┌──────────┐
 │ Service  │ ─ ─ ─▶ │  (none)  │
 │ app=api- │        │ endpoints│
 │ server   │        └──────────┘
 └──────────┘
```
<!-- /d2:topology -->
```

- **TUI / iximiuz / GitHub**: it is a plain fenced block — renders everywhere, no directive
  support needed from any platform. This is the reason for embedding rather than inventing
  a `::diagram` directive: iximiuz Labs renders these same files and we cannot extend its
  markdown pipeline.
- **Site**: `gen-catalog.py` swaps the marked block for `<img src="/diagrams/<slug>-topology.svg">`
  when it writes `docs/src/data/lesson-details/<slug>.md`.
- If the swap never runs, the ASCII still shows. Graceful degradation, no broken page.

The sync tool (`tools/diagrams/main.go`, or a `just` recipe over the `d2` CLI) does:
`.d2` → `.txt` → inject between markers → `.d2` → `.svg`, then fails if any committed
artifact is stale (CI gate).

### 1.2 Diagram house style (drives ASCII fidelity)

**What the diagram must say.** It shows the *causal chain the lesson is about* — the step
where the mechanism breaks — not a restatement of the title and not an inventory of objects.
"Service → empty endpoint list → pods that never match" teaches; "a Service, some pods"
does not. If the diagram would only name things the prose already names, skip it.

**Size.** Max 6 nodes / 8 edges, one idea. Labels short; detail belongs in prose. No
animation, sketch mode, or custom fonts (all no-ops in ASCII).

**Orientation is automatic.** `scripts/gen-diagrams.py` renders each source both ways and
keeps the one that fits and reads best: fits 56 columns first, then fewest rows, then
narrowest. An explicit `direction:` in the source is treated as an author override and left
alone. In practice 2-node chains land horizontal (41×4), 3+ nodes go vertical (22×20) — a
4-node vertical chain already costs 28 rows, past the budget.

**Budgets, enforced by the generator (hard failures):**

- ≤ **56 columns** — the TUI right pane at an 80-column terminal.
- ≤ **24 rows** — taller means scrolling past the picture to reach the task.

Fix a violation in the diagram (shorter labels, fewer nodes, a step folded into an edge
label), never by widening the pane.

Diagram only where structure is the lesson (traffic path, controller loop, scheduling,
ownership chains). Not for lessons whose insight is a single command's output.

Target: ~40 of 116 lessons get a diagram, biased to M4 Networking, M2 Workloads
controller-behaviour lessons, and the storage/PV lifecycle lessons.

---

## 2. TUI content rendering

Current state: `detail()` prints title, raw description, and metadata rows; the actual
lesson body only appears inside the shell. Hint/solution already go through glamour
(`internal/ui/markdown.go`).

Changes:

1. **Render the body in the detail pane.** `course.Lesson.Task` is already parsed from
   `unit-1.md`; pass it through `renderMarkdown(l.Task, m.vp.Width)` under the metadata
   rows. The pane is already scrollable (viewport + wheel), so length is not a problem.
2. **Render the description too** — it contains backticks (`` `api` ``) that today print raw.
3. **Cache renders.** Add `mdCache map[string]string` on `model`, keyed
   `lesson|mode|width`; glamour re-rendering on every cursor move and resize is the one
   real cost of (1). Invalidate on `reload()` and on width change.
4. **Style pass on `mdStyle`**: heading prefixes are already stripped; also flatten
   `BlockQuote`/`CodeBlock` margins so a fenced block doesn't eat 4 columns of a 56-column
   pane, and set `Document.Margin = 0`.
5. **Diagram passthrough**: the ASCII block is a ```text fence — confirm glamour renders it
   without re-wrapping (word wrap at pane width would destroy box art). If it re-wraps,
   render fenced `text` blocks ourselves (split on the fence, emit verbatim, dim border).
   This is the one genuine technical risk in the TUI half; spike it first (§P0).

---

## 3. Content quality — rubric + linter

### 3.1 Lesson template (enforced order)

| Section | Budget | Rule |
|---|---|---|
| Situation | ≤ 70 words | What is broken, in the voice of the on-call engineer. No API explanation. |
| Signal | 1 command + its output | The one command that reveals the fault. Verbatim block. |
| Diagram | optional | Only when structure is the lesson. |
| Your task | 3–5 numbered items | Each independently checkable; must match `verify_done`. |
| Hint (`<details>`) | ≤ 60 words | Progressive: name the mechanism, then the command. Never the answer verbatim. |
| Solution (`<details>`) | Root cause → Fix → Verify → Prevention | Four fixed subheads. Prevention ≤ 3 bullets. |

Style rules: line width ≤ 90 chars; active voice; second person; no "simply/just/obviously";
resource names in backticks; every `kubectl` line runnable as-is with `-n kubelings`.

### 3.2 Linter (`tools/lintcontent`, Go, ~200 lines)

Checks per lesson, exit non-zero on violation:

- required sections present, in order, within word budget
- `<details>` hint **and** solution exist for every lesson with `tasks:` (known gap:
  advanced lessons are missing these)
- solution has all four subheads
- every fenced `sh` block's `kubectl` invocations parse and reference namespace `kubelings`
- prose line width, banned-word list, heading depth
- task count in "Your task" ≥ 3 and ≤ 5
- diagram blocks: markers balanced, ASCII width ≤ 56, `.txt`/`.svg` regenerable from `.d2`
  (byte-compare against a fresh render)

Wire as `just lint-content` and a CI job. This is what makes a 116-lesson rewrite tractable —
the linter finds the work, the rewrite is then mechanical per lesson.

---

## 4. Site

- `gen-catalog.py`: swap ASCII blocks → `<img>`; copy `*.svg` into `docs/public/diagrams/`.
- D2's `--theme 0 --dark-theme 200` produces a single SVG that follows the reader's colour
  scheme — matches Starlight's dark/light toggle with no extra CSS.
- Lesson-detail pages gain the same section headings as the rubric, so the site and the TUI
  read identically. No divergent copy.
- Add the diagram to the catalog card hover/detail JSON only if free; not required.

---

## 5. Phases

| Phase | Work | Output | Est. |
|---|---|---|---|
| **P0 spike** | Install `d2` (pin in `mise.toml`); draw `selector-mismatch` topology; render `.txt`+`.svg`; paste ASCII into the TUI pane and check glamour doesn't re-wrap it; check 80-col terminal | go/no-go on D2 + on fenced-block passthrough | 0.5 d |
| **P1 TUI** | detail-pane markdown body, description rendering, render cache, glamour margin fixes | TUI shows full lesson prose | 1 d |
| **P2 Rubric** | Write `docs/CONTENT-STYLE.md`; build `tools/lintcontent`; run it to produce the backlog report | ranked list of the worst lessons | 1 d |
| **P3 Diagram pipeline** | `just diagrams` (generate + inject + staleness check), CI gate, `docs/public/diagrams` wiring, `gen-catalog.py` swap | one `.d2` → both targets, enforced | 1 d |
| **P4 Content wave** | Rewrite to rubric, module by module (M1 → M11), diagrams for the ~40 structural lessons; land one PR per module | 116 lessons conformant | 6–8 d |
| **P5 Site polish** | Section parity, diagram styling, spot-check dark/light | site matches TUI | 0.5 d |

P4 is the bulk and is parallel-safe: each module is an independent PR, gated by the linter.

## 6. Risks

- **D2 ASCII is alpha.** Mitigated by the ≤6-node house style and the width lint. Fallback if
  a specific diagram renders badly: hand-edit the `.txt` and mark it `# frozen` (generator
  skips regeneration but still checks width) — keeps the SVG in sync for the site while the
  terminal art stays legible.
- **Glamour re-wrapping fenced blocks** would break box art in the TUI — settled in P0, with
  a custom fence passthrough as the fallback.
- **iximiuz rendering**: the embedded ASCII is a plain fence, so nothing new is asked of the
  platform; the HTML comment markers are invisible there.
- **Scope**: 116 lessons is the long pole. The linter report should drive order — fix the
  lessons that fail the most checks first; a lesson that already conforms gets left alone.

## 7. Open decisions

1. Diagram budget: ~40 lessons, or diagram everything structural in M2/M4 only first?
2. `.txt` in-repo location — inside `unit-1.md` only (single file, generator rewrites it) vs
   `diagrams/*.txt` as separate committed artifacts (plan assumes both: file + injection).
3. Whether the TUI shows the diagram inline in the detail pane or only in the shell `task`
   output (inline assumed here).
