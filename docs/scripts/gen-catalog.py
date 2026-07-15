#!/usr/bin/env python3
# Regenerate docs/src/data/catalog.ts from the course (source of truth).
# Run from anywhere:  python3 docs/scripts/gen-catalog.py
import os, re, glob, json

_HERE = os.path.dirname(os.path.abspath(__file__))
_REPO = os.path.normpath(os.path.join(_HERE, "..", ".."))
ROOT = os.path.join(_REPO, "courses", "kubelings")

MODULES = {
    "M1":  ("Foundations",          "#7c6af5"),
    "M2":  ("Workloads",            "#3b9eff"),
    "M3":  ("Config & Storage",     "#d29922"),
    "M4":  ("Networking",           "#4fa86d"),
    "M5":  ("Scheduling & Placement","#9b5de5"),
    "M6":  ("Security",             "#c53030"),
    "M7":  ("Internals",            "#e36f0e"),
    "M8":  ("Observability & SRE",  "#1f8a9c"),
    "M9":  ("War Stories",          "#e85d9f"),
    "M10": ("Platform Engineering", "#db6d28"),
}

def parse_frontmatter(path):
    with open(path, encoding="utf-8") as fh:
        text = fh.read()
    if not text.startswith("---"):
        return {}, text
    # frontmatter = between first '---' and the next line that is exactly '---'
    lines = text.splitlines()
    end = None
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            end = i
            break
    fm = "\n".join(lines[1:end]) if end else ""
    return fm

def get_title(fm):
    # title may be single/double quoted or plain; single-line in these files.
    m = re.search(r'(?m)^title:\s*(.+?)\s*$', fm)
    if not m:
        return None
    v = m.group(1).strip()
    if v.startswith("'") and v.endswith("'") and len(v) >= 2:
        v = v[1:-1].replace("''", "'")        # YAML single-quote unescape
    elif v.startswith('"') and v.endswith('"') and len(v) >= 2:
        v = v[1:-1].replace('\\"', '"').replace('\\\\', '\\')
    return v

def has_tasks(fm):
    return re.search(r'(?m)^tasks:\s*$', fm) is not None

entries = []
for n in range(1, 11):
    mod = f"M{n}"
    dirs = glob.glob(os.path.join(ROOT, f"module-{n}", "[0-9]*"))
    def keyf(d):
        b = os.path.basename(d)
        return int(b.split(".")[0])
    for d in sorted(dirs, key=keyf):
        idx = os.path.basename(d)
        slug = idx.split(".", 1)[1]
        f = os.path.join(d, "index.md")
        if not os.path.isfile(f):
            continue
        fm = parse_frontmatter(f)
        title = get_title(fm) or slug
        tasks = has_tasks(fm)
        if not tasks:
            typ = "read"
        elif slug.startswith("pattern-"):
            typ = "drill"
        elif slug.startswith("incident-"):
            typ = "incident"
        else:
            typ = "lab"
        entries.append({
            "module": mod, "slug": slug, "scenario": title,
            "type": typ, "iximiuz": True, "kind": tasks,
        })

# ---- emit catalog.ts ----
def esc(s):
    return json.dumps(s, ensure_ascii=False)

out = []
out.append("// src/data/catalog.ts")
out.append("// AUTO-DERIVED from courses/kubelings/ (the source of truth). 107 lessons.")
out.append("// Regenerate when lessons change; do not hand-edit entries.")
out.append("")
out.append("export type CatalogEntry = {")
out.append("  module: string;")
out.append("  slug: string;")
out.append("  scenario: string;")
out.append("  type: 'lab' | 'incident' | 'drill' | 'read';")
out.append("  iximiuz: boolean;")
out.append("  kind: boolean;")
out.append("};")
out.append("")
out.append("export const MODULES: Record<string, { label: string; color: string }> = {")
for k, (label, color) in MODULES.items():
    out.append(f"  {k+':':4} {{ label: {esc(label)}, color: '{color}' }},")
out.append("};")
out.append("")
out.append(f"export const CATALOG: CatalogEntry[] = [")
last = None
for e in entries:
    if e["module"] != last:
        last = e["module"]
        out.append(f"  // ── {e['module']} {MODULES[e['module']][0]} " + "─" * 6)
    out.append(
        "  { module:%s, slug:%s, scenario:%s, type:%s, iximiuz:%s, kind:%s },"
        % (esc(e["module"]), esc(e["slug"]), esc(e["scenario"]),
           esc(e["type"]), "true" if e["iximiuz"] else "false",
           "true" if e["kind"] else "false")
    )
out.append("];")
out.append("")

dest = os.path.join(_REPO, "docs", "src", "data", "catalog.ts")
os.makedirs(os.path.dirname(dest), exist_ok=True)
with open(dest, "w", encoding="utf-8") as fh:
    fh.write("\n".join(out))

from collections import Counter
c = Counter(e["type"] for e in entries)
print(f"wrote {len(entries)} entries -> {dest}")
print("types:", dict(c))
print("per-module:", dict(Counter(e["module"] for e in entries)))
