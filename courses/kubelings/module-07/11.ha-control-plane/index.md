---
kind: lesson
title: 'HA control plane: three of everything'
description: |
  What it takes for the control plane to survive losing a node: stacked vs
  external etcd, the --control-plane-endpoint decision, joining more
  control-plane nodes, who load-balances the apiserver, and how the singleton
  controllers pick a leader. Then run the leader-election half for real — kill
  the scheduler's Lease holder and prove a new identity acquires it.
name: ha-control-plane
slug: ha-control-plane
createdAt: "2026-07-13"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      set -euo pipefail
      BASE=/etc/kubernetes/kubelings-ha-baseline
      export KUBECONFIG=/etc/kubernetes/admin.conf

      holder="$(kubectl -n kube-system get lease kube-scheduler \
        -o jsonpath='{.spec.holderIdentity}' 2>/dev/null || true)"
      if [ -z "$holder" ]; then
        echo "could not read the kube-scheduler Lease — is the control plane up?"
        exit 1
      fi

      printf 'SCHED_HOLDER=%s\n' "$holder" >"$BASE"
      chmod 600 "$BASE"

      echo "The singleton controllers elect a leader through a Lease object."
      echo "Right now the kube-scheduler Lease is held by:"
      echo
      echo "    $holder"
      echo
      kubectl -n kube-system get leases kube-scheduler kube-controller-manager 2>/dev/null || true
      echo
      echo "The holderIdentity is <node>_<uuid> — a specific *process*. Kill the"
      echo "current scheduler leader and a NEW identity has to acquire the Lease"
      echo "before scheduling resumes. Make that happen."
      echo
      echo "(baseline in $BASE — don't edit it, the check reads it)"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    run: |
      BASE=/etc/kubernetes/kubelings-ha-baseline
      export KUBECONFIG=/etc/kubernetes/admin.conf

      if [ ! -s "$BASE" ]; then
        echo "not yet: baseline file $BASE is missing — re-run init for this lesson."
        exit 1
      fi
      # shellcheck disable=SC1090
      . "$BASE"

      now_holder="$(kubectl -n kube-system get lease kube-scheduler \
        -o jsonpath='{.spec.holderIdentity}' 2>/dev/null || true)"

      if [ -z "$now_holder" ]; then
        echo "not yet: the kube-scheduler Lease has no holder right now."
        echo "         The old leader is gone but nothing has acquired it yet —"
        echo "         give the kubelet a few seconds to restart the static pod."
        exit 1
      fi

      if [ "$now_holder" = "${SCHED_HOLDER:-}" ]; then
        echo "not yet: the kube-scheduler Lease is still held by the same identity:"
        echo "           $now_holder"
        echo "         That process is still alive. Note: 'kubectl delete pod"
        echo "         kube-scheduler-cplane-01' does NOT restart it — that only"
        echo "         deletes the API mirror; the kubelet re-creates it pointing"
        echo "         at the same running container. You have to restart the"
        echo "         container, e.g. bounce its static-pod manifest:"
        echo "           mv /etc/kubernetes/manifests/kube-scheduler.yaml /tmp/ && sleep 8 \\"
        echo "             && mv /tmp/kube-scheduler.yaml /etc/kubernetes/manifests/"
        exit 1
      fi

      # A changed holder is only half the story — the new leader must actually
      # be alive and renewing, or scheduling is dead, not failed-over.
      renew="$(kubectl -n kube-system get lease kube-scheduler \
        -o jsonpath='{.spec.renewTime}' 2>/dev/null || true)"
      renew_epoch="$(date -d "$renew" +%s 2>/dev/null || echo 0)"
      now_epoch="$(date +%s)"
      if [ "$((now_epoch - renew_epoch))" -gt 60 ]; then
        echo "not yet: a new identity holds the Lease ($now_holder) but it hasn't"
        echo "         renewed in over a minute — the new leader may not be healthy."
        echo "         kubectl -n kube-system get pods | grep kube-scheduler"
        exit 1
      fi

      echo "PASS — the leader died and a new identity acquired the Lease:"
      echo "    was: ${SCHED_HOLDER}"
      echo "    now: ${now_holder}   (renewing, alive)"
      echo
      kubectl -n kube-system get leases kube-scheduler 2>/dev/null || true
---
