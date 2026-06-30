#!/usr/bin/env bash
# Scaffold a new kubelings course LESSON (course is the single source of truth).
#
# Usage:
#   tools/scaffold-lesson.sh <module-dir> <order> <name> <title> <playground>
# Example:
#   tools/scaffold-lesson.sh module-2 8 ingress "Fix the broken Ingress" k8s-omni
#
# Creates courses/kubelings/<module-dir>/<order>.<name>/{index.md,unit-1.md}.
# Edit the init/verify task `run:` blocks and the prose, then test locally with
#   scripts/run-challenge-local.sh <name> init && scripts/run-challenge-local.sh <name> verify
# and publish the whole course:
#   labctl content push course $(awk -F'\t' '$1=="kubelings-course"{print $2}' .labctl/slugs.tsv) --dir courses/kubelings --force
set -euo pipefail

MOD="${1:?module-dir, e.g. module-2}"
ORDER="${2:?order number, e.g. 8}"
NAME="${3:?lesson name, e.g. ingress}"
TITLE="${4:?title (>=10 chars)}"
PG="${5:?playground, e.g. k8s-omni}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIR="$ROOT/courses/kubelings/$MOD/$ORDER.$NAME"
[ -e "$DIR" ] && { echo "refusing to overwrite $DIR"; exit 2; }
[ -d "$ROOT/courses/kubelings/$MOD" ] || { echo "no module dir $MOD (create courses/kubelings/$MOD/0.index.md first)"; exit 2; }
mkdir -p "$DIR"
DATE="$(date +%F)"

cat >"$DIR/index.md" <<EOF
---
kind: lesson
title: "$TITLE"
description: |
  TODO one-paragraph scenario summary — what's broken/to build and what "done" means.
name: $NAME
slug: $NAME
createdAt: $DATE
playground:
  name: $PG
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "\$NS" --dry-run=client -o yaml | kubectl apply -f -
      # TODO build the broken/baseline resources here.
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # TODO assert success with plain kubectl; exit 1 (not yet) / 0 (pass).
      echo "PASS"
---
EOF

cat >"$DIR/unit-1.md" <<EOF
---
kind: unit
title: "$TITLE"
name: $NAME-unit
---

## The situation

TODO real-world framing.

## Your task

TODO numbered objectives.

<details>
<summary>Hint</summary>

TODO progressive hint(s).

</details>

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>

TODO root cause + fix commands + verification.

</details>
EOF

echo "scaffolded $DIR/{index.md,unit-1.md}"
echo "next: edit it, test with: scripts/run-challenge-local.sh $NAME init|verify"
