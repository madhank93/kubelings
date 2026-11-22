#!/usr/bin/env python3
"""Add a quick check (multiple-choice answers in a ConfigMap) to a reading lesson.

Usage: tools/add-quick-check.py <lesson-dir> <spec.json>

spec.json: {"questions": [{"q": "...", "options": ["...", "..."], "answer": "b", "why": "..."}]}

Writes init/quick_check tasks into index.md and a "Quick check" section into
unit-1.md, replacing its "*No check ...*" line if there is one.
"""
import json
import re
import sys

lesson, spec_path = sys.argv[1], sys.argv[2]
qs = json.load(open(spec_path))["questions"]
letters = "abcd"
want = " ".join(f"q{i}={q['answer']}" for i, q in enumerate(qs, 1))

tasks = f"""tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" delete configmap quick-check --ignore-not-found
  quick_check:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get configmap quick-check >/dev/null 2>&1; then
        echo "not yet: no ConfigMap 'quick-check' in namespace '$NS'"; exit 1
      fi
      wrong=""
      for kv in {want}; do
        k=${{kv%%=*}}
        got=$(kubectl -n "$NS" get configmap quick-check -o jsonpath="{{.data.$k}}" | tr -d ' )' | tr 'A-Z' 'a-z')
        [ "$got" = "${{kv#*=}}" ] || wrong="$wrong $k"
      done
      if [ -n "$wrong" ]; then
        echo "not yet: re-read the lesson and try again:$wrong"; exit 1
      fi
      echo "PASS — quick check complete."
"""

idx = f"{lesson}/index.md"
src = open(idx).read()
assert "\ntasks:" not in src, f"{idx} already has tasks"
head, sep, rest = src.partition("\n---")
after = rest.split("\n", 1)[1] if "\n" in rest else ""
open(idx, "w").write(head.rstrip("\n") + "\n" + tasks + "---\n" + after)

body = ["## Quick check", "",
        "Record your answers in a ConfigMap named `quick-check` in the `kubelings`",
        "namespace: one key per question, the letter as the value.", ""]
for i, q in enumerate(qs, 1):
    body.append(f"{i}. {q['q']}")
    body.append("")
    for l, o in zip(letters, q["options"]):
        body.append(f"   - **{l})** {o}")
    body.append("")
literals = " \\\n  ".join(f"--from-literal=q{i}=<letter>" for i in range(1, len(qs) + 1))
body += ["```sh", f"kubectl -n kubelings create configmap quick-check \\\n  {literals}", "```", "",
         "Wrong answer? `kubectl -n kubelings delete configmap quick-check` and create it again.", "",
         "::simple-task", "---", ":tasks: tasks", ":name: quick_check", "---", "#active",
         "Answer the questions above — this check turns green once every answer is right.", "",
         "#completed", "✅ Solved — nicely done!", "::", "",
         "<details>", "<summary>Answers</summary>", ""]
for i, q in enumerate(qs, 1):
    body.append(f"{i}. **{q['answer']}**: {q['why']}")
body += ["", "</details>", ""]
section = "\n".join(body)

unit = f"{lesson}/unit-1.md"
u = open(unit).read()
new, n = re.subn(r"(?m)^\*No check[^\n]*(\n[^\n]+)*?\*\s*$", lambda m: section, u, count=1)
if n == 0:
    m = re.search(r"\n<!-- d2:", u)
    new = (u[:m.start()].rstrip("\n") + "\n\n" + section + "\n" + u[m.start():]) if m else u.rstrip("\n") + "\n\n" + section + "\n"
open(unit, "w").write(new)
print(f"{lesson}: {len(qs)} questions ({want}){'' if n else ' [appended]'}")
