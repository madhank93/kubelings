#!/usr/bin/env bash
# Scaffold a new kubelings iximiuz challenge from the proven template.
#
# Usage:
#   tools/scaffold.sh <id> <title> <type> <cert> <playground> [difficulty]
# Example:
#   tools/scaffold.sh kb-wl-01 "Fix the rolling update" Fix-It CKAD k8s-omni easy
#
# Writes challenges/<id>/{index.md,solution.md,__static__/.gitkeep}. Does NOT
# publish — run tools/publish.sh <id> afterwards.
set -euo pipefail

ID="${1:?usage: scaffold.sh <id> <title> <type> <cert> <playground> [difficulty]}"
TITLE="${2:?title required}"
TYPE="${3:?type required (Fix-It|Build-It|Debug-It|Disaster-It)}"
CERT="${4:?cert required (cka|ckad|cks|ckne)}"
PG="${5:?playground required (e.g. k8s-omni)}"
DIFF="${6:-easy}"   # easy | medium | hard

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIR="$ROOT/challenges/$ID"
[ -e "$DIR" ] && { echo "refusing to overwrite existing $DIR"; exit 2; }

mkdir -p "$DIR/__static__"
touch "$DIR/__static__/.gitkeep"

cat >"$DIR/index.md" <<EOF
---
kind: challenge

title: "$TITLE"
description: |
  TODO one-paragraph scenario summary. What is broken / to build, and what
  "done" looks like. ($TYPE, $CERT)

categories:
- kubernetes

# tagz MUST NOT repeat a category word (kubernetes/networking/security/...).
tagz:
- ${CERT,,}
- workloads

difficulty: $DIFF

createdAt: $(date +%F)

playground:
  name: $PG

tasks:
  # Build the scenario before the learner sees the cluster.
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "\$NS" --dry-run=client -o yaml | kubectl apply -f -
      # TODO: create the broken/baseline resources here.

  # Passes when the learner's fix is in place. Keep verify SELF-CONTAINED:
  # plain kubectl + bash, no external sourcing (challenges must run standalone).
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # TODO: assert success with plain kubectl, e.g.:
      # desired=\$(kubectl -n "\$NS" get deploy web -o jsonpath='{.spec.replicas}')
      # avail=\$(kubectl -n "\$NS" get deploy web -o jsonpath='{.status.availableReplicas}')
      # [ "\${avail:-0}" -ge "\${desired:-1}" ] || { echo "not yet: web not ready"; exit 1; }
      echo "PASS"
---

## The situation

TODO real-world framing.

## Your task

TODO numbered objectives.

<details>
<summary>Hint</summary>

TODO progressive hint(s).

</details>
EOF

cat >"$DIR/solution.md" <<EOF
# Solution

## Root cause

TODO.

## Fix

\`\`\`sh
TODO commands
\`\`\`

## Verify

\`\`\`sh
TODO
\`\`\`
EOF

echo "scaffolded $DIR"
echo "next: edit index.md/solution.md, then tools/publish.sh $ID"
