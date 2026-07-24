---
kind: lesson
title: 'etcd backup & restore: the runbook you rehearse before you need it'
description: |
  Actually do it: snapshot etcd, destroy the object, restore it back. The
  cert flags everyone fumbles, the restore's new-cluster semantics, and why
  Reddit's Pi-Day recovery took five hours. The last lesson of "etcd is the
  cluster" — knowing how to put it back, rehearsed rather than read.
name: etcd-backup-restore
slug: etcd-backup-restore
createdAt: "2026-07-07"
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
      BASE=/etc/kubernetes/kubelings-etcd-baseline

      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" delete configmap treasure --ignore-not-found >/dev/null 2>&1 || true
      kubectl -n "$NS" create configmap treasure \
        --from-literal=ledger='the only copy of something that matters' >/dev/null

      uid="$(kubectl -n "$NS" get configmap treasure -o jsonpath='{.metadata.uid}')"

      etcd_json() {
        POD="$(kubectl -n kube-system get pod -l component=etcd \
          -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
        [ -n "$POD" ] || return 1
        kubectl -n kube-system exec "$POD" -- etcdctl \
          --endpoints=https://127.0.0.1:2379 \
          --cacert=/etc/kubernetes/pki/etcd/ca.crt \
          --cert=/etc/kubernetes/pki/etcd/server.crt \
          --key=/etc/kubernetes/pki/etcd/server.key \
          endpoint status --write-out=json 2>/dev/null
      }
      cid="$(etcd_json | grep -o '"cluster_id":[0-9]*' | head -1 | cut -d: -f2 || true)"

      mkdir -p /backup
      printf 'treasure_uid=%s\ncluster_id=%s\n' "$uid" "${cid:-0}" >"$BASE"
      chmod 600 "$BASE"

      echo "Created ConfigMap kubelings/treasure — pretend it's the only copy of"
      echo "something that matters."
      echo
      echo "  uid:            $uid"
      echo "  etcd cluster:   ${cid:-unknown}"
      echo
      echo "/backup/ exists and is empty. Nothing is broken yet — that's the point."
      echo "(baseline recorded in $BASE — don't edit it, the check reads it)"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    run: |
      NS=kubelings
      BASE=/etc/kubernetes/kubelings-etcd-baseline

      if [ ! -s "$BASE" ]; then
        echo "not yet: baseline file $BASE is missing — re-run init for this lesson."
        exit 1
      fi
      # shellcheck disable=SC1090
      . "$BASE"

      POD="$(kubectl -n kube-system get pod -l component=etcd \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
      if [ -z "$POD" ]; then
        echo "not yet: no etcd pod in kube-system — is the control plane back up?"
        echo "         After a restore the kubelet needs the static-pod manifests"
        echo "         returned to /etc/kubernetes/manifests/."
        exit 1
      fi
      cid_now="$(kubectl -n kube-system exec "$POD" -- etcdctl \
        --endpoints=https://127.0.0.1:2379 \
        --cacert=/etc/kubernetes/pki/etcd/ca.crt \
        --cert=/etc/kubernetes/pki/etcd/server.crt \
        --key=/etc/kubernetes/pki/etcd/server.key \
        endpoint status --write-out=json 2>/dev/null \
        | grep -o '"cluster_id":[0-9]*' | head -1 | cut -d: -f2 || true)"

      # A restored member believes it is founding a NEW cluster, so the cluster
      # ID changes. Nothing else in this lesson changes it — which makes it an
      # honest signal that a restore actually happened.
      if [ -z "${cid_now:-}" ] || [ "${cid_now:-0}" = "${cluster_id:-0}" ]; then
        echo "not yet: etcd is still running the original cluster (id ${cid_now:-unknown})."
        echo "         No restore has happened yet. The drill is:"
        echo "           1. snapshot save   (while everything is still fine)"
        echo "           2. delete kubelings/treasure   (the disaster)"
        echo "           3. snapshot restore into a NEW data dir, repoint etcd, bring it back"
        exit 1
      fi

      if ! kubectl -n "$NS" get configmap treasure >/dev/null 2>&1; then
        echo "not yet: a restore happened (new etcd cluster id $cid_now) — but"
        echo "         kubelings/treasure is not there."
        echo "         Your snapshot was taken BEFORE the ConfigMap existed, so the"
        echo "         world you restored never had it. A backup is only as good as"
        echo "         its timing: snapshot first, then break things."
        exit 1
      fi

      uid_now="$(kubectl -n "$NS" get configmap treasure -o jsonpath='{.metadata.uid}' 2>/dev/null)"
      if [ "${uid_now:-}" != "${treasure_uid:-}" ]; then
        echo "not yet: kubelings/treasure exists, but it is a different object."
        echo "           expected uid: ${treasure_uid:-?}"
        echo "           actual uid:   ${uid_now:-?}"
        echo "         A restore brings back the original object, identity and all."
        echo "         A hand-recreated ConfigMap with the same name is a new object —"
        echo "         and in a real incident, that difference is every reference,"
        echo "         ownerReference and audit trail that pointed at the old one."
        exit 1
      fi

      echo "PASS — etcd was restored (new cluster id $cid_now) and kubelings/treasure"
      echo "       came back as the same object it always was:"
      echo "         uid $uid_now"
      echo
      echo "You have now rehearsed the thing Reddit did for the first time under"
      echo "pressure, for five hours, on Pi Day."
---
