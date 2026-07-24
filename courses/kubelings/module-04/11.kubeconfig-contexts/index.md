---
kind: lesson
title: 'kubeconfig: contexts, the merge, and the prod you almost touched'
description: |
  A kubeconfig with prod/staging/dev contexts has current-context pointed at
  prod — one habitual `kubectl apply` away from a bad day. Switch to staging,
  then merge a teammate's kubeconfig into one flattened file without
  clobbering either. kubectl config end to end.
name: kubeconfig-contexts
slug: kubeconfig-contexts
createdAt: "2026-07-13"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      DIR=/tmp/kubelings-kubeconfig
      rm -rf "$DIR"; mkdir -p "$DIR"
      # Base: the real working credentials, minus their context name.
      kubectl config view --raw --minify > "$DIR/main"
      cluster=$(kubectl config get-clusters --kubeconfig "$DIR/main" | tail -1)
      user=$(kubectl --kubeconfig "$DIR/main" config view -o jsonpath='{.users[0].name}')
      old=$(kubectl config current-context --kubeconfig "$DIR/main")
      kubectl config rename-context "$old" prod --kubeconfig "$DIR/main"
      kubectl config set-context prod    --namespace=default   --cluster="$cluster" --user="$user" --kubeconfig "$DIR/main"
      kubectl config set-context staging --namespace=kubelings --cluster="$cluster" --user="$user" --kubeconfig "$DIR/main"
      kubectl config set-context dev     --namespace=dev       --cluster="$cluster" --user="$user" --kubeconfig "$DIR/main"
      # The hazard: everything defaults to prod.
      kubectl config use-context prod --kubeconfig "$DIR/main"
      # The teammate's file to merge in: one extra context, same cluster.
      kubectl config view --raw --minify > "$DIR/extra"
      old2=$(kubectl config current-context --kubeconfig "$DIR/extra")
      kubectl config rename-context "$old2" observability --kubeconfig "$DIR/extra"
      kubectl config set-context observability --namespace=monitoring --cluster="$cluster" --user="$user" --kubeconfig "$DIR/extra"
      echo "kubeconfigs ready under $DIR"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      DIR=/tmp/kubelings-kubeconfig
      cur=$(kubectl config current-context --kubeconfig "$DIR/main" 2>/dev/null)
      if [ "$cur" != "staging" ]; then
        echo "not yet: current-context in $DIR/main is '${cur:-unset}' — it must be staging (one habitual apply against prod is the incident)"; exit 1
      fi
      ns=$(kubectl config view --kubeconfig "$DIR/main" -o jsonpath='{.contexts[?(@.name=="staging")].context.namespace}' 2>/dev/null)
      if [ "$ns" != "kubelings" ]; then
        echo "not yet: the staging context's namespace is '${ns:-unset}', expected kubelings"; exit 1
      fi
      if [ ! -s "$DIR/merged" ]; then
        echo "not yet: no merged kubeconfig at $DIR/merged — merge main + extra with KUBECONFIG and --flatten"; exit 1
      fi
      for ctx in prod staging dev observability; do
        if ! kubectl config get-contexts "$ctx" --kubeconfig "$DIR/merged" >/dev/null 2>&1; then
          echo "not yet: merged file is missing context '$ctx' — all four must survive the merge"; exit 1
        fi
      done
      if ! kubectl --kubeconfig "$DIR/merged" --context=staging auth can-i list pods -n kubelings >/dev/null 2>&1; then
        echo "not yet: the merged file can't talk to the cluster as staging — was it flattened (--flatten), credentials and all?"; exit 1
      fi
      echo "PASS — context switched off prod, two kubeconfigs merged into one flattened, working file."
---
