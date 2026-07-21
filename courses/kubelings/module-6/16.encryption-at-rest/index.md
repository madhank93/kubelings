---
kind: lesson
title: 'Encryption at rest: the full runbook'
description: |
  Encrypt Secrets at rest on a real control plane: write an
  EncryptionConfiguration, wire the kube-apiserver static pod, and prove
  ciphertext with etcdctl — then run the re-encrypt migration everyone
  forgets. Covers provider order semantics, aescbc vs KMS v2, and key
  rotation without locking yourself out.
name: encryption-at-rest
slug: encryption-at-rest
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
        --from-literal=username=app \
        --from-literal=password=s3cure-NEW-9917 >/dev/null
      echo "Created Secret kubelings/db-creds."
      echo
      echo "Now look at what is actually on disk:"
      POD="$(kubectl -n kube-system get pod -l component=etcd \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
      if [ -n "$POD" ]; then
        kubectl -n kube-system exec "$POD" -- etcdctl \
          --endpoints=https://127.0.0.1:2379 \
          --cacert=/etc/kubernetes/pki/etcd/ca.crt \
          --cert=/etc/kubernetes/pki/etcd/server.crt \
          --key=/etc/kubernetes/pki/etcd/server.key \
          get /registry/secrets/"$NS"/db-creds 2>/dev/null \
          | head -c 400 | tr -c '[:print:]\n' '.' || true
        echo
        echo "↑ 's3cure-NEW-9917' is sitting there in plaintext, in every etcd"
        echo "  snapshot ever taken. RBAC guards the API; it does not guard the disk."
      fi
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    run: |
      NS=kubelings
      PLAIN=s3cure-NEW-9917

      # The Secret must still decrypt through the API: encryption is meant to be
      # transparent to clients, so deleting or rewriting the data is not a fix.
      got="$(kubectl -n "$NS" get secret db-creds -o jsonpath='{.data.password}' 2>/dev/null \
        | base64 -d 2>/dev/null || true)"
      if [ "$got" != "$PLAIN" ]; then
        echo "not yet: kubelings/db-creds no longer reads back as its original value."
        echo "         Encryption must be transparent to API clients — don't delete or"
        echo "         change the Secret, encrypt it where it is stored."
        exit 1
      fi

      POD="$(kubectl -n kube-system get pod -l component=etcd \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
      if [ -z "$POD" ]; then
        echo "not yet: no etcd pod in kube-system — is the control plane healthy?"
        exit 1
      fi
      raw="$(mktemp)"
      kubectl -n kube-system exec "$POD" -- etcdctl \
        --endpoints=https://127.0.0.1:2379 \
        --cacert=/etc/kubernetes/pki/etcd/ca.crt \
        --cert=/etc/kubernetes/pki/etcd/server.crt \
        --key=/etc/kubernetes/pki/etcd/server.key \
        get /registry/secrets/"$NS"/db-creds >"$raw" 2>/dev/null || true
      if [ ! -s "$raw" ]; then
        echo "not yet: could not read the etcd row for kubelings/db-creds."
        exit 1
      fi

      if grep -qa "$PLAIN" "$raw"; then
        echo "not yet: the raw etcd row still contains the plaintext password."
        echo "         Enabling encryption only affects NEW writes. Rewrite what is"
        echo "         already there:  kubectl -n $NS get secrets -o json | kubectl replace -f -"
        exit 1
      fi
      if ! grep -qa 'k8s:enc:' "$raw"; then
        echo "not yet: the etcd row carries no 'k8s:enc:' prefix, so the apiserver is"
        echo "         not encrypting Secrets. Check that --encryption-provider-config"
        echo "         is on kube-apiserver and that the static pod actually restarted:"
        echo "           crictl ps | grep apiserver"
        exit 1
      fi
      echo "PASS — db-creds still decrypts through the API, and on disk it is ciphertext."
      echo "       etcd backups of this cluster no longer leak your Secrets."
---
