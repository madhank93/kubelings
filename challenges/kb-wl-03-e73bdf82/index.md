---
kind: challenge

title: "StatefulSet with Stable Pod Identity + Headless Service"
description: |
  A stateful app needs stable, predictable pod names and per-pod DNS. Build a
  StatefulSet fronted by a headless Service so each replica gets a durable
  network identity, then confirm all replicas are Ready.

categories:
- kubernetes

tagz:
- cka
- ckad
- workloads
- statefulset

difficulty: medium

createdAt: 2026-06-30

playground:
  name: k8s-omni

tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 180
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -

  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # Headless Service: clusterIP must be None.
      kubectl -n "$NS" get svc web >/dev/null 2>&1 || { echo "not yet: no Service 'web'"; exit 1; }
      cip=$(kubectl -n "$NS" get svc web -o jsonpath='{.spec.clusterIP}')
      [ "$cip" = "None" ] || { echo "not yet: Service 'web' is not headless (clusterIP=$cip, want None)"; exit 1; }
      # StatefulSet must be fully ready.
      kubectl -n "$NS" get statefulset web >/dev/null 2>&1 || { echo "not yet: no StatefulSet 'web'"; exit 1; }
      desired=$(kubectl -n "$NS" get sts web -o jsonpath='{.spec.replicas}')
      ready=$(kubectl -n "$NS" get sts web -o jsonpath='{.status.readyReplicas}')
      if [ "${desired:-0}" -lt 2 ]; then echo "not yet: want at least 2 replicas (got ${desired:-0})"; exit 1; fi
      if [ "${ready:-0}" -ne "${desired:-0}" ]; then echo "not yet: $ready/$desired replicas Ready"; exit 1; fi
      # serviceName must wire the STS to the headless Service.
      svcname=$(kubectl -n "$NS" get sts web -o jsonpath='{.spec.serviceName}')
      [ "$svcname" = "web" ] || { echo "not yet: StatefulSet.spec.serviceName must be 'web' (got '$svcname')"; exit 1; }
      echo "PASS — StatefulSet web has $ready stable replicas behind headless Service web."
---

## The situation

You're deploying a clustered, stateful workload where peers must address each
other by **stable hostname** (e.g. `web-0.web`, `web-1.web`). A Deployment +
ClusterIP Service can't give that — you need a **StatefulSet** plus a **headless
Service** (`clusterIP: None`) to publish per-pod DNS records.

## Your task

In namespace `kubelings`:

1. Create a **headless Service** named `web` (`clusterIP: None`) selecting `app=web`.
2. Create a **StatefulSet** named `web` with **≥ 2 replicas**, `serviceName: web`,
   image `ghcr.io/iximiuz/labs/nginx:alpine`.
3. All replicas must be Ready (`web-0`, `web-1`, …).

```sh
kubectl -n kubelings get sts,svc,pods
kubectl -n kubelings exec web-0 -- nslookup web-1.web 2>/dev/null || true
```

<details>
<summary>Hint</summary>

The Service must come first and be headless; the StatefulSet's `serviceName`
must point at it. See `solution.md` for a complete manifest.

</details>
