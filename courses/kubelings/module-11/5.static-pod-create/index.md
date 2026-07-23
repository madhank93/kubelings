---
kind: lesson
title: 'static pods: author one by hand, find its mirror'
description: |
  The classic exam task and the mechanism the whole control plane is built on:
  drop a pod manifest into /etc/kubernetes/manifests on a node and the kubelet
  runs it directly — no scheduler, no API needed to start it — then publishes a
  read-only "mirror" object so the pod shows up in kubectl. You'll write one,
  watch the kubelet pick it up, and confirm the mirror.
name: static-pod-create
slug: static-pod-create
createdAt: "2026-07-23"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: node-02
    user: root
    timeout_seconds: 120
    run: |
      set -euo pipefail
      MAN=/etc/kubernetes/manifests/web.yaml

      # Clean slate: remove any leftover from a previous attempt so the check
      # can't pass on a stale mirror pod.
      rm -f "$MAN" 2>/dev/null || true

      echo "Task: author a STATIC POD on node-02."
      echo
      echo "  * Write a pod manifest to:  $MAN"
      echo "  * Pod name:                 web"
      echo "  * Namespace:                default"
      echo "  * One container, any image that starts and stays up"
      echo "    (registry.k8s.io/pause:3.9 is already on this node and ideal)."
      echo
      echo "The kubelet watches /etc/kubernetes/manifests and will start it"
      echo "directly — no scheduler involved. It then publishes a read-only"
      echo "MIRROR pod in the API named 'web-node-02'."
      echo
      echo "Done when 'web-node-02' shows Running in the API and is a genuine"
      echo "static pod (config source: file), not a pod you kubectl-created."
  verify_done:
    needs:
      - init_scenario
    machine: node-02
    user: root
    timeout_seconds: 120
    run: |
      MAN=/etc/kubernetes/manifests/web.yaml
      CONF=/etc/kubernetes/kubelet.conf
      MIRROR=web-node-02

      if [ ! -s "$MAN" ]; then
        echo "not yet: no manifest at $MAN."
        echo "         Write a pod named 'web' there; the kubelet runs it on sight."
        exit 1
      fi

      if [ ! -s "$CONF" ]; then
        echo "not yet: no $CONF on node-02 — can't query the API from here."
        exit 1
      fi

      # The Node authorizer lets node-02's kubelet read pods bound to node-02 —
      # the mirror pod is one, so the node's own credential can verify it.
      phase="$(kubectl --kubeconfig="$CONF" -n default get pod "$MIRROR" \
        -o jsonpath='{.status.phase}' 2>/dev/null || true)"
      if [ -z "$phase" ]; then
        echo "not yet: the mirror pod '$MIRROR' isn't in the API yet."
        echo "         'crictl ps | grep web' — is the kubelet running the container?"
        echo "         Static-pod mirrors are named <pod>-<nodeName>; on node-02"
        echo "         a pod 'web' appears as 'web-node-02'. Give it a few seconds."
        exit 1
      fi

      src="$(kubectl --kubeconfig="$CONF" -n default get pod "$MIRROR" \
        -o jsonpath='{.metadata.annotations.kubernetes\.io/config\.source}' 2>/dev/null || true)"
      if [ "$src" != "file" ]; then
        echo "not yet: '$MIRROR' exists but its config source is '$src', not 'file'."
        echo "         That means it isn't a static pod. Remove it and instead place"
        echo "         the manifest in $MAN so the kubelet owns it."
        exit 1
      fi

      if [ "$phase" != "Running" ]; then
        echo "not yet: '$MIRROR' is a static pod (good) but phase=$phase, not Running."
        echo "         'crictl ps -a | grep web' and check the container/image."
        exit 1
      fi

      echo "PASS — 'web-node-02' is a Running static pod sourced from a file. That's"
      echo "the same mechanism that runs the control plane's own pods."
      echo
      kubectl --kubeconfig="$CONF" -n default get pod "$MIRROR" -o wide 2>/dev/null || true
---
