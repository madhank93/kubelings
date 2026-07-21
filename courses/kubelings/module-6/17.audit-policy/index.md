---
kind: lesson
title: 'Audit policy: who touched that Secret?'
description: |
  Turn on audit logging for real: write a policy that captures Secret reads
  and pod exec without drowning in kubelet noise, wire the kube-apiserver
  static pod, then answer "who read that Secret?" from the log — without
  copying the Secret itself into it.
name: audit-policy
slug: audit-policy
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
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" delete secret db-creds --ignore-not-found >/dev/null 2>&1 || true
      kubectl -n "$NS" create secret generic db-creds \
        --from-literal=password=s3cure-AUDIT-4242 >/dev/null
      mkdir -p /etc/kubernetes/audit /var/log/kubernetes/audit
      echo "Created Secret kubelings/db-creds."
      echo
      echo "Now answer this: who has read it?"
      echo
      if [ -s /var/log/kubernetes/audit/audit.log ]; then
        echo "  (an audit log already exists at /var/log/kubernetes/audit/audit.log)"
      else
        echo "  You can't. The apiserver records nothing about requests by default —"
        echo "  no record of who read which Secret, who exec'd where, who deleted what."
        echo "  Audit logging is opt-in, and the opt-in is a policy file plus a flag."
      fi
      echo
      echo "Empty directories are waiting for you:"
      echo "  /etc/kubernetes/audit/     ← your policy.yaml"
      echo "  /var/log/kubernetes/audit/ ← where the apiserver should write"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    run: |
      NS=kubelings
      PLAIN=s3cure-AUDIT-4242

      LOG=/var/log/kubernetes/audit/audit.log
      if [ ! -s "$LOG" ]; then
        # Be forgiving about the exact path the learner chose.
        LOG="$(find /var/log -name 'audit*.log' -size +0 2>/dev/null | head -1)"
      fi
      if [ -z "${LOG:-}" ] || [ ! -s "$LOG" ]; then
        echo "not yet: no non-empty audit log found."
        echo "         Expected /var/log/kubernetes/audit/audit.log — check that"
        echo "         --audit-policy-file and --audit-log-path are both on"
        echo "         kube-apiserver, that the log dir is mounted hostPath"
        echo "         read-write, and that the static pod actually restarted:"
        echo "           crictl ps | grep apiserver"
        exit 1
      fi

      # The event that answers "who read that Secret?"
      hits="$(mktemp)"
      grep -a '"resource":"secrets"' "$LOG" 2>/dev/null \
        | grep -a '"name":"db-creds"' \
        | grep -a '"verb":"get"' >"$hits" 2>/dev/null || true
      if [ ! -s "$hits" ]; then
        echo "not yet: found an audit log ($LOG), but no recorded 'get' of"
        echo "         kubelings/db-creds in it."
        echo "         Two things to check:"
        echo "           1. does your policy record Secret reads at Metadata or above?"
        echo "              (a 'level: None' rule earlier in the list wins — order is the policy)"
        echo "           2. have you actually read it since wiring the apiserver?"
        echo "                kubectl -n $NS get secret db-creds -o yaml"
        exit 1
      fi

      # …recorded without copying the Secret into the log. A 'get' logged at
      # RequestResponse writes the credential itself into the audit file —
      # which is the exact leak the audit trail exists to detect.
      if grep -qa "$PLAIN" "$hits"; then
        echo "not yet: the read IS recorded — but the audit event contains the"
        echo "         Secret's value in plaintext."
        echo "         A 'get secret' response body IS the secret. Logging reads at"
        echo "         Request/RequestResponse copies every credential into the audit"
        echo "         log, creating the leak you're auditing for. Record Secret"
        echo "         *reads* at Metadata; save RequestResponse for writes."
        exit 1
      fi

      echo "PASS — the audit log answers 'who read kubelings/db-creds?':"
      grep -o '"username":"[^"]*"' "$hits" | sort -u | head -5 | sed 's/^/       /'
      echo "       …and it did so without copying the Secret into the log."
---
