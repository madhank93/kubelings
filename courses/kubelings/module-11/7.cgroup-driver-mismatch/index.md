---
kind: lesson
title: 'cgroup driver mismatch: kubelet and containerd disagree'
description: |
  A node's kubelet and its container runtime are set to different cgroup drivers
  — the single most common kubeadm install gotcha. It's a latent fault: the node
  can look Ready while resource accounting is quietly split across two schemes,
  which is unsupported and bites under memory pressure. You'll detect the
  disagreement by reading both configs and align them to systemd, the correct
  driver on a systemd host.
name: cgroup-driver-mismatch
slug: cgroup-driver-mismatch
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
      KCFG=/var/lib/kubelet/config.yaml
      BASE=/root/kubelings-cgroup-baseline

      # Record the original driver so the lesson is undoable and the check has a
      # reference point.
      { echo "kubelet=$(grep -E '^cgroupDriver:' "$KCFG" 2>/dev/null | awk '{print $2}')";
        echo "containerd=$(containerd config dump 2>/dev/null | grep -iE 'SystemdCgroup' | head -1 | awk -F= '{gsub(/ /,"",$2);print $2}')";
      } >"$BASE"
      chmod 600 "$BASE"

      # Introduce the mismatch: flip the KUBELET to cgroupfs while containerd
      # stays on systemd. Now the two manage cgroups under different schemes.
      if grep -qE '^cgroupDriver:' "$KCFG"; then
        sed -i 's/^cgroupDriver:.*/cgroupDriver: cgroupfs/' "$KCFG"
      else
        printf '\ncgroupDriver: cgroupfs\n' >>"$KCFG"
      fi
      systemctl restart kubelet >/dev/null 2>&1 || true

      echo "node-02's kubelet now uses cgroupDriver=cgroupfs while containerd"
      echo "uses systemd. They disagree — a latent misconfiguration. The node may"
      echo "still look Ready, but this split is unsupported and unsafe under load."
      echo
      echo "Detect it on node-02 by comparing the two configs:"
      echo "    grep cgroupDriver $KCFG                      # kubelet's driver"
      echo "    containerd config dump | grep -i SystemdCgroup   # runtime's driver"
      echo
      echo "Align BOTH to systemd (correct on a systemd host), restart, and"
      echo "confirm node-02 is Ready. (baseline in $BASE)"
  verify_done:
    needs:
      - init_scenario
    machine: node-02
    user: root
    timeout_seconds: 120
    run: |
      KCFG=/var/lib/kubelet/config.yaml
      CONF=/etc/kubernetes/kubelet.conf

      kdrv="$(grep -E '^cgroupDriver:' "$KCFG" 2>/dev/null | awk '{print $2}')"
      if [ "$kdrv" != "systemd" ]; then
        echo "not yet: the kubelet's cgroupDriver is '$kdrv', not systemd."
        echo "         Set 'cgroupDriver: systemd' in $KCFG and restart the kubelet."
        exit 1
      fi

      cdrv="$(containerd config dump 2>/dev/null | grep -iE 'SystemdCgroup' | head -1 \
        | awk -F= '{gsub(/ /,"",$2);print $2}')"
      if [ "$cdrv" != "true" ]; then
        echo "not yet: containerd's SystemdCgroup is '$cdrv', not true — it isn't on"
        echo "         systemd. Set SystemdCgroup = true in containerd's config and"
        echo "         restart containerd so both sides agree on systemd."
        exit 1
      fi

      if ! systemctl is-active --quiet kubelet; then
        echo "not yet: the kubelet isn't running — restart it after aligning the driver."
        exit 1
      fi

      if [ ! -s "$CONF" ]; then
        echo "not yet: no $CONF on node-02 — can't confirm node status from here."
        exit 1
      fi

      ready="$(kubectl --kubeconfig="$CONF" get node node-02 \
        -o jsonpath='{range .status.conditions[?(@.type=="Ready")]}{.status}{end}' 2>/dev/null || true)"
      if [ "$ready" != "True" ]; then
        echo "not yet: drivers agree but node-02 is Ready=$ready — give the kubelet a"
        echo "         few seconds to settle after the restart, then re-check."
        exit 1
      fi

      echo "PASS — kubelet and containerd both use the systemd cgroup driver, and"
      echo "node-02 is Ready."
      echo
      echo "kubelet:    $(grep -E '^cgroupDriver:' "$KCFG")"
      echo "containerd: SystemdCgroup = $cdrv"
---
