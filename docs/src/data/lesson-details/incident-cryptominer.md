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
