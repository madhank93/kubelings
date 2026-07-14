---
kind: unit
title: "HA control plane: three of everything"
name: ha-control-plane-unit
---


> **Reading.** Real HA needs multiple machines and a load balancer —
> iximiuz multi-VM territory. The kind snippet at the end validates the
> *concepts* on a laptop. Builds on `kubeadm-bootstrap` (7.10); read that
> first.

## Why one control-plane node is a countdown

Lesson 7.10's cluster has one control-plane node. When it dies: running
pods keep running (kubelets and containers don't need permission to
continue) — but nothing can *change*. No rescheduling of failed pods, no
scaling, no deploys, no kubectl. Monzo's cascade (M9.1) and every
control-plane war story in M9 turn on this: the control plane is the
cluster's ability to *react*.

HA means surviving the loss of a control-plane node with reaction intact.
The recipe is three-of-everything — but each component multiplies
differently:

| Component | Multiplies how |
|---|---|
| etcd | Raft quorum — **odd counts only**: 3 tolerates 1 loss, 5 tolerate 2 (M7.3/7.6 taught why: majority writes) |
| kube-apiserver | freely — stateless, all active, behind one address |
| scheduler / controller-manager | all running, **one working** — leader election |

## Decision 1 — stacked or external etcd

**Stacked** (default): etcd runs as a static pod on each control-plane
node. 3 nodes = 3 apiservers + 3 etcd members, co-located. One machine
down = one apiserver *and* one etcd member down — fine at 3, but the fates
are coupled, and a second loss kills quorum, not just capacity.

**External**: etcd gets its own 3 machines; control-plane nodes run only
apiserver/scheduler/controller-manager and point at the remote etcd
(`kubeadm init --config` with an `external.etcd` block, certs included).
Decoupled failure domains, double the machines. Managed clouds run this
shape for you; self-managed shops mostly run stacked and accept the
coupling.

## Decision 2 — the endpoint (irreversible, again)

Every kubelet, every kubeconfig, every join ticket names *one* API address.
On a single-CP cluster that's the node's IP — which is exactly the problem:
certs and configs bake it in. HA needs a **stable virtual endpoint** from
day one:

```sh
kubeadm init \
  --control-plane-endpoint "api.cluster.example:6443" \
  --upload-certs \
  --pod-network-cidr=10.244.0.0/16
```

Rule: **always set `--control-plane-endpoint`, even for a single node** —
it's a DNS name you can repoint; retrofitting one later means re-issuing
certs across the cluster. 7.10 had one irreversible flag; this is the
second.

`--upload-certs` stashes the CA bundle encrypted in a Secret so joining CP
nodes can fetch it (expires after 2h; `kubeadm init phase upload-certs
--upload-certs` re-arms it).

## Who answers on that endpoint

The endpoint must resolve to *some healthy apiserver*:

- **Cloud**: an L4 load balancer with a `/readyz` health check — done.
- **On-prem classic**: **keepalived** (one VIP floats between CP nodes via
  VRRP) + **HAProxy** (spreads TCP across all three apiservers). Runs as
  static pods or systemd units on the CP nodes themselves.
- Skipping the LB and pointing DNS at one node's IP quietly rebuilds the
  single point of failure with extra steps — the most common fake-HA.

## Joining the other two

```sh
kubeadm join api.cluster.example:6443 --token … \
  --discovery-token-ca-cert-hash sha256:… \
  --control-plane --certificate-key <from --upload-certs>
```

Same join as a worker plus `--control-plane`: the node pulls the CA bundle,
generates its component certs, writes its static-pod manifests, and (if
stacked) its etcd member joins the Raft cluster. Repeat once more; quorum
at three.

```sh
kubectl get nodes -l node-role.kubernetes.io/control-plane
kubectl -n kube-system exec etcd-cp1 -- etcdctl member list …   # 3 members
```

## The singletons: leader election

Three schedulers can't all place pods — they'd fight (7.2 showed one
scheduler binding; imagine three racing). Scheduler and
controller-manager run **active-passive**: each replica tries to hold a
**Lease**:

```sh
kubectl -n kube-system get leases
# kube-scheduler            cp2_0a4f…   (holder = current leader)
# kube-controller-manager   cp1_9be2…
```

The holder renews every few seconds (those renewals are the
`system-leader-election` APF lane from 7.8 — now you know why it outranks
everything). Leader dies → lease expires (~15s) → another replica acquires
it → reconciliation resumes. Watch a failover live: `kubectl get lease
kube-scheduler -n kube-system -w` while stopping the leader's kubelet.

## Concept validation on a laptop

kind assembles the whole shape — 3 CP nodes, embedded HAProxy as the
endpoint — in one file:

```yaml
# ha.yaml — kind: 3 control-plane + 2 workers
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: control-plane
  - role: control-plane
  - role: worker
  - role: worker
```

```sh
kind create cluster --name ha --config ha.yaml
docker ps | grep haproxy          # the "load balancer"
kubectl -n kube-system get pods | grep -E 'etcd|apiserver'   # three of each
kubectl -n kube-system get leases # the singletons' election
```

Kill a control-plane container and watch: kubectl still answers (HAProxy
routes around it), leases change holders, etcd stays writable — 2 of 3 is
still a majority.

## Takeaway

- HA = odd etcd count (quorum) + N stateless apiservers (LB) + leader
  election for the singletons. Three different replication strategies, one
  cluster.
- `--control-plane-endpoint` on day one, always — the second irreversible
  flag.
- Stacked etcd couples failure domains; external decouples them at double
  the machine cost. Know which one you're running before the incident.
- A VIP/LB nobody health-checks, or DNS pinned to one node, is fake HA —
  test by actually killing a control-plane node in staging.
- CKA asks the topology diagram + the join flags; M9's cascades show what
  the topology is *for*.
