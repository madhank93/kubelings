---
kind: lesson
title: 'Certificates: the outage scheduled a year in advance'
description: |
  kubeadm clusters mint one-year certificates, and expired certs are a
  whole-cluster lockout that arrives on schedule. Run the real drill on a
  real control plane: check-expiration, renew all, restart the static pods,
  re-copy admin.conf — and prove the apiserver is actually serving the new
  certificate, not just holding it on disk.
name: cert-rotation
slug: cert-rotation
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
      CRT=/etc/kubernetes/pki/apiserver.crt
      BASE=/etc/kubernetes/kubelings-cert-baseline

      end_of() { openssl x509 -enddate -noout -in "$1" 2>/dev/null | cut -d= -f2; }
      epoch_of() { date -d "$1" +%s 2>/dev/null || echo 0; }

      disk_end="$(end_of "$CRT")"
      # The certificate the apiserver is actually SERVING, which is the one it
      # loaded into memory at startup — not necessarily the one on disk.
      served_end="$(echo | openssl s_client -connect 127.0.0.1:6443 2>/dev/null \
        | openssl x509 -enddate -noout 2>/dev/null | cut -d= -f2 || true)"

      printf 'disk_epoch=%s\nserved_epoch=%s\n' \
        "$(epoch_of "$disk_end")" "$(epoch_of "${served_end:-}")" >"$BASE"
      chmod 600 "$BASE"

      echo "Control-plane PKI, as of right now:"
      echo
      kubeadm certs check-expiration 2>/dev/null | head -20 || true
      echo
      echo "apiserver cert on disk    expires: ${disk_end:-unknown}"
      echo "apiserver cert being SERVED expires: ${served_end:-unknown}"
      echo
      echo "Those two are the same today. Your job is to make both of them later"
      echo "than they are now — which takes two different actions."
      echo
      echo "(baseline recorded in $BASE — don't edit it, the check reads it)"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    run: |
      CRT=/etc/kubernetes/pki/apiserver.crt
      BASE=/etc/kubernetes/kubelings-cert-baseline

      if [ ! -s "$BASE" ]; then
        echo "not yet: baseline file $BASE is missing — re-run init for this lesson."
        exit 1
      fi
      # shellcheck disable=SC1090
      . "$BASE"

      epoch_of() { date -d "$1" +%s 2>/dev/null || echo 0; }
      disk_now="$(epoch_of "$(openssl x509 -enddate -noout -in "$CRT" 2>/dev/null | cut -d= -f2)")"
      served_now="$(epoch_of "$(echo | openssl s_client -connect 127.0.0.1:6443 2>/dev/null \
        | openssl x509 -enddate -noout 2>/dev/null | cut -d= -f2)")"

      if [ "${disk_now:-0}" -le "${disk_epoch:-0}" ]; then
        echo "not yet: /etc/kubernetes/pki/apiserver.crt still has its original expiry."
        echo "         Nothing has been renewed yet:"
        echo "           kubeadm certs check-expiration"
        echo "           kubeadm certs renew all"
        exit 1
      fi
      if [ "${served_now:-0}" -le "${served_epoch:-0}" ]; then
        echo "not yet: the certificate ON DISK is renewed, but the apiserver is still"
        echo "         SERVING the old one — it loaded that into memory at startup and"
        echo "         has no idea the file changed."
        echo
        echo "         This is the classic incomplete fix: renewed files are not"
        echo "         renewed certs. Restart the static pods:"
        echo "           mkdir -p /tmp/manifests"
        echo "           mv /etc/kubernetes/manifests/*.yaml /tmp/manifests/ && sleep 20 \\"
        echo "             && mv /tmp/manifests/*.yaml /etc/kubernetes/manifests/"
        echo "           crictl ps | grep -E 'apiserver|scheduler|controller|etcd'"
        exit 1
      fi
      if ! kubectl get nodes >/dev/null 2>&1; then
        echo "not yet: certs are renewed and served, but kubectl can't reach the cluster."
        echo "         Your kubeconfig still embeds the OLD client cert — 'renew all'"
        echo "         rewrote /etc/kubernetes/admin.conf, so copy it again:"
        echo "           cp /etc/kubernetes/admin.conf ~/.kube/config"
        exit 1
      fi

      echo "PASS — renewed on disk, served from memory, and kubectl still works."
      echo
      kubeadm certs check-expiration 2>/dev/null | head -12 || true
---
