---
kind: lesson
title: 'Incident replay — a couple of characters pruned the world (Skyscanner)'
description: |
  Replay of Skyscanner's cited 2021 global outage: a rarely-touched config file
  lost its template delimiters, rendered to nothing, and the GitOps reconciler
  did exactly as instructed — it pruned every Service it managed, worldwide.
  Restore the config, then make the reconciler fail closed so an empty render can
  never be mistaken for "delete everything".
name: incident-gitops-prune
slug: incident-gitops-prune
source: https://medium.com/@SkyscannerEng/how-a-couple-of-characters-brought-down-our-site-356ccaf1fbc3
createdAt: "2026-07-27"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 300
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" delete svc -l app.kubernetes.io/managed-by=cells --ignore-not-found >/dev/null 2>&1 || true

      kubectl apply -n "$NS" -f - <<'YAML'
      # The per-region Service template. One "cell" per region.
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: cells-template
      data:
        service.tmpl: |
          apiVersion: v1
          kind: Service
          metadata:
            name: web-{{REGION}}
            labels:
              app.kubernetes.io/managed-by: cells
              cells.kubelings.dev/region: "{{REGION}}"
          spec:
            selector:
              app: web
            ports:
              - port: 80
                targetPort: 80
      ---
      # The regional config: root of the cells tree, changed roughly never,
      # rolled out to every cell at once.
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: cells-config
      data:
        regions: "eu us ap"
      ---
      # A frozen copy of what the bad commit rendered — kept for the check.
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: cells-config-broken
      data:
        regions: ""
      ---
      # The reconciler: render desired state, then apply it with --prune, so
      # anything it manages but no longer renders is deleted. No guard rail.
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: cells-reconciler
      data:
        run.sh: |
          set -eu
          NS="${NS:-kubelings}"
          CFG="${CONFIG_CM:-cells-config}"
          regions=$(kubectl -n "$NS" get configmap "$CFG" -o jsonpath='{.data.regions}')
          tmpl=$(kubectl -n "$NS" get configmap cells-template -o jsonpath='{.data.service\.tmpl}')
          rendered=$(mktemp)
          # An index of what this reconcile believes exists — always rendered, so
          # the apply set is never literally empty.
          cat >>"$rendered" <<EOF
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: cells-index
            labels:
              app.kubernetes.io/managed-by: cells
          data:
            regions: "$regions"
          ---
          EOF
          for r in $regions; do
            echo "$tmpl" | sed "s/{{REGION}}/$r/g" >> "$rendered"
            echo "---" >> "$rendered"
          done
          echo "rendered desired state:"
          grep -E '^(kind:|  name:)' "$rendered"
          kubectl -n "$NS" apply \
            --prune --prune-allowlist=core/v1/Service --prune-allowlist=core/v1/ConfigMap \
            -l app.kubernetes.io/managed-by=cells -f "$rendered"
      YAML

      reconcile() {   # $1 = config ConfigMap
        kubectl -n "$NS" get configmap cells-reconciler -o jsonpath='{.data.run\.sh}' > /tmp/cells-run.sh
        NS="$NS" CONFIG_CM="$1" sh /tmp/cells-run.sh
      }

      # 1) A healthy reconcile: the three regional cells exist.
      reconcile cells-config
      kubectl -n "$NS" get svc -l app.kubernetes.io/managed-by=cells

      # 2) The commit that shipped: the regions line lost its template
      #    delimiters, so it rendered to nothing — globally, immediately.
      kubectl -n "$NS" patch configmap cells-config --type=merge -p '{"data":{"regions":""}}'

      # 3) The reconciler runs on the new config and prunes what no longer renders.
      reconcile cells-config || true
      echo "--- surviving cells ---"
      kubectl -n "$NS" get svc -l app.kubernetes.io/managed-by=cells 2>&1 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      for r in eu us ap; do
        if ! kubectl -n "$NS" get svc "web-$r" >/dev/null 2>&1; then
          echo "not yet: Service web-$r is still missing — restore the region list and reconcile"; exit 1
        fi
      done
      kubectl -n "$NS" get configmap cells-reconciler -o jsonpath='{.data.run\.sh}' > /tmp/cells-verify.sh 2>/dev/null
      if [ ! -s /tmp/cells-verify.sh ]; then
        echo "not yet: ConfigMap cells-reconciler has no run.sh — keep the reconciler, guard it"; exit 1
      fi
      # 1) A valid render must still reconcile. A guard that refuses everything
      #    is not a guard, it's an outage with better manners.
      if ! NS="$NS" CONFIG_CM=cells-config sh /tmp/cells-verify.sh >/tmp/cells-good.log 2>&1; then
        echo "not yet: the reconciler now fails on the GOOD config too:"; tail -3 /tmp/cells-good.log; exit 1
      fi
      for r in eu us ap; do
        if ! kubectl -n "$NS" get svc "web-$r" >/dev/null 2>&1; then
          echo "not yet: a reconcile with the good config removed web-$r — the render must produce all three cells"; exit 1
        fi
      done
      # 2) The empty render must stop the reconciler before it deletes anything.
      if NS="$NS" CONFIG_CM=cells-config-broken sh /tmp/cells-verify.sh >/tmp/cells-bad.log 2>&1; then
        echo "not yet: the reconciler ran the EMPTY render to completion — it must fail closed. Refuse to apply when the render is missing its regions"; exit 1
      fi
      missing=""
      for r in eu us ap; do
        kubectl -n "$NS" get svc "web-$r" >/dev/null 2>&1 || missing="$missing web-$r"
      done
      if [ -n "$missing" ]; then
        echo "not yet: the empty render still pruned$missing before failing — refuse BEFORE the apply, not after"; exit 1
      fi
      echo "PASS — valid renders reconcile, empty renders stop the pipeline instead of the business. Three cells standing."
---
