---
kind: unit
title: "Incident replay — the exposed dashboard (JW Player's cryptominer)"
name: incident-cryptominer-unit
---


## The real incident

**JW Player** found a cryptocurrency miner running on their internal Kubernetes
clusters. Not a sophisticated nation-state operation — an **internal tool
(Weave Scope) exposed to the public internet via a cloud LoadBalancer with no
authentication.** Scope offers a UI to launch commands in containers. To an
attacker scanning for open dashboards, that's a free "run my miner here" button
against someone else's compute bill.

Source: [How a cryptocurrency miner made its way onto our internal Kubernetes clusters — JW Player](https://medium.com/jw-player-engineering/how-a-cryptocurrency-miner-made-its-way-onto-our-internal-kubernetes-clusters-9b09c4704205)

The lesson is unglamorous and universal: **the breach wasn't a Kubernetes
vulnerability — it was exposure plus missing auth.** `type: LoadBalancer` /
`NodePort` on an internal tool is a door to the internet, and the internet scans
every door continuously.

## This cluster, right now

Two findings sit in `kubelings`:

- `ops-console` — an internal tool on `NodePort 31337`. Public. No auth.
- `sys-helper` — a Deployment in no git repo, its pods busy-looping ("hashing"),
  labeled `workload=xmrig-lookalike`. The intruder.

```sh
kubectl -n kubelings get deploy,svc -o wide
kubectl -n kubelings get pods --show-labels
```

## Your task

Incident response, correct order:

1. **Contain the active threat:** remove the `sys-helper` miner workload.
2. **Close the entry:** stop `ops-console` being publicly reachable — make it
   `ClusterIP` (or delete the Service). Kill the process *then* the door, so the
   attacker can't just redeploy while you're cleaning up.

<details>
<summary>Hint</summary>

```sh
# 1. evict the intruder
kubectl -n kubelings delete deploy sys-helper
# 2. shut the public door
kubectl -n kubelings patch svc ops-console --type=merge -p '{"spec":{"type":"ClusterIP"}}'
```

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


## Response order matters

Real IR sequence, mirrored by this lesson:

1. **Contain** — stop the bleeding (kill the miner). If you closed the door
   first but left the miner, it keeps burning compute and may re-establish.
2. **Close the vector** — the exposed Service. Do this immediately after
   containment or the attacker re-enters through the same door mid-cleanup.
3. *(Real world, beyond this lab)* **Eradicate & investigate** — rotate every
   credential the compromised pods could reach, audit for persistence
   (extra RBAC bindings, cron pods, mutating webhooks), preserve logs, dig for
   *how* far in they got. A miner is often the *loudest* thing an attacker does,
   not the only thing.

## The defenses that would have stopped it (layers)

- **Exposure hygiene:** internal tools never get `LoadBalancer`/`NodePort`.
  Reach them via `kubectl port-forward` or an authenticated ingress. Audit:
  `kubectl get svc -A | grep -vE 'ClusterIP|kube-system'`.
- **Auth in front of every UI** — no unauthenticated dashboard, ever, internal
  or not.
- **RBAC least-privilege (previous lesson):** even having landed, the miner's
  SA should be able to do almost nothing.
- **Pod Security / NetworkPolicy (next lessons):** block privileged pods and
  egress to mining pools, so the payload fails even if launched.
- **Runtime detection:** a pod pinning CPU with an unknown image + a workload
  absent from git = two alertable signals.

## Why this lives in Security, not Networking

The NodePort is Module 4 mechanics — but the *failure* is a security-model gap:
"internal" was assumed, never enforced. Every exposure decision is a security
decision. The [Incident Library](https://kubelings.madhan.app/reference/incident-library/)
has more of these; they nearly all reduce to *a door someone forgot was a door.*

</details>
