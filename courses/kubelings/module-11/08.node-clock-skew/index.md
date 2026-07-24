---
kind: lesson
title: 'clock skew: when a node time-travels out of the cluster'
description: |
  A worker's clock has drifted far from real time, and suddenly its kubelet
  can't talk to the API server — TLS handshakes fail with "certificate has
  expired or is not yet valid" even though every cert is fine. Time is a
  security input, and a skewed node is an untrusted node. You'll spot the skew,
  restore time synchronisation, and watch the node rejoin.
name: node-clock-skew
slug: node-clock-skew
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

      # Turn off time sync so nothing silently corrects the skew we introduce.
      timedatectl set-ntp false >/dev/null 2>&1 || true
      systemctl stop chronyd chrony systemd-timesyncd ntpd >/dev/null 2>&1 || true

      # Jump the clock forward past the validity window of the 1-year kubelet /
      # API-server certs. To the kubelet, those certs now look EXPIRED, so its
      # TLS to the API server fails and node-01 falls out of the cluster.
      if ! date -s '+400 days' >/dev/null 2>&1; then
        timedatectl set-time "$(date -d '+400 days' '+%Y-%m-%d %H:%M:%S')" >/dev/null 2>&1 || true
      fi

      systemctl restart kubelet >/dev/null 2>&1 || true

      echo "node-01's clock is now ~400 days fast and time sync is off."
      echo "From cplane-01 it's going NotReady; the kubelet's TLS to the API"
      echo "server fails with x509 'certificate has expired or is not yet valid'."
      echo
      echo "Diagnose on node-01:"
      echo "    date ; timedatectl                       # how far off is it?"
      echo "    journalctl -u kubelet -n 30 --no-pager   # x509 expiry errors"
      echo
      echo "Restore correct time by turning synchronisation back on, then confirm"
      echo "node-01 is Ready again."
  verify_done:
    needs:
      - init_scenario
    machine: node-01
    user: root
    timeout_seconds: 120
    run: |
      CONF=/etc/kubernetes/kubelet.conf

      # Require time sync to be back on — the real fix is restoring NTP, not
      # hand-setting the clock once.
      ntp="$(timedatectl show -p NTP --value 2>/dev/null || true)"
      if [ "$ntp" != "yes" ]; then
        echo "not yet: time synchronisation is off (NTP=$ntp)."
        echo "         Turn it back on: 'timedatectl set-ntp true' and start your"
        echo "         time service (chronyd / systemd-timesyncd), then let it sync."
        exit 1
      fi

      if ! systemctl is-active --quiet kubelet; then
        echo "not yet: the kubelet on node-01 isn't running."
        exit 1
      fi

      if [ ! -s "$CONF" ]; then
        echo "not yet: no $CONF on node-01 — can't reach the API from here."
        exit 1
      fi

      # The cluster's own TLS is the time oracle: if node-01's clock were still
      # skewed, this call would fail x509 validation. Ready proves time is sane
      # AND the kubelet has re-authenticated.
      ready="$(kubectl --kubeconfig="$CONF" get node node-01 \
        -o jsonpath='{range .status.conditions[?(@.type=="Ready")]}{.status}{end}' 2>/dev/null || true)"
      if [ "$ready" != "True" ]; then
        echo "not yet: node-01 is Ready=$ready. If time only just corrected, give the"
        echo "         kubelet a few seconds to re-establish TLS, then re-check."
        echo "         (still x509 errors in 'journalctl -u kubelet'? the clock is"
        echo "          not yet synced — 'chronyc makestep' can force an immediate step.)"
        exit 1
      fi

      echo "PASS — time sync is on, node-01's clock is trusted again, and it's Ready."
      echo
      timedatectl | grep -Ei 'Local time|synchronized|NTP service' || true
---
