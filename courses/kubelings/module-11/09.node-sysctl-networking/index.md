---
kind: lesson
title: 'pod networking dead: the kernel knobs kubeadm needs'
description: |
  Pods on a node can't route and Service traffic isn't being NAT'd — because the
  kernel prerequisites every kubeadm node depends on are off: the br_netfilter
  module isn't loaded and ip_forward / bridge-nf-call-iptables are 0. You'll
  restore the module and the sysctls at runtime AND persist them so they survive
  a reboot — the install gotcha that bites people on the second boot.
name: node-sysctl-networking
slug: node-sysctl-networking
createdAt: "2026-07-23"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: node-02
    user: root
    timeout_seconds: 180
    run: |
      set -euo pipefail

      # Make sure the module is present so we can flip its sysctl off, then break
      # the kernel networking prerequisites at runtime.
      modprobe br_netfilter 2>/dev/null || true
      sysctl -w net.ipv4.ip_forward=0 >/dev/null 2>&1 || true
      sysctl -w net.bridge.bridge-nf-call-iptables=0 >/dev/null 2>&1 || true

      # Strip the PERSISTENCE too — disable any sysctl.d / modules-load.d files
      # that would set these back. This is the real trap: fix it at runtime only
      # and the node breaks again on its next reboot.
      for f in /etc/sysctl.conf /etc/sysctl.d/*.conf; do
        [ -f "$f" ] || continue
        if grep -qE 'ip_forward|bridge-nf-call-iptables' "$f" 2>/dev/null; then
          sed -i -E 's/^([[:space:]]*net\.(ipv4\.ip_forward|bridge\.bridge-nf-call-iptables).*)/# kubelings-disabled: \1/' "$f"
        fi
      done
      for f in /etc/modules-load.d/*.conf; do
        [ -f "$f" ] || continue
        grep -q br_netfilter "$f" 2>/dev/null && mv "$f" "$f.kubelings-disabled" || true
      done

      # Best-effort unload so "load the module" is a real step (often in use).
      modprobe -r br_netfilter 2>/dev/null || true

      echo "node-02's pod-networking kernel prerequisites are off:"
      echo "    net.ipv4.ip_forward = 0"
      echo "    net.bridge.bridge-nf-call-iptables = 0"
      echo "and the config files that set them (and load br_netfilter) at boot are disabled."
      echo
      echo "Diagnose on node-02:"
      echo "    sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables"
      echo "    lsmod | grep br_netfilter    # (empty here — this kernel builds it in)"
      echo
      echo "Restore the sysctls at runtime AND make everything persist across reboot"
      echo "(sysctl.d for the knobs, modules-load.d for br_netfilter)."
  verify_done:
    needs:
      - init_scenario
    machine: node-02
    user: root
    timeout_seconds: 120
    run: |
      fail=0

      ipf="$(sysctl -n net.ipv4.ip_forward 2>/dev/null || echo x)"
      if [ "$ipf" != "1" ]; then
        echo "not yet: net.ipv4.ip_forward is '$ipf', must be 1."
        fail=1
      fi

      brn="$(sysctl -n net.bridge.bridge-nf-call-iptables 2>/dev/null || echo x)"
      if [ "$brn" != "1" ]; then
        echo "not yet: net.bridge.bridge-nf-call-iptables is '$brn', must be 1"
        echo "         (this key only exists once br_netfilter is present — load it"
        echo "          with 'modprobe br_netfilter' if your kernel builds it as a module)."
        fail=1
      fi

      # Persistence — the point of the lesson. A runtime-only fix reverts on reboot.
      if ! grep -rhsE '^[[:space:]]*net\.ipv4\.ip_forward[[:space:]]*=[[:space:]]*1' \
           /etc/sysctl.conf /etc/sysctl.d/ >/dev/null 2>&1; then
        echo "not yet: ip_forward=1 isn't persisted — add it under /etc/sysctl.d/"
        echo "         (e.g. /etc/sysctl.d/k8s.conf) so it survives a reboot."
        fail=1
      fi
      if ! grep -rhsE '^[[:space:]]*net\.bridge\.bridge-nf-call-iptables[[:space:]]*=[[:space:]]*1' \
           /etc/sysctl.conf /etc/sysctl.d/ >/dev/null 2>&1; then
        echo "not yet: bridge-nf-call-iptables=1 isn't persisted under /etc/sysctl.d/."
        fail=1
      fi
      if ! grep -rhsq --include='*.conf' '^br_netfilter' /etc/modules-load.d/ 2>/dev/null; then
        echo "not yet: br_netfilter isn't set to load at boot — add it under"
        echo "         /etc/modules-load.d/ (e.g. echo br_netfilter > /etc/modules-load.d/k8s.conf)."
        fail=1
      fi

      [ "$fail" -eq 0 ] || exit 1

      echo "PASS — ip_forward and bridge-nf-call-iptables are 1, and all the"
      echo "prerequisites are persisted (sysctl.d + modules-load.d) to survive a reboot."
      echo
      sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables 2>/dev/null || true
---
