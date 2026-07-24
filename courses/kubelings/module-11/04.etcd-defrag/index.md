---
kind: lesson
title: 'etcd NOSPACE: the alarm that makes your cluster read-only'
description: |
  etcd has tripped its backend quota — the "mvcc: database space exceeded"
  alarm is armed and the whole cluster has gone read-only: no writes, no new
  pods, nothing. This is the etcd failure that isn't backup/restore (that's
  7.6). You'll read the alarm, compact old revisions, defragment the backend to
  reclaim disk, right-size the quota, and disarm — the exact NOSPACE runbook.
name: etcd-defrag
slug: etcd-defrag
createdAt: "2026-07-23"
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
      MAN=/etc/kubernetes/manifests/etcd.yaml
      BASE=/root/kubelings-etcd-baseline.yaml
      KC=/etc/kubernetes/admin.conf
      cp "$MAN" "$BASE"

      ectl() {
        kubectl --kubeconfig="$KC" -n kube-system exec etcd-cplane-01 -c etcd -- \
          etcdctl --endpoints=https://127.0.0.1:2379 \
          --cacert=/etc/kubernetes/pki/etcd/ca.crt \
          --cert=/etc/kubernetes/pki/etcd/server.crt \
          --key=/etc/kubernetes/pki/etcd/server.key "$@"
      }

      # Read etcd's current on-disk size, then set the backend quota BELOW it.
      # etcd allows startup but rejects the next write — the NOSPACE alarm arms
      # on its own from the API server's normal write traffic. No fill loop.
      DB=$(ectl endpoint status -w json 2>/dev/null | grep -o '"dbSize":[0-9]*' | head -1 | cut -d: -f2)
      DB=${DB:-8388608}
      QUOTA=$(( DB * 6 / 10 ))
      [ "$QUOTA" -lt 1048576 ] && QUOTA=1048576
      echo "etcd dbSize=${DB}B — setting an artificially low quota of ${QUOTA}B."

      # Replace an existing quota flag or inject one right after the binary name.
      if grep -q 'quota-backend-bytes' "$MAN"; then
        sed -i "s#\(- --quota-backend-bytes=\).*#\1${QUOTA}#" "$MAN"
      else
        awk -v q="$QUOTA" '{print} /- etcd$/{print "    - --quota-backend-bytes=" q}' \
          "$BASE" > "$MAN"
      fi

      # Bounce the etcd static pod so the new quota takes effect.
      mv "$MAN" /root/etcd.staged.yaml; sleep 8; mv /root/etcd.staged.yaml "$MAN"
      echo "waiting for etcd to come back..."
      for _ in $(seq 1 60); do ectl endpoint health >/dev/null 2>&1 && break; sleep 2; done

      # Poke writes until the alarm arms (the API server would do this anyway).
      for i in $(seq 1 30); do
        kubectl --kubeconfig="$KC" -n kube-system create configmap "kubelings-poke-$i" \
          --from-literal=x=y >/dev/null 2>&1 || true
        ectl alarm list 2>/dev/null | grep -qi NOSPACE && break
        sleep 1
      done

      echo
      if ectl alarm list 2>/dev/null | grep -qi NOSPACE; then
        echo "DONE — etcd has raised the NOSPACE alarm; the cluster is read-only."
      else
        echo "NOTE — alarm not visible yet; a write will trip it. Check 'alarm list'."
      fi
      echo
      echo "Recover it, on cplane-01, via etcdctl inside the etcd pod:"
      echo "  1. see it:     etcdctl alarm list ; etcdctl endpoint status -w table"
      echo "  2. compact:    etcdctl compact <current-revision>"
      echo "  3. defrag:     etcdctl defrag --command-timeout=60s"
      echo "  4. right-size: raise --quota-backend-bytes in $MAN back to a sane value"
      echo "  5. disarm:     etcdctl alarm disarm"
      echo "(baseline manifest at $BASE)"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      KC=/etc/kubernetes/admin.conf
      ectl() {
        kubectl --kubeconfig="$KC" -n kube-system exec etcd-cplane-01 -c etcd -- \
          etcdctl --endpoints=https://127.0.0.1:2379 \
          --cacert=/etc/kubernetes/pki/etcd/ca.crt \
          --cert=/etc/kubernetes/pki/etcd/server.crt \
          --key=/etc/kubernetes/pki/etcd/server.key "$@"
      }

      if ! ectl endpoint health >/dev/null 2>&1; then
        echo "not yet: etcd isn't answering. If you edited the manifest, wait for"
        echo "         the kubelet to rebuild the etcd pod (crictl ps | grep etcd)."
        exit 1
      fi

      if ectl alarm list 2>/dev/null | grep -qi NOSPACE; then
        echo "not yet: the NOSPACE alarm is still armed."
        echo "         Reclaim space first (compact + defrag) and raise the quota,"
        echo "         THEN 'etcdctl alarm disarm' — disarming while still over"
        echo "         quota just re-arms on the next write."
        exit 1
      fi

      # The real proof the cluster is writable again: a fresh write must stick
      # AND not re-arm the alarm (which it would if the quota is still too low).
      PROBE="kubelings-defrag-ok-$$"
      kubectl --kubeconfig="$KC" -n kube-system delete configmap "$PROBE" >/dev/null 2>&1 || true
      if ! kubectl --kubeconfig="$KC" -n kube-system create configmap "$PROBE" \
           --from-literal=ok=1 >/dev/null 2>&1; then
        echo "not yet: a test write failed — etcd is still refusing writes."
        exit 1
      fi
      sleep 2
      if ectl alarm list 2>/dev/null | grep -qi NOSPACE; then
        echo "not yet: the write re-armed NOSPACE — the quota is still smaller than"
        echo "         the database. Raise --quota-backend-bytes in the etcd manifest"
        echo "         to a sane value (e.g. the kubeadm default), then disarm again."
        kubectl --kubeconfig="$KC" -n kube-system delete configmap "$PROBE" >/dev/null 2>&1 || true
        exit 1
      fi

      # Tidy the pokes init left behind so the drill leaves a clean namespace.
      kubectl --kubeconfig="$KC" -n kube-system delete configmap "$PROBE" >/dev/null 2>&1 || true
      for i in $(seq 1 30); do
        kubectl --kubeconfig="$KC" -n kube-system delete configmap "kubelings-poke-$i" >/dev/null 2>&1 || true
      done

      echo "PASS — no NOSPACE alarm, and writes stick. etcd is healthy again."
      echo
      ectl endpoint status -w table 2>/dev/null || true
---
