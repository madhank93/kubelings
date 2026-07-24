---
kind: lesson
title: 'kubeadm: a cluster from three commands'
description: |
  The full kubeadm init → join runbook: what each phase does, the
  pod-network-cidr decision you can't take back, join tokens and their
  expiry, and the CNI step everyone forgets. Then run the join half for
  real — init tears node-02 out of the cluster, and you mint a fresh join
  command and bring it back Ready, the exact TLS-bootstrap flow a new node
  takes.
name: kubeadm-bootstrap
slug: kubeadm-bootstrap
createdAt: "2026-07-13"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: node-02
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      BASE=/root/kubelings-kubeadm-baseline

      # Record the moment we tore the node out. The check proves the node was
      # RE-joined (kubelet.conf newer than this) — not that it was never gone.
      printf 'RESET_EPOCH=%s\n' "$(date +%s)" >"$BASE"
      chmod 600 "$BASE"

      echo "Tearing worker node-02 out of the cluster (kubeadm reset)..."
      # -f: no interactive confirm. On a worker this removes /etc/kubernetes,
      # the PKI, and kubelet.conf, and stops the kubelet's cluster membership.
      kubeadm reset -f >/dev/null 2>&1 || true
      systemctl stop kubelet >/dev/null 2>&1 || true

      # reset tells you what it leaves behind — the CNI config and iptables.
      rm -rf /etc/cni/net.d/* 2>/dev/null || true

      # A leftover kubeconfig would let the check pass for the wrong reason.
      rm -f /etc/kubernetes/kubelet.conf 2>/dev/null || true

      echo
      echo "node-02 is now un-joined: no kubelet.conf, no PKI, kubelet stopped."
      echo "From cplane-01 it will show NotReady, then vanish."
      echo
      echo "Bring it back. On cplane-01, mint a join command:"
      echo "    kubeadm token create --print-join-command"
      echo "Then run that command HERE on node-02, as root."
      echo
      echo "(baseline in $BASE — don't edit it, the check reads it)"
  verify_done:
    needs:
      - init_scenario
    machine: node-02
    user: root
    run: |
      BASE=/root/kubelings-kubeadm-baseline
      CONF=/etc/kubernetes/kubelet.conf

      if [ ! -s "$BASE" ]; then
        echo "not yet: baseline file $BASE is missing — re-run init for this lesson."
        exit 1
      fi
      # shellcheck disable=SC1090
      . "$BASE"

      if [ ! -s "$CONF" ]; then
        echo "not yet: node-02 has no $CONF — it hasn't re-joined."
        echo "         On cplane-01, in order:"
        echo "           kubectl delete node node-02       # drop the stale Node object"
        echo "           kubeadm token create --print-join-command"
        echo "         Then run that 'kubeadm join …' line here on node-02 as root."
        echo "         (skip the delete and join fails: 'a Node named node-02 already exists')"
        exit 1
      fi

      conf_epoch="$(stat -c %Y "$CONF" 2>/dev/null || echo 0)"
      if [ "${conf_epoch:-0}" -le "${RESET_EPOCH:-0}" ]; then
        echo "not yet: the kubelet.conf on node-02 is older than the reset —"
        echo "         this looks like a leftover, not a fresh join. Re-join:"
        echo "         on cplane-01 'kubectl delete node node-02' then"
        echo "         'kubeadm token create --print-join-command', run join here."
        exit 1
      fi

      if ! systemctl is-active --quiet kubelet; then
        echo "not yet: kubelet.conf exists but the kubelet isn't running."
        echo "         systemctl status kubelet ; journalctl -u kubelet -n30"
        exit 1
      fi

      # The Node authorizer lets a kubelet read its OWN node object — use the
      # node's own credential to prove the cluster considers it Ready.
      ready="$(kubectl --kubeconfig="$CONF" get node node-02 \
        -o jsonpath='{range .status.conditions[?(@.type=="Ready")]}{.status}{end}' 2>/dev/null || true)"
      if [ "$ready" != "True" ]; then
        echo "not yet: node-02 re-joined but isn't Ready yet (Ready=$ready)."
        echo "         Give the CNI DaemonSet a moment to land, then re-check:"
        echo "         crictl ps | grep -E 'kube-proxy|flannel|cni'"
        exit 1
      fi

      echo "PASS — node-02 re-joined via a fresh TLS bootstrap and is Ready."
      echo
      kubectl --kubeconfig="$CONF" get node node-02 -o wide 2>/dev/null || true
---
