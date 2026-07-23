---
kind: lesson
title: 'kubelet down: a node goes NotReady from the host up'
description: |
  A worker's kubelet won't start — the node has gone NotReady and its pods are
  stranded. This is the failure M8's node-notready triage points at but can't
  reach: someone left an invalid flag in the kubelet's extra-args file, not
  anything the API can see. Diagnose it with journalctl and systemctl the way
  you would at 3am, clear the bad config, and bring the node back Ready.
name: kubelet-down
slug: kubelet-down
createdAt: "2026-07-23"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: node-01
    user: root
    timeout_seconds: 180
    run: |
      set -euo pipefail
      DEF=/etc/default/kubelet
      BASE=/root/kubelings-kubelet-def.bak

      # Plant the fault a real operator leaves behind: an invalid flag in the
      # kubelet's extra-args file. kubeadm splices $KUBELET_EXTRA_ARGS (sourced
      # from here) into the kubelet ExecStart, so it now dies at startup with
      # "unknown flag" and systemd holds it in a crash loop.
      cp "$DEF" "$BASE"
      if grep -q '^KUBELET_EXTRA_ARGS=' "$DEF"; then
        sed -i 's|^KUBELET_EXTRA_ARGS=.*|KUBELET_EXTRA_ARGS=--kubelings-nonexistent-flag=1|' "$DEF"
      else
        echo 'KUBELET_EXTRA_ARGS=--kubelings-nonexistent-flag=1' >>"$DEF"
      fi

      systemctl restart kubelet >/dev/null 2>&1 || true

      echo "node-01's kubelet is crash-looping and will not stay up."
      echo
      echo "From cplane-01 the node is going NotReady. Get on node-01 and find"
      echo "out why the kubelet won't come up:"
      echo "    systemctl status kubelet"
      echo "    journalctl -u kubelet -n 40 --no-pager"
      echo
      echo "The service log names the problem. Fix the host, then restart the"
      echo "kubelet and confirm node-01 is Ready again."
      echo "(original extra-args file backed up at $BASE)"
  verify_done:
    needs:
      - init_scenario
    machine: node-01
    user: root
    run: |
      DEF=/etc/default/kubelet
      CONF=/etc/kubernetes/kubelet.conf

      if grep -q 'kubelings-nonexistent-flag' "$DEF" 2>/dev/null; then
        echo "not yet: the invalid flag is still in $DEF."
        echo "         journalctl -u kubelet shows the flag it chokes on. Remove it"
        echo "         from KUBELET_EXTRA_ARGS, then restart the kubelet."
        exit 1
      fi

      if ! systemctl is-active --quiet kubelet; then
        echo "not yet: the kubelet on node-01 isn't running."
        echo "         systemctl status kubelet ; journalctl -u kubelet -n40 --no-pager"
        exit 1
      fi

      if [ ! -s "$CONF" ]; then
        echo "not yet: no $CONF on node-01 — this node isn't a cluster member."
        exit 1
      fi

      # The Node authorizer lets a kubelet read its OWN node object. Use the
      # node's own credential to prove the cluster now considers it Ready.
      ready="$(kubectl --kubeconfig="$CONF" get node node-01 \
        -o jsonpath='{range .status.conditions[?(@.type=="Ready")]}{.status}{end}' 2>/dev/null || true)"
      if [ "$ready" != "True" ]; then
        echo "not yet: kubelet is up but node-01 is Ready=$ready."
        echo "         Give it a few seconds to post status, then re-check."
        exit 1
      fi

      echo "PASS — the bad flag is gone, the kubelet is running, and node-01 is Ready."
      echo
      kubectl --kubeconfig="$CONF" get node node-01 -o wide 2>/dev/null || true
---
