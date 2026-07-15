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

def get_description(fm):
    lines = fm.splitlines()
    for i, ln in enumerate(lines):
        m = re.match(r'^description:\s*(.*)$', ln)
        if not m:
            continue
        val = m.group(1).strip()
        if val in ('|', '|-', '|+', '>', '>-', '>+', ''):
            # block scalar: gather following indented lines
            buf = []
            for nxt in lines[i + 1:]:
                if nxt.strip() == '':
                    buf.append('')
                    continue
                if nxt[:1] in (' ', '\t'):
                    buf.append(nxt.strip())
                else:
                    break
            return ' '.join(x for x in buf if x != '').strip()
        if (val.startswith('"') and val.endswith('"')) or (val.startswith("'") and val.endswith("'")):
            q = val[0]
            val = val[1:-1].replace(q + q, q) if q == "'" else val[1:-1].replace('\\"', '"')
        return val.strip()
    return ''

# Real, cited company incidents reproduced or documented as lessons.
# slug -> (company, source_url, case_study_path_or_None)
INCIDENTS = {
    "incident-cpu-throttling":     ("Omio",         "https://medium.com/omio-engineering/cpu-limits-and-aggressive-throttling-in-kubernetes-c5b20bd8a718", None),
    "incident-dns-ndots":          ("Zalando",      "https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jan-2019-dns-outage.md", "/incidents/zalando-dns-outage/"),
    "incident-graceful-shutdown":  ("Ravelin",      "https://philpearl.github.io/post/k8s_ingress/", "/incidents/ravelin-graceful-shutdown/"),
    "incident-same-node":          ("Moonlight",    "https://updates.moonlightwork.com/outage-post-mortem-87370", "/incidents/moonlight-same-node/"),
    "incident-priority-preemption":("Grafana Labs", "https://grafana.com/blog/2019/07/24/how-a-production-outage-was-caused-using-kubernetes-pod-priorities/", "/incidents/grafana-priority-preemption/"),
    "incident-cryptominer":        ("JW Player",    "https://medium.com/jw-player-engineering/how-a-cryptocurrency-miner-made-its-way-onto-our-internal-kubernetes-clusters-9b09c4704205", "/incidents/jwplayer-cryptominer/"),
    "incident-webhook-outage":     ("Jetstack",     "https://blog.jetstack.io/blog/gke-webhook-outage", "/incidents/jetstack-webhook-outage/"),
    "incident-node-oom":           ("Blue Matador", "https://www.bluematador.com/blog/post-mortem-kubernetes-node-oom", "/incidents/bluematador-node-oom/"),
    "incident-conntrack":          ("loveholidays", "https://deploy.live/blog/kubernetes-networking-problems-due-to-the-conntrack/", "/incidents/conntrack-exhaustion/"),
    "incident-datadog-cilium":     ("Datadog",      "https://www.datadoghq.com/blog/2023-03-08-multiregion-infrastructure-connectivity-issue/", "/incidents/datadog-cilium-routes/"),
    "incident-monzo-cascade":      ("Monzo",        "https://community.monzo.com/t/resolved-current-account-payments-may-fail-major-outage-27-10-2017/26296/95", "/incidents/monzo-cascade/"),
    "incident-openai-cascade":     ("OpenAI",       "https://status.openai.com/incidents/ctrsv3lwd797", "/incidents/openai-telemetry-cascade/"),
    "incident-reddit-piday":       ("Reddit",       "https://www.reddit.com/r/RedditEng/comments/11xx5o0/you_broke_reddit_the_piday_outage/", "/incidents/reddit-piday/"),
    "incident-black-friday":       ("Algolia",      "https://www.youtube.com/watch?v=Fjyg7cxRZQs", "/incidents/algolia-black-friday/"),
    "incident-target-cascade":     ("Target",       "https://medium.com/@daniel.p.woods/on-infrastructure-at-scale-a-cascading-failure-of-distributed-systems-7cff2a3cd2df", "/incidents/target-cascade/"),
    "incident-spotify-delete":     ("Spotify",      "https://www.youtube.com/watch?v=ix0Tw8uinWs", "/incidents/spotify-delete/"),
}

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
        desc = get_description(fm)
        tasks = has_tasks(fm)
        if not tasks:
            typ = "read"
        elif slug.startswith("pattern-"):
            typ = "drill"
        elif slug.startswith("incident-"):
            typ = "incident"
        else:
            typ = "lab"
        inc = INCIDENTS.get(slug)
        entries.append({
            "module": mod, "slug": slug, "scenario": title,
            "type": typ, "iximiuz": True, "kind": tasks,
            "handsOn": tasks, "description": desc,
            "real": inc is not None,
            "company": inc[0] if inc else None,
            "source": inc[1] if inc else None,
            "caseStudy": inc[2] if inc else None,
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
out.append("  handsOn: boolean;")
out.append("  description: string;")
out.append("  real: boolean;              // reproduces a cited, real company incident")
out.append("  company?: string;")
out.append("  source?: string;            // link to the public postmortem")
out.append("  caseStudy?: string;         // local /incidents/* write-up, if any")
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
    parts = [
        "module:%s" % esc(e["module"]),
        "slug:%s" % esc(e["slug"]),
        "scenario:%s" % esc(e["scenario"]),
        "type:%s" % esc(e["type"]),
        "iximiuz:%s" % ("true" if e["iximiuz"] else "false"),
        "kind:%s" % ("true" if e["kind"] else "false"),
        "handsOn:%s" % ("true" if e["handsOn"] else "false"),
        "description:%s" % esc(e["description"]),
        "real:%s" % ("true" if e["real"] else "false"),
    ]
    if e["company"]:
        parts.append("company:%s" % esc(e["company"]))
    if e["source"]:
        parts.append("source:%s" % esc(e["source"]))
    if e["caseStudy"]:
        parts.append("caseStudy:%s" % esc(e["caseStudy"]))
    out.append("  { " + ", ".join(parts) + " },")
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
