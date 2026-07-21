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
NODE="${CLUSTER}-control-plane"
PROGRESS="$ROOT/.labctl/progress.tsv"

die() { echo "error: $*" >&2; exit 2; }

# SECURITY: lesson task scripts are authored content = untrusted code. They run
# ONLY inside the kind control-plane node container (isolated from the host
# filesystem/processes), using the node's in-cluster admin kubeconfig — never as
# bash on the host. Cluster lifecycle (kind create/delete) stays on the host.
_in_node() {
  docker exec -i -e KUBECONFIG=/etc/kubernetes/admin.conf "$NODE" bash -c "$1"
}

# Defense-in-depth: enforce Pod Security 'baseline' on the lesson namespace so an
# untrusted lesson manifest can't create privileged / hostPath / hostNetwork /
# hostPID pods — the pod→node→host escape vectors. Non-fatal.
_harden_ns() {
  _in_node "kubectl label namespace \"$NS\" \
    pod-security.kubernetes.io/enforce=baseline \
    pod-security.kubernetes.io/warn=baseline --overwrite >/dev/null 2>&1 || true"
}

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
for bin in kind kubectl yq docker; do command -v "$bin" >/dev/null || die "missing dependency: $bin"; done

# YAML frontmatter of a markdown file.
frontmatter() { awk '/^---$/{c++; next} c==1{print}' "$1"; }
# Does a lesson index.md declare any tasks?
has_tasks() { [ "$(frontmatter "$1" | yq -r '.tasks // {} | length')" -gt 0 ] 2>/dev/null; }

# Cloud-only lessons need real-VM/host access (systemctl, sysctl, static pod
# manifests, etcd on disk, a node reboot). Lesson scripts here are confined to
# the kind node container by design, so those lessons exist on iximiuz Labs only.
# Registry: .labctl/cloud-only.tsv (see internal/course/cloudonly.go).
CLOUD_ONLY="$ROOT/.labctl/cloud-only.tsv"
COURSE_URL="https://labs.iximiuz.com/courses/$(
  awk -F'\t' '$1=="kubelings-course"{print $2; exit}' "$ROOT/.labctl/slugs.tsv" 2>/dev/null \
    || true)"
[ "$COURSE_URL" = "https://labs.iximiuz.com/courses/" ] \
  && COURSE_URL="https://labs.iximiuz.com/courses/kubelings-dbd840c8"

# Reason string for a cloud-only lesson; non-zero if the lesson isn't in the registry.
cloud_only_reason() {
  [ -f "$CLOUD_ONLY" ] || return 1
  awk -F'\t' -v l="$1" '
    /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
    $1==l { i=index($0,"\t"); print substr($0,i+1); found=1; exit }
    END { exit !found }' "$CLOUD_ONLY"
}

# is_cloud_only <lesson-name> <index.md> — registry, or the cloudOnly frontmatter fallback.
is_cloud_only() {
  cloud_only_reason "$1" >/dev/null 2>&1 && return 0
  [ -n "${2:-}" ] && [ "$(frontmatter "$2" | yq -r '.cloudOnly // false')" = "true" ]
}

# SECURITY: only ever resolve to an index.md that lives UNDER courses/kubelings.
# Rejects path traversal / symlinks pointing outside, so the runner can't be
# tricked into executing an arbitrary file's task blocks.
_confine() {
  local p real course_real
  real="$(cd "$(dirname "$1")" 2>/dev/null && pwd -P)/$(basename "$1")" || die "bad path"
  course_real="$(cd "$COURSE" && pwd -P)"
  case "$real" in
    "$course_real"/*) echo "$real" ;;
    *) die "refusing to run a lesson outside $COURSE" ;;
  esac
}

# Resolve <lesson> to its index.md path (confined to the course tree).
resolve_lesson() {
  local arg="$1"
  [ -n "$arg" ] || die "lesson required (try: $0 list)"
  [ -f "$arg/index.md" ] && { _confine "$arg/index.md"; return; }
  [ -f "$arg" ] && [ "$(basename "$arg")" = "index.md" ] && { _confine "$arg"; return; }
  local d base nm name slug
  for d in "$COURSE"/module-*/*/; do
    [ -f "$d/index.md" ] || continue
    base="$(basename "$d")"; nm="${base#*.}"     # strip leading "N."
    name="$(frontmatter "$d/index.md" | yq -r '.name // ""')"
    slug="$(frontmatter "$d/index.md" | yq -r '.slug // ""')"
    if [ "$arg" = "$nm" ] || [ "$arg" = "$name" ] || [ "$arg" = "$slug" ]; then
      _confine "$d/index.md"; return
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
    _in_node "$script"; trc=$?
    [ "$trc" -ne 0 ] && { echo "   ↳ task '$k' exited $trc"; rc=$trc; }
  done <<<"$keys"
  return $rc
}

# Confine all cluster work to the kind node. Never touches the host's kubeconfig
# or any other cluster — so a stray KUBECONFIG pointing at prod can't be affected.
ensure_node() {
  kind get clusters 2>/dev/null | grep -qx "$CLUSTER" \
    || die "kind cluster '$CLUSTER' not found — run: $0 up"
  [ "$(docker inspect -f '{{.State.Running}}' "$NODE" 2>/dev/null)" = "true" ] \
    || die "node '$NODE' not running — run: $0 up"
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
      # Cloud-only lessons are listed, not hidden: a user reads the slug on
      # /catalog and types it into `just run`. Silent omission reads as a broken
      # checkout, so make `list` the place the constraint is learned.
      if is_cloud_only "$nm" "$d/index.md"; then
        printf '%-18s %s  [iximiuz-only]\n' "$nm" "$title"
        any_cloud=1
      else
        printf '%-18s %s\n' "$nm" "$title"
      fi
    done
    if [ -n "${any_cloud:-}" ]; then
      echo
      echo "[iximiuz-only] = needs real VMs; runs at $COURSE_URL, not on local kind."
    fi
    ;;
  progress)
    [ -f "$PROGRESS" ] && cat "$PROGRESS" || true
    ;;
  ""|-h|--help|help)
    print_help
    ;;
  *)
    LESSON="$A1"; VERB="${A2:-verify}"
    IDX="$(resolve_lesson "$LESSON")"
    [ -n "$IDX" ] && [ -f "$IDX" ] || die "could not resolve lesson '$LESSON' (rejected or not found)"
    LDIR="$(dirname "$IDX")"; LNAME="$(_lesson_name "$IDX")"

    # Refuse cloud-only lessons before any verb runs, so ensure_node, run_tasks,
    # _in_node, _harden_ns and _set_progress are all unreachable for them.
    # Placed after _confine/resolve_lesson: path confinement stays first.
    if is_cloud_only "$LNAME" "$IDX"; then
      REASON="$(cloud_only_reason "$LNAME" 2>/dev/null)" || REASON="it needs real-VM/host access"
      [ -n "$REASON" ] || REASON="it needs real-VM/host access"
      {
        echo "☁ '$LNAME' runs on iximiuz Labs only."
        echo "  Not on local kind, because $REASON."
        echo "  Lesson scripts are confined to the kind node container, so host-level"
        echo "  work has nowhere to happen locally. On iximiuz it gets real VMs."
        echo "  Run it: $COURSE_URL"
      } >&2
      case "$VERB" in
        # Nothing was going to happen; not a user error.
        init|reset) exit 0 ;;
        # 3 = "this check does not exist here", distinct from 0=pass, 1=not solved.
        verify) exit 3 ;;
        # Static markdown: executes nothing, and this is what the user most wants.
        solution) echo >&2; for u in "$LDIR"/unit-*.md; do [ -f "$u" ] && cat "$u"; done; exit 0 ;;
        *) die "unknown verb '$VERB' (use init|verify|reset|solution)" ;;
      esac
    fi

    case "$VERB" in
      init)
        ensure_node; run_tasks "$IDX" true; _harden_ns
        _set_progress "$LNAME" started
        echo; echo "scenario ready. Solve it, then: $0 $LESSON verify"
        ;;
      verify)
        # A reading has no checks. Without this, run_tasks prints
        # "(content-only lesson)", returns 0, and the reading is marked solved.
        has_tasks "$IDX" || {
          echo "'$LNAME' is a reading — it has no check to run."; exit 0; }
        ensure_node
        if run_tasks "$IDX" false; then
          _set_progress "$LNAME" solved; echo; echo "✅ PASS"
        else
          echo; echo "❌ not solved yet"; exit 1
        fi
        ;;
      reset)
        ensure_node
        _in_node "kubectl delete namespace \"$NS\" --ignore-not-found --wait=true"
        run_tasks "$IDX" true; _harden_ns
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
