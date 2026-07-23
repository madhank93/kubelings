#!/usr/bin/env python3
# Regenerate docs/src/data/catalog.ts from the course (source of truth).
# Run from anywhere:  python3 docs/scripts/gen-catalog.py
import os, re, glob, json

_HERE = os.path.dirname(os.path.abspath(__file__))
_REPO = os.path.normpath(os.path.join(_HERE, "..", ".."))
ROOT = os.path.join(_REPO, "courses", "kubelings")

# label, colour, and the skill narrative (absorbed from the retired
# /guides/curriculum page — the catalog module rows are now its only home).
MODULES = {
    "M1":  ("Foundations",          "#7c6af5",
            "pods, Deployments, Services, namespaces, labels & selectors, the triage loop (`describe` → `logs` → fix → watch)"),
    "M2":  ("Workloads",            "#3b9eff",
            "rolling updates, blue/green & canary, DaemonSets, StatefulSets, Jobs, CronJobs, HPA, OOMKill & right-sizing, CPU throttling, probes, init containers, PDBs, QoS, ephemeral-container debugging, multi-container patterns, readiness/CronJob/rollout failure drills, VPA, KEDA"),
    "M3":  ("Config & Storage",     "#d29922",
            "ConfigMaps, Secrets, PV/PVC lifecycle, StorageClasses, access modes, finalizer traps, kustomize, Helm release lifecycle, ghost-endpoint & secret-rotation & stuck-namespace drills"),
    "M4":  ("Networking",           "#4fa86d",
            "Services & endpoints, Ingress & Gateway API, NetworkPolicy, CoreDNS & the ndots amplifier, kube-proxy dataplane, CNI anatomy & triage, conntrack, graceful shutdown, kubeconfig contexts"),
    "M5":  ("Scheduling & Placement","#9b5de5",
            "affinity/anti-affinity, taints & tolerations, topology spread, priority & preemption, noisy neighbors"),
    "M6":  ("Security",             "#c53030",
            "RBAC, ServiceAccounts & tokens, Pod Security, admission webhooks, container hardening, CIS benchmarks, egress lockdown, image digests, Gatekeeper & Kyverno policy engines, trivy scanning, cosign signatures & SBOMs, seccomp/AppArmor, encryption-at-rest, audit policy, Falco runtime detection"),
    "M7":  ("Internals",            "#e36f0e",
            "API server request & admission flow, watch/informers & APF, etcd (incl. backup/restore), CRDs & building operators, scheduler internals, controller reconciliation, kubelet ↔ CRI, leader election, kubeadm bootstrap, HA control planes, certificate rotation"),
    "M8":  ("Observability & SRE",  "#1f8a9c",
            "events forensics, node NotReady triage, quotas, disk pressure & eviction, cluster upgrades, node maintenance, SLO burn-rate alerting, OTel tracing pipelines, debugging playbooks"),
    "M9":  ("War Stories",          "#e85d9f",
            "multi-concept cascade incidents from cited postmortems — everything at once, then the final boss"),
    "M10": ("Platform Engineering", "#db6d28",
            "GitOps with Argo CD (incl. app-of-apps) and Flux, multi-tenancy with Capsule, Cluster API, Crossplane compositions"),
    "M11": ("Node & Control Plane", "#6b8299",
            "node and control-plane failures that need a real machine, not a container — kubelet and containerd outages, a crash-looped kube-apiserver, etcd NOSPACE compact/defrag, authoring static pods, approving node CSRs, cgroup-driver mismatch, clock skew, and the kernel sysctls/modules pod networking depends on (iximiuz Labs only)"),
}

# ---- cloud-only registry (.labctl/cloud-only.tsv) ----
# Lessons whose tasks need real-VM/host access. The local kind runner confines
# lesson scripts to the node container, so these run on iximiuz Labs only.
# Kept out of lesson frontmatter because `labctl content push` 400s on unknown
# keys — see internal/course/cloudonly.go for the same parse in Go.
CLOUD_ONLY_TSV = os.path.join(_REPO, ".labctl", "cloud-only.tsv")

def load_cloud_only():
    out = {}
    if not os.path.isfile(CLOUD_ONLY_TSV):
        return out
    with open(CLOUD_ONLY_TSV, encoding="utf-8") as fh:
        for line in fh:
            line = line.rstrip("\n")
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            name, _, reason = line.partition("\t")
            out[name.strip()] = reason.strip()
    return out

CLOUD_ONLY = load_cloud_only()

def is_cloud_only(fm, slug):
    if slug in CLOUD_ONLY:
        return True
    return re.search(r'(?m)^cloudOnly:\s*true\s*$', fm) is not None

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

# ---- lesson detail (the "problem" prose shown in the catalog modal) ----
# Source: each lesson's unit-1.md. For hands-on lessons we stop at the first
# hint/solution so the modal can't spoil the exercise; readings have no task to
# spoil, so they carry through in full.
DETAILS_DIR = os.path.join(_REPO, "docs", "src", "data", "lesson-details")

def strip_frontmatter(text):
    if not text.startswith("---"):
        return text
    lines = text.splitlines()
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            return "\n".join(lines[i + 1:])
    return text

def extract_detail(unit_path, hands_on):
    if not os.path.isfile(unit_path):
        return None
    body = strip_frontmatter(open(unit_path, encoding="utf-8").read())
    # Drop pointers to the old /incidents/* pages — the modal renders that
    # write-up directly beneath this prose, so the link would point at itself.
    body = re.sub(r'(?m)^>.*write-up:.*(?:\n>.*)*\n?', '', body)
    body = re.sub(r'\s*·?\s*\[[^\]]*\]\(https://kubelings\.madhan\.app/incidents/[^)]*\)', '', body)
    if hands_on:
        # Cut at the first hint block or task directive: everything before it is
        # situation + task, everything after is hint/solution/check plumbing.
        cuts = [m.start() for m in re.finditer(r'(?m)^(?:<details>|::[a-z-]+)', body) if m]
        if cuts:
            body = body[:min(cuts)]
    else:
        # Readings: keep the prose, drop only the iximiuz task directives.
        body = re.sub(r'(?ms)^::simple-task.*?^::\s*$', '', body)
    body = re.sub(r'\n{3,}', '\n\n', body).strip()
    return body or None

# Regenerated wholesale — clear stale details from renamed/removed lessons.
for stale in glob.glob(os.path.join(DETAILS_DIR, "*.md")):
    os.remove(stale)

# Modules come from disk, not a hardcoded range, so adding module-11 is a
# directory + a MODULES entry. Fail loudly on a module with no MODULES entry —
# otherwise the separator emitter below dies on a bare KeyError.
_module_nums = sorted(
    int(re.match(r'module-(\d+)$', os.path.basename(d)).group(1))
    for d in glob.glob(os.path.join(ROOT, "module-*"))
    if os.path.isdir(d) and re.match(r'module-\d+$', os.path.basename(d))
)
for _n in _module_nums:
    if f"M{_n}" not in MODULES:
        raise SystemExit(
            f"courses/kubelings/module-{_n}/ exists but M{_n} has no MODULES entry "
            f"in {os.path.relpath(__file__, _REPO)} — add its label, colour, and narrative."
        )

entries = []
for n in _module_nums:
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
        # A task-less lesson has nothing to run anywhere, so it stays a reading
        # rather than becoming "cloud-only".
        cloud = tasks and is_cloud_only(fm, slug)
        if not tasks:
            typ = "read"
        elif slug.startswith("pattern-"):
            typ = "drill"
        elif slug.startswith("incident-"):
            typ = "incident"
        else:
            typ = "lab"
        inc = INCIDENTS.get(slug)

        # The long-form write-up (formerly a page under /incidents/) now renders
        # inside the modal. Company incidents map via INCIDENTS; pattern drills
        # share their slug with the write-up filename.
        def write_up_path(name):
            return os.path.join(_REPO, "docs", "src", "data", "incidents", name + ".md")

        write_up = None
        if inc and inc[2]:
            # A cited incident that claims a write-up must have one.
            write_up = inc[2].strip("/").split("/")[-1]
            if not os.path.isfile(write_up_path(write_up)):
                raise SystemExit(f"{slug}: write-up '{write_up}.md' not found in src/data/incidents/")
        elif slug.startswith("pattern-") and os.path.isfile(write_up_path(slug)):
            # Only some pattern drills have a long-form write-up.
            write_up = slug

        detail = extract_detail(os.path.join(d, "unit-1.md"), tasks)
        if detail:
            os.makedirs(DETAILS_DIR, exist_ok=True)
            with open(os.path.join(DETAILS_DIR, slug + ".md"), "w", encoding="utf-8") as fh:
                fh.write(detail + "\n")

        entries.append({
            "module": mod, "slug": slug, "scenario": title,
            # iximiuz + kind are *execution* platforms: only hands-on lessons
            # run there. Read-only lessons are runbooks — nothing to execute — so
            # they carry neither platform tag (the table shows them as "runbook").
            # The three are independent: a cloud-only lesson is hands-on and runs
            # on iximiuz, but not on kind.
            "type": typ, "iximiuz": tasks, "kind": tasks and not cloud,
            "handsOn": tasks, "cloudOnly": cloud, "description": desc,
            "real": inc is not None,
            "company": inc[0] if inc else None,
            "source": inc[1] if inc else None,
            "writeUp": write_up,
            "detail": bool(detail),
        })

# ---- validate the cloud-only registry ----
# The registry does not sit next to the lessons it names, so nothing else would
# catch a typo, a rename, or a row pointing at a lesson with no tasks to run.
_by_slug = {e["slug"]: e for e in entries}
_errs = []
for _name, _reason in sorted(CLOUD_ONLY.items()):
    e = _by_slug.get(_name)
    if e is None:
        _errs.append(f"  {_name}: no such lesson under courses/kubelings/")
    elif not e["handsOn"]:
        _errs.append(f"  {_name}: has no tasks — a reading is already unrunnable everywhere; "
                     f"drop the row instead")
    if not _reason:
        _errs.append(f"  {_name}: no reason — add a tab and a sentence completing "
                     f"\"it can't run locally because …\"")
if _errs:
    raise SystemExit(".labctl/cloud-only.tsv is invalid:\n" + "\n".join(_errs))

# ---- emit catalog.ts ----
def esc(s):
    return json.dumps(s, ensure_ascii=False)

out = []
out.append("// src/data/catalog.ts")
out.append(f"// AUTO-DERIVED from courses/kubelings/ (the source of truth). {len(entries)} lessons.")
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
out.append("  cloudOnly?: boolean;        // needs real VMs — iximiuz Labs only, never local kind")
out.append("  description: string;")
out.append("  real: boolean;              // reproduces a cited, real company incident")
out.append("  company?: string;")
out.append("  source?: string;            // link to the public postmortem")
out.append("  writeUp?: string;           // src/data/incidents/<name>.md, rendered in the modal")
out.append("  detail: boolean;            // has src/data/lesson-details/<slug>.md")
out.append("};")
out.append("")
out.append("export const MODULES: Record<string, { label: string; color: string; learn: string }> = {")
for k, (label, color, learn) in MODULES.items():
    out.append(f"  {k+':':4} {{ label: {esc(label)}, color: '{color}', learn: {esc(learn)} }},")
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
    ]
    # Emitted only when true, so existing rows don't churn.
    if e["cloudOnly"]:
        parts.append("cloudOnly:true")
    parts += [
        "description:%s" % esc(e["description"]),
        "real:%s" % ("true" if e["real"] else "false"),
    ]
    if e["company"]:
        parts.append("company:%s" % esc(e["company"]))
    if e["source"]:
        parts.append("source:%s" % esc(e["source"]))
    if e["writeUp"]:
        parts.append("writeUp:%s" % esc(e["writeUp"]))
    parts.append("detail:%s" % ("true" if e["detail"] else "false"))
    out.append("  { " + ", ".join(parts) + " },")
out.append("];")
out.append("")

dest = os.path.join(_REPO, "docs", "src", "data", "catalog.ts")
os.makedirs(os.path.dirname(dest), exist_ok=True)
with open(dest, "w", encoding="utf-8") as fh:
    fh.write("\n".join(out))

# ---- emit incident-redirects.json ----
# The /incidents/<name>/ pages are retired; their write-ups render in the catalog
# modal. Lesson prose published on iximiuz still links the old absolute URLs, so
# every write-up keeps a redirect to the row that now carries it. Consumed by
# astro.config.mjs.
by_write_up = {}
for e in entries:
    if e["writeUp"]:
        by_write_up[e["writeUp"]] = e["slug"]

redirects = {}
for f in sorted(glob.glob(os.path.join(_REPO, "docs", "src", "data", "incidents", "*.md"))):
    name = os.path.basename(f)[:-3]
    # Write-ups with no lesson (further-reading only) open under their own name.
    redirects[f"/incidents/{name}/"] = f"/catalog/?lesson={by_write_up.get(name, name)}"

rdest = os.path.join(_REPO, "docs", "src", "data", "incident-redirects.json")
with open(rdest, "w", encoding="utf-8") as fh:
    json.dump(redirects, fh, indent=2, ensure_ascii=False)
    fh.write("\n")

from collections import Counter
c = Counter(e["type"] for e in entries)
print(f"wrote {len(entries)} entries -> {dest}")
print("types:", dict(c))
print("per-module:", dict(Counter(e["module"] for e in entries)))
print(f"details: {sum(1 for e in entries if e['detail'])}/{len(entries)} -> {DETAILS_DIR}")
print(f"write-ups linked: {sum(1 for e in entries if e['writeUp'])}")
