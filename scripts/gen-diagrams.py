#!/usr/bin/env python3
"""Render every lesson diagram from its .d2 source, both ways.

One source of truth per diagram:

    courses/kubelings/<module>/<NN.lesson>/diagrams/<name>.d2
        -> <name>.txt   ASCII, injected into unit-1.md between
                        <!-- d2:<name> --> ... <!-- /d2:<name> --> markers
                        (the TUI, iximiuz Labs and GitHub all render it)
        -> <name>.svg   for the docs site; gen-catalog.py swaps the ASCII
                        block for this image when it writes lesson-details/

Usage:
    python3 scripts/gen-diagrams.py            # render + inject
    python3 scripts/gen-diagrams.py --check    # fail if anything is stale (CI)
"""
import os, re, sys, glob, shutil, subprocess, tempfile

_HERE = os.path.dirname(os.path.abspath(__file__))
_REPO = os.path.normpath(os.path.join(_HERE, ".."))
ROOT = os.path.join(_REPO, "courses", "kubelings")

# The TUI's right pane is ~56 columns on an 80-column terminal. D2's ASCII
# renderer has no width control, so the fix for a violation is the diagram:
# shorter labels, fewer nodes, or `direction: down`.
MAX_WIDTH = 56
# The pane is ~25 rows on a normal terminal. A diagram taller than this makes
# the learner scroll past the picture to reach the task, which is worse than a
# smaller picture: drop a node or fold it into an edge label instead.
MAX_HEIGHT = 24

check = "--check" in sys.argv
stale, rendered = [], 0

if not shutil.which("d2"):
    sys.exit("d2 not found — install it (brew install d2) or see plans/2026-07-25-content-and-diagrams.md")


def render(src, out):
    """Render src to out (extension picks the renderer). Returns file bytes."""
    subprocess.run(["d2"] + (["--theme", "0", "--dark-theme", "200", "--pad", "20"]
                            if out.endswith(".svg") else []) + [src, out],
                   check=True, capture_output=True)
    with open(out, "rb") as fh:
        return fh.read()


def measure(txt):
    lines = [l.rstrip() for l in txt.rstrip("\n").splitlines()]
    return max((len(l) for l in lines), default=0), len(lines)


def best_layout(src, name, tmp):
    """Render the diagram both ways and keep the orientation that reads best.

    D2 lays out left-to-right or top-to-bottom, and which one suits a diagram
    is a property of the diagram, not of the author's memory. Horizontal keeps
    the terminal pane short but blows past the column budget quickly; vertical
    always fits but scrolls. So: try both, discard what doesn't fit the pane,
    and prefer the shortest of what's left. A `direction:` in the source is an
    explicit author override and is left alone.
    """
    text = open(src, encoding="utf-8").read()
    if re.search(r'(?m)^\s*direction\s*:', text):
        txt = render(src, os.path.join(tmp, name + ".txt")).decode("utf-8")
        return src, txt, measure(txt), "author"

    best = None
    for d in ("right", "down"):
        cand = os.path.join(tmp, f"{name}-{d}.d2")
        with open(cand, "w", encoding="utf-8") as fh:
            fh.write(f"direction: {d}\n" + text)
        txt = render(cand, os.path.join(tmp, f"{name}-{d}.txt")).decode("utf-8")
        w, h = measure(txt)
        # Fit first, then fewest lines, then narrowest.
        key = (w > MAX_WIDTH, h, w)
        if best is None or key < best[0]:
            best = (key, cand, txt, (w, h), d)
    return best[1], best[2], best[3], best[4]


def inject(unit, name, art):
    """Put the ASCII between this diagram's markers in unit-1.md."""
    if not os.path.isfile(unit):
        return False
    body = open(unit, encoding="utf-8").read()
    pat = re.compile(r'(?ms)^(<!-- d2:%s -->\n)```text\n.*?\n```\n(<!-- /d2:%s -->)' % (name, name))
    if not pat.search(body):
        print(f"  ! {os.path.relpath(unit, _REPO)} has no <!-- d2:{name} --> markers — skipped")
        return False
    new = pat.sub(lambda m: m.group(1) + "```text\n" + art.rstrip("\n") + "\n```\n" + m.group(2), body)
    if new == body:
        return False
    open(unit, "w", encoding="utf-8").write(new)
    return True


for src in sorted(glob.glob(os.path.join(ROOT, "module-*", "*", "diagrams", "*.d2"))):
    name = os.path.basename(src)[:-3]
    ddir = os.path.dirname(src)
    lesson = os.path.dirname(ddir)
    rel = os.path.relpath(src, _REPO)

    with tempfile.TemporaryDirectory() as tmp:
        # Both outputs come from the same (possibly reoriented) source, so the
        # SVG on the site and the ASCII in the terminal show the same shape.
        chosen, txt, (width, height), direction = best_layout(src, name, tmp)
        svg = render(chosen, os.path.join(tmp, name + ".svg"))

    if width > MAX_WIDTH:
        sys.exit(f"{rel}: narrowest layout is {width} columns, limit is {MAX_WIDTH} — "
                 f"shorten labels or drop a node")
    if height > MAX_HEIGHT:
        sys.exit(f"{rel}: layout is {height} rows, limit is {MAX_HEIGHT} — "
                 f"drop a node or fold a step into an edge label")

    for path, data in ((os.path.join(ddir, name + ".txt"), txt.encode()),
                       (os.path.join(ddir, name + ".svg"), svg)):
        old = open(path, "rb").read() if os.path.isfile(path) else None
        if old == data:
            continue
        if check:
            stale.append(os.path.relpath(path, _REPO))
            continue
        with open(path, "wb") as fh:
            fh.write(data)
        rendered += 1

    unit = os.path.join(lesson, "unit-1.md")
    before = open(unit, encoding="utf-8").read() if os.path.isfile(unit) else ""
    if check:
        if f"<!-- d2:{name} -->" in before and txt.rstrip("\n") not in before:
            stale.append(os.path.relpath(unit, _REPO))
    elif inject(unit, name, txt):
        rendered += 1
    print(f"  {rel}  {width}x{height}  layout: {direction}")

if check and stale:
    sys.exit("stale diagram artifacts — run `just diagrams`:\n  " + "\n  ".join(sorted(set(stale))))
print(f"diagrams: {'up to date' if check else str(rendered) + ' file(s) written'}")
