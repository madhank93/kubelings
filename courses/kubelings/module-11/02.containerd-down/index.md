---
kind: lesson
title: 'containerd down: the CRI is gone and nothing will schedule'
description: |
  The container runtime on a worker won't start, so the kubelet has no CRI to
  drive — pods stay Pending or dead and crictl can't connect. You'll diagnose
  from the runtime side (crictl, systemctl, journalctl), clear the broken
  containerd config, and watch the node accept workloads again.
name: containerd-down
slug: containerd-down
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
      DROPIN=/etc/systemd/system/containerd.service.d/99-kubelings-broken.conf

      # Break containerd the way a bad config change does: override ExecStart to
      # point at a config file that doesn't exist. containerd exits immediately,
      # the CRI socket disappears, and the kubelet loses its runtime.
      mkdir -p "$(dirname "$DROPIN")"
      cat >"$DROPIN" <<'EOF'
      [Service]
      ExecStart=
      ExecStart=/usr/bin/containerd --config /etc/containerd/kubelings-nonexistent.toml
      EOF

      systemctl daemon-reload
      systemctl restart containerd >/dev/null 2>&1 || true

      echo "containerd on node-01 is down: no CRI socket, kubelet can't run pods."
      echo
      echo "Diagnose from the runtime side, on node-01:"
      echo "    crictl ps                 # connection refused — the socket is gone"
      echo "    systemctl status containerd"
      echo "    journalctl -u containerd -n 40 --no-pager"
      echo
      echo "Fix the runtime, restart it, and confirm node-01 can schedule again."
  verify_done:
    needs:
      - init_scenario
    machine: node-01
    user: root
    run: |
      DROPIN=/etc/systemd/system/containerd.service.d/99-kubelings-broken.conf
      CONF=/etc/kubernetes/kubelet.conf

      if [ -f "$DROPIN" ]; then
        echo "not yet: the broken containerd drop-in is still on disk —"
        echo "         $DROPIN"
        echo "         'systemctl cat containerd' shows the bad ExecStart it adds."
        echo "         Remove it, 'systemctl daemon-reload', restart containerd."
        exit 1
      fi

      if ! systemctl is-active --quiet containerd; then
        echo "not yet: containerd on node-01 isn't running."
        echo "         systemctl status containerd ; journalctl -u containerd -n40 --no-pager"
        exit 1
      fi

      # A live CRI must answer crictl — this is what the kubelet also depends on.
      if ! crictl version >/dev/null 2>&1; then
        echo "not yet: containerd is 'active' but crictl can't talk to the CRI socket."
        echo "         crictl version   # inspect the endpoint error"
        exit 1
      fi

      if [ ! -s "$CONF" ]; then
        echo "not yet: no $CONF on node-01 — this node isn't a cluster member."
        exit 1
      fi

      # With the CRI back, the kubelet reconnects and node-01 returns to Ready.
      ready="$(kubectl --kubeconfig="$CONF" get node node-01 \
        -o jsonpath='{range .status.conditions[?(@.type=="Ready")]}{.status}{end}' 2>/dev/null || true)"
      if [ "$ready" != "True" ]; then
        echo "not yet: containerd is up but node-01 is Ready=$ready — give the"
        echo "         kubelet a few seconds to reconnect to the CRI, then re-check."
        exit 1
      fi

      echo "PASS — containerd is running, crictl talks to it, and node-01 is Ready."
      echo
      crictl version | head -2 2>/dev/null || true
---
