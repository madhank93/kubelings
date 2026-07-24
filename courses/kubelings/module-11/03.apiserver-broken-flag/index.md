---
kind: lesson
title: 'you broke the control plane: kube-apiserver in a crash loop'
description: |
  A bad flag in the kube-apiserver static-pod manifest sends it into a crash
  loop and the API server is gone — kubectl times out cluster-wide. This is the
  exam's "you broke the control plane, fix it" task: with no API to query, you
  drop to crictl to read the container's logs, find the flag it's dying on,
  correct the manifest, and watch the kubelet bring the API back.
name: apiserver-broken-flag
slug: apiserver-broken-flag
createdAt: "2026-07-23"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 180
    run: |
      set -euo pipefail
      MAN=/etc/kubernetes/manifests/kube-apiserver.yaml
      BASE=/root/kubelings-apiserver-baseline.yaml

      # Safety net + baseline. The drill is to FIND and remove the bad flag, but
      # a pristine copy means a botched edit is never fatal.
      cp "$MAN" "$BASE"

      # Inject a flag kube-apiserver doesn't recognise, right after the binary
      # name. It will exit immediately at startup — the kubelet keeps recreating
      # it (crash loop) and the API server never becomes ready.
      awk '{print} /- kube-apiserver$/{print "    - --kubelings-invalid-flag=true"}' \
        "$BASE" > "$MAN"

      echo "kube-apiserver on cplane-01 is crash-looping — the API is down."
      echo "kubectl will time out until you fix it."
      echo
      echo "With no API to ask, drop to the runtime to read the crash:"
      echo "    crictl ps -a | grep kube-apiserver"
      echo "    crictl logs <container-id>        # the exact flag it dies on"
      echo
      echo "Then edit $MAN, remove the bad flag, and let the kubelet rebuild"
      echo "the static pod. (Pristine copy at $BASE if you need it.)"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      MAN=/etc/kubernetes/manifests/kube-apiserver.yaml
      BASE=/root/kubelings-apiserver-baseline.yaml
      KC=/etc/kubernetes/admin.conf

      if [ ! -s "$BASE" ]; then
        echo "not yet: baseline $BASE missing — re-run init for this lesson."
        exit 1
      fi

      if grep -q 'kubelings-invalid-flag' "$MAN"; then
        echo "not yet: the bad flag is still in $MAN."
        echo "         Remove the '- --kubelings-invalid-flag=true' line and save;"
        echo "         the kubelet recreates the static pod within ~20s."
        exit 1
      fi

      # The real test: is the API server actually serving again?
      if ! kubectl --kubeconfig="$KC" get --raw=/readyz >/dev/null 2>&1; then
        echo "not yet: the API server isn't ready yet."
        echo "         crictl ps | grep kube-apiserver   # is it running now?"
        echo "         crictl logs <id>                   # any remaining startup error?"
        echo "         Give the kubelet ~20s after your edit to rebuild the pod."
        exit 1
      fi

      echo "PASS — the manifest is clean and kube-apiserver is serving again."
      echo
      kubectl --kubeconfig="$KC" get --raw='/readyz?verbose' 2>/dev/null | tail -3 || true
      kubectl --kubeconfig="$KC" get nodes 2>/dev/null || true
---
