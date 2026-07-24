---
kind: unit
title: "Incident replay — the ndots:5 DNS amplifier (Zalando, Jan 2019)"
name: incident-dns-ndots-unit
---


## The real incident

**January 7, 2019. Zalando — Europe's biggest fashion platform.** A downstream
service slows, the aggregation layer starts retrying, request volume spikes. Each
retried request opens fresh connections; each connection resolves the target's
hostname again (the Node.js apps had **no DNS caching**).

And then the multiplier kicks in. Every Kubernetes pod with the default
`dnsPolicy: ClusterFirst` gets a resolv.conf like this:

```
search kubelings.svc.cluster.local svc.cluster.local cluster.local ...
options ndots:5
```

`ndots:5` means: *any name with fewer than 5 dots is tried through every search
domain first.* `payments.example.com` has 2 dots — so the resolver tries
`payments.example.com.kubelings.svc.cluster.local`,
`.svc.cluster.local`, `.cluster.local`, … (each as A **and** AAAA) before finally
asking for the real name. **~10 DNS queries for every single external lookup.**

CoreDNS — running with a `100Mi` memory limit — ate the amplified flood, blew
past its limit, and was **OOMKilled. All replicas. Simultaneously.** Total
cluster DNS outage. The monitoring stack needed DNS too, so paging failed.
Fashion store error rates stayed high for **over an hour**.

Source: [Zalando postmortem — Jan 2019 DNS outage](https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jan-2019-dns-outage.md)

## This cluster, right now

The `checkout` Deployment in `kubelings` resolves `payments.example.com` in a
loop — stand-in for prod retry traffic. It runs with stock DNS settings, i.e.
the amplifier is armed.

## Your task

Defuse the amplifier without losing cluster DNS:

1. Look at what the pod's resolver is actually configured to do.
2. Count the fan-out yourself (debug a lookup if you like).
3. Fix the Deployment so external lookups cost **one** query: keep
   `dnsPolicy: ClusterFirst`, but set `ndots` to `1` (or `2`) via
   `spec.template.spec.dnsConfig`.

```sh
POD=$(kubectl -n kubelings get pods -l app=checkout -o name | head -1)
kubectl -n kubelings exec ${POD#pod/} -- cat /etc/resolv.conf
```

<details>
<summary>Hint</summary>

`dnsConfig.options` merges into the generated resolv.conf. Patch the pod template:

```sh
kubectl -n kubelings patch deploy checkout --type=strategic -p '
spec:
  template:
    spec:
      dnsConfig:
        options:
          - name: ndots
            value: "1"
'
```

Wait for the rollout, then re-read `/etc/resolv.conf` in a new pod — `ndots:1`,
search path intact.

</details>

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## Root cause chain (the real one)

1. **Trigger:** downstream timeout → aggregation-layer retries → traffic spike.
2. **Multiplier 1:** no application-level DNS caching → every request resolves.
3. **Multiplier 2:** `ndots:5` + 3-domain search path → ~10 queries per external
   lookup (each search candidate, A + AAAA).
4. **Weak link:** CoreDNS memory limit `100Mi` → OOMKilled under the flood —
   *all replicas at once*, because they all saw the same load.
5. **Blindfold:** monitoring depended on cluster DNS → no automatic paging.

No single step was a bug. The outage lived in the multiplication.

## Fix (this lesson)

```sh
kubectl -n kubelings patch deploy checkout --type=strategic -p '
spec:
  template:
    spec:
      dnsConfig:
        options:
          - name: ndots
            value: "1"
'
kubectl -n kubelings rollout status deploy/checkout
```

`ndots:1` → `payments.example.com` (2 dots ≥ 1) is queried as-is, first try.
Cluster-internal short names (`api`, `web.kubelings`) still work — they go
through the search path as before.

Alternative with zero pod changes: always use FQDNs with a **trailing dot**
(`payments.example.com.`) — the dot marks the name absolute and skips the search
path entirely.

## What Zalando actually did

- **Resilient DNS:** node-local dnsmasq caching in front of CoreDNS (today
  you'd use [NodeLocal DNSCache](https://kubernetes.io/docs/tasks/administer-cluster/nodelocaldns/)),
  so a query flood hits a per-node cache, not the central pods.
- **Right-sized CoreDNS** memory + kept it off aggressive limits.
- **External monitoring** that doesn't need cluster DNS to page a human.

## Prevention checklist

- Audit `ndots` for workloads that talk to external names in volume.
- Cache DNS in the app or at the node; never let each request pay for a lookup.
- Give CoreDNS headroom and alert on its memory *before* the OOMKill.
- Monitoring must not share fate with the thing it monitors.

</details>
