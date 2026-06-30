#!/usr/bin/env bash
# Run a kubelings course lesson on a local kind cluster.
#
# The course (courses/kubelings/) is the single source of truth. Each lesson's
# index.md frontmatter carries the init/verify `tasks:` — this runner extracts
# those `run:` blocks with yq and executes them against your current kube-context,
# so "works locally" and "works on iximiuz Labs" stay in sync. The iximiuz-only
# `machine:` field is ignored locally.
#
# Usage:
#   scripts/run-challenge-local.sh up                 # create the kind cluster
#   scripts/run-challenge-local.sh list               # list runnable lessons
#   scripts/run-challenge-local.sh <lesson> init      # build the scenario
#   scripts/run-challenge-local.sh <lesson> verify    # run the check(s)
#   scripts/run-challenge-local.sh <lesson> reset     # wipe ns + re-init
#   scripts/run-challenge-local.sh <lesson> solution  # print the lesson content
#   scripts/run-challenge-local.sh down               # delete the cluster
#
# <lesson> may be a lesson name (e.g. rolling-update), its slug, or a dir path.
#
# Requires: kind, kubectl, yq, and a running Docker runtime (OrbStack/Docker).
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COURSE="$ROOT/courses/kubelings"
CLUSTER="${KUBELINGS_CLUSTER:-kubelings}"
WORKERS="${KIND_WORKERS:-2}"
NS="${KUBELINGS_NS:-kubelings}"
PROGRESS="$ROOT/.labctl/progress.tsv"

die() { echo "error: $*" >&2; exit 2; }

# Persist per-lesson progress (last write wins): <lesson>\t<none|started|solved>\t<epoch>
_set_progress() {
  local l="$1" s="$2" tmp; tmp="$(mktemp)"
  mkdir -p "$(dirname "$PROGRESS")"; touch "$PROGRESS"
  awk -F'\t' -v l="$l" '$1!=l' "$PROGRESS" > "$tmp"
  printf '%s\t%s\t%s\n' "$l" "$s" "$(date +%s)" >> "$tmp"
  mv "$tmp" "$PROGRESS"
}
# Canonical lesson name from a resolved index.md path (dir basename minus "N.").
_lesson_name() { local b; b="$(basename "$(dirname "$1")")"; echo "${b#*.}"; }
for bin in kind kubectl yq; do command -v "$bin" >/dev/null || die "missing dependency: $bin"; done

# YAML frontmatter of a markdown file.
frontmatter() { awk '/^---$/{c++; next} c==1{print}' "$1"; }
# Does a lesson index.md declare any tasks?
has_tasks() { [ "$(frontmatter "$1" | yq -r '.tasks // {} | length')" -gt 0 ] 2>/dev/null; }

# Resolve <lesson> to its index.md path.
resolve_lesson() {
  local arg="$1"
  [ -n "$arg" ] || die "lesson required (try: $0 list)"
  [ -f "$arg/index.md" ] && { echo "$arg/index.md"; return; }
  [ -f "$arg" ] && { echo "$arg"; return; }
  local d base nm name slug
  for d in "$COURSE"/module-*/*/; do
    [ -f "$d/index.md" ] || continue
    base="$(basename "$d")"; nm="${base#*.}"     # strip leading "N."
    name="$(frontmatter "$d/index.md" | yq -r '.name // ""')"
    slug="$(frontmatter "$d/index.md" | yq -r '.slug // ""')"
    if [ "$arg" = "$nm" ] || [ "$arg" = "$name" ] || [ "$arg" = "$slug" ]; then
      echo "$d/index.md"; return
    fi
  done
  die "no lesson matching '$arg' (try: $0 list)"
}

# Run every task whose init flag matches $2 (true|false), in file order.
run_tasks() {
  local idx="$1" want_init="$2" fm keys rc=0
  fm="$(frontmatter "$idx")"
  keys="$(echo "$fm" | yq -r '.tasks // {} | keys | .[]')"
  [ -n "$keys" ] || { echo "(content-only lesson — no tasks to run)"; return 0; }
  while IFS= read -r k; do
    [ -n "$k" ] || continue
    local is_init script trc
    is_init="$(echo "$fm" | yq -r ".tasks.\"$k\".init // false")"
    [ "$is_init" = "$want_init" ] || continue
    script="$(echo "$fm" | yq -r ".tasks.\"$k\".run")"
    echo "── task: $k ─────────────────────────────────────────────"
    bash -c "$script"; trc=$?
    [ "$trc" -ne 0 ] && { echo "   ↳ task '$k' exited $trc"; rc=$trc; }
  done <<<"$keys"
  return $rc
}

ensure_context() {
  kind get clusters 2>/dev/null | grep -qx "$CLUSTER" \
    || die "kind cluster '$CLUSTER' not found — run: $0 up"
  kubectl config use-context "kind-$CLUSTER" >/dev/null
}

print_help() { grep '^#' "$0" | sed 's/^# \{0,1\}//' | sed -n '1,30p'; }

A1="${1:-}"; A2="${2:-}"

case "$A1" in
  up)
    if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
      echo "cluster '$CLUSTER' already exists."
    else
      cfg="$(mktemp)"
      { echo "kind: Cluster"; echo "apiVersion: kind.x-k8s.io/v1alpha4"; echo "nodes:";
        echo "  - role: control-plane";
        for _ in $(seq 1 "$WORKERS"); do echo "  - role: worker"; done; } >"$cfg"
      kind create cluster --name "$CLUSTER" --config "$cfg" || die "kind create failed"
      rm -f "$cfg"
    fi
    kubectl config use-context "kind-$CLUSTER" >/dev/null
    kubectl wait --for=condition=Ready nodes --all --timeout=120s || true
    echo "cluster '$CLUSTER' ready ($((WORKERS+1)) node(s)). Next: $0 <lesson> init"
    ;;
  down)
    kind delete cluster --name "$CLUSTER"
    ;;
  list)
    for d in "$COURSE"/module-*/*/; do
      [ -f "$d/index.md" ] || continue
      has_tasks "$d/index.md" || continue
      nm="$(basename "$d")"; nm="${nm#*.}"
      title="$(frontmatter "$d/index.md" | yq -r '.title')"
      printf '%-18s %s\n' "$nm" "$title"
    done
    ;;
  progress)
    [ -f "$PROGRESS" ] && cat "$PROGRESS" || true
    ;;
  ""|-h|--help|help)
    print_help
    ;;
  *)
    LESSON="$A1"; VERB="${A2:-verify}"
    IDX="$(resolve_lesson "$LESSON")"; LDIR="$(dirname "$IDX")"; LNAME="$(_lesson_name "$IDX")"
    case "$VERB" in
      init)
        ensure_context; run_tasks "$IDX" true
        _set_progress "$LNAME" started
        echo; echo "scenario ready. Solve it, then: $0 $LESSON verify"
        ;;
      verify)
        ensure_context
        if run_tasks "$IDX" false; then
          _set_progress "$LNAME" solved; echo; echo "✅ PASS"
        else
          echo; echo "❌ not solved yet"; exit 1
        fi
        ;;
      reset)
        ensure_context
        kubectl delete namespace "$NS" --ignore-not-found --wait=true
        run_tasks "$IDX" true
        _set_progress "$LNAME" started
        echo; echo "scenario reset."
        ;;
      solution)
        for u in "$LDIR"/unit-*.md; do [ -f "$u" ] && cat "$u"; done
        ;;
      *)
        die "unknown verb '$VERB' (use init|verify|reset|solution)"
        ;;
    esac
    ;;
esac
