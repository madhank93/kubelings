#!/usr/bin/env bash
# Run an iximiuz-format kubelings challenge on a local kind cluster.
#
# The init/verify logic lives ONLY in challenges/<slug>/index.md (YAML frontmatter
# tasks). This runner extracts those `run:` blocks with yq and executes them
# against your current kubectl context — the same scripts iximiuz Labs runs, so
# "works locally" and "works in the lab" stay in sync. The `machine:` field is
# iximiuz-only and ignored here.
#
# Usage:
#   scripts/run-challenge-local.sh up                 # create the kind cluster
#   scripts/run-challenge-local.sh <challenge> init   # build the scenario
#   scripts/run-challenge-local.sh <challenge> verify # run the check(s)
#   scripts/run-challenge-local.sh <challenge> reset  # wipe ns + re-init
#   scripts/run-challenge-local.sh <challenge> solution  # print solution.md
#   scripts/run-challenge-local.sh list               # list challenges
#   scripts/run-challenge-local.sh down               # delete the cluster
#
# <challenge> may be an id (kb-wl-01), a slug (kb-wl-01-53e1821a), or a dir path.
#
# Requires: kind, kubectl, yq, and a running Docker runtime (OrbStack/Docker).
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLUSTER="${KUBELINGS_CLUSTER:-kubelings}"
WORKERS="${KIND_WORKERS:-2}"     # extra worker nodes (mirrors k8s-omni's multi-node)
NS="${KUBELINGS_NS:-kubelings}"

die() { echo "error: $*" >&2; exit 2; }
for bin in kind kubectl yq; do command -v "$bin" >/dev/null || die "missing dependency: $bin"; done

# Resolve a challenge argument to its directory under challenges/.
resolve_dir() {
  local arg="$1" slug
  [ -n "$arg" ] || die "challenge required (try: $0 list)"
  if [ -d "$ROOT/challenges/$arg" ]; then echo "$ROOT/challenges/$arg"; return; fi
  if [ -d "$arg" ] && [ -f "$arg/index.md" ]; then echo "$arg"; return; fi
  slug="$(awk -F'\t' -v i="$arg" '$1==i{print $2}' "$ROOT/.labctl/slugs.tsv" 2>/dev/null)"
  if [ -n "$slug" ] && [ -d "$ROOT/challenges/$slug" ]; then echo "$ROOT/challenges/$slug"; return; fi
  local m=("$ROOT"/challenges/"$arg"*)
  if [ -d "${m[0]}" ]; then echo "${m[0]}"; return; fi
  die "no challenge matching '$arg' (try: $0 list)"
}

# Print the YAML frontmatter block of an index.md.
frontmatter() { awk '/^---$/{c++; next} c==1{print}' "$1/index.md"; }

# Run every task whose init flag matches $2 (true|false), in file order.
run_tasks() {
  local dir="$1" want_init="$2" fm rc=0
  fm="$(frontmatter "$dir")"
  local keys; keys="$(echo "$fm" | yq -r '.tasks | keys | .[]')"
  [ -n "$keys" ] || die "no tasks in $dir/index.md"
  while IFS= read -r k; do
    [ -n "$k" ] || continue
    local is_init; is_init="$(echo "$fm" | yq -r ".tasks.\"$k\".init // false")"
    [ "$is_init" = "$want_init" ] || continue
    local script; script="$(echo "$fm" | yq -r ".tasks.\"$k\".run")"
    echo "── task: $k ─────────────────────────────────────────────"
    bash -c "$script"
    local trc=$?
    if [ "$trc" -ne 0 ]; then echo "   ↳ task '$k' exited $trc"; rc=$trc; fi
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
    echo "cluster '$CLUSTER' ready ($((WORKERS+1)) node(s)). Next: $0 <challenge> init"
    ;;
  down)
    kind delete cluster --name "$CLUSTER"
    ;;
  list)
    for d in "$ROOT"/challenges/*/; do
      [ -f "$d/index.md" ] || continue
      title="$(frontmatter "$d" | yq -r '.title')"
      printf '%-26s %s\n' "$(basename "$d")" "$title"
    done
    ;;
  ""|-h|--help|help)
    print_help
    ;;
  *)
    # Per-challenge:  <challenge> <init|verify|reset|solution>  (default verify)
    CH="$A1"; VERB="${A2:-verify}"
    DIR="$(resolve_dir "$CH")"
    case "$VERB" in
      init)
        ensure_context
        run_tasks "$DIR" true
        echo; echo "scenario ready. Solve it, then: $0 $CH verify"
        ;;
      verify)
        ensure_context
        if run_tasks "$DIR" false; then
          echo; echo "✅ PASS"
        else
          echo; echo "❌ not solved yet"; exit 1
        fi
        ;;
      reset)
        ensure_context
        kubectl delete namespace "$NS" --ignore-not-found --wait=true
        run_tasks "$DIR" true
        echo; echo "scenario reset."
        ;;
      solution)
        [ -f "$DIR/solution.md" ] && cat "$DIR/solution.md" || echo "no solution.md"
        ;;
      *)
        die "unknown verb '$VERB' (use init|verify|reset|solution)"
        ;;
    esac
    ;;
esac
