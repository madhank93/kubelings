---
kind: unit
title: "kubeadm: a cluster from three commands"
name: kubeadm-bootstrap-unit
---


> **☁ iximiuz Labs only.** kubeadm runs as root on real machines — host-level,
> outside the kubectl sandbox — so this can't run on your local `kind` cluster.
> Read the full `init → join` runbook below; then, since this playground is
> *already* a live cluster (you can't re-`init` one), the drill at the bottom
> exercises the **join half for real**: `init` tears worker `node-02` out, and
> you bring it back with a fresh `kubeadm join`. Upgrades are deliberately *not*
> here — that's M8.7 (`upgrade-runbook`), same tool, different day.

## What kubeadm is (and isn't)

Everything M7 taught — apiserver, etcd, scheduler, controller-manager,
kubelet (7.4) — has to get onto machines somehow. **kubeadm** is the
official minimal way: it bootstraps a control plane and joins nodes,
generating the certificates, kubeconfigs, and static-pod manifests you've
been reading all module. It does **not** install an OS, a container
runtime, a CNI, or manage machines afterward (that's the platform layer —
Cluster API, M10.5, drives kubeadm for you at fleet scale).

Prerequisites on every node, control plane or worker: a container runtime
(containerd), `kubelet` + `kubeadm` + `kubectl` packages, swap off, and the
`br_netfilter`/ip-forwarding sysctls the install docs list.

## 1 · `kubeadm init` — the control plane

```sh
kubeadm init --pod-network-cidr=10.244.0.0/16
```

That flag deserves its own sentence: it's the address space every pod IP
comes from (M4.10's per-node slices are carved from it). It must not
collide with your VPC/host networks, **your CNI must agree with it**
(Flannel's manifest defaults to exactly `10.244.0.0/16`), and changing it
later is effectively a cluster rebuild. The one irreversible decision in
the ceremony.

Watch the phase output scroll — it's this module in order:

```
[certs]           CA + component certs into /etc/kubernetes/pki
[kubeconfig]      admin.conf, kubelet.conf, … (cert-wrapped identities)
[etcd]            static-pod manifest for local etcd        ← 7.3's database
[control-plane]   apiserver/scheduler/controller-manager    ← static pods (7.4)
                  manifests into /etc/kubernetes/manifests/
[bootstrap-token] join tokens + the cluster-info ConfigMap
[addons]          CoreDNS (pending until CNI!) + kube-proxy DaemonSet
```

The kubelet was already running as a systemd service; it sees the manifest
files appear and starts the control plane — nobody `kubectl apply`s the
apiserver; there's no API to apply it to yet. Chicken, meet static-pod egg.

## 2 · The kubeconfig copy (the step everyone fumbles once)

```sh
mkdir -p $HOME/.kube
sudo cp /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
kubectl get nodes
# control-plane   NotReady   control-plane   1m   v1.31.x
```

`admin.conf` is cluster-admin (M4.11 taught you to treat such files as
standing credentials). And yes — **NotReady is correct**:

## 3 · The CNI step (why NotReady is by design)

M4.10 explained this exactly: no conflist in `/etc/cni/net.d/` → kubelet
reports `NetworkPluginNotReady` → node NotReady, CoreDNS Pending. Install a
CNI whose pod CIDR matches step 1:

```sh
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
kubectl get nodes -w     # NotReady → Ready as the DaemonSet lands
```

(Pin the manifest version in anything real — the course's own install
policy.)

## 4 · `kubeadm join` — workers

`init`'s last lines printed the ticket:

```sh
kubeadm join 10.0.0.10:6443 --token abcdef.0123456789abcdef \
  --discovery-token-ca-cert-hash sha256:1234…
```

Two credentials doing two jobs: the **token** proves the node may join
(bootstrap-token auth, expires in 24h); the **ca-cert-hash** proves the node
is joining the *right* cluster — it pins the CA so a MITM can't hand your
kubelet a fake control plane. Lost the ticket? Mint a fresh one:

```sh
kubeadm token create --print-join-command
```

On the worker, join runs the short version of init's phases: discovery →
TLS bootstrap (the kubelet requests a client cert via CSR — the
`Node,RBAC` scoping from M6.8 §3 starts here) → kubelet.conf written →
kubelet starts, CNI DaemonSet schedules onto it, node goes Ready.

```sh
kubectl get nodes
# control-plane   Ready    control-plane   12m
# worker-1        Ready    <none>          1m
```

## Reading a bootstrap that went sideways

| Symptom | Almost always |
|---|---|
| `init` hangs at `[wait-control-plane]` | runtime dead or kubelet can't start static pods — `journalctl -u kubelet`, `crictl ps -a` |
| node stays NotReady after join | no/broken CNI (M4.10 triage table) |
| CoreDNS Pending forever | same — CNI |
| `join` fails with x509/token errors | token expired (24h!) or wrong ca-cert-hash — mint a new join command |
| pods can't cross nodes | pod-network-cidr ↔ CNI manifest mismatch — the step-1 decision biting |

`kubeadm reset` on a node undoes its half (then clean `/etc/cni/net.d/` and
iptables yourself — reset tells you what it left behind).

## Takeaway

- The ceremony is: `init` → copy kubeconfig → CNI → `join` ×N. Four moves,
  in that order, CNI is not optional.
- `--pod-network-cidr` is the irreversible flag: match it to the CNI, keep
  it off your real networks.
- Join security = token (may I?) + ca-cert-hash (is this really you?);
  tokens expire by design — `kubeadm token create --print-join-command` is
  the reflex.
- Everything kubeadm writes, this module already taught you to read:
  `/etc/kubernetes/pki`, static-pod manifests, kubeconfigs. The magic is
  file placement.
- CKA's Installation & Configuration domain (~25%) is this page plus the
  upgrade runbook (M8.7) and HA (next lesson).

## Your turn

You can't `kubeadm init` a cluster that's already running — so this drill is
the other half of the ceremony, the one you actually repeat in production:
**join**. `init` ran `kubeadm reset` on worker `node-02` and stopped its
kubelet. From `cplane-01` it now reads NotReady and then disappears
altogether — no `kubelet.conf`, no PKI, no membership.

Bring it back Ready, using the real bootstrap-token flow:

1. On **cplane-01**, mint a join ticket — the printed command carries a fresh
   token *and* the CA cert hash: `kubeadm token create --print-join-command`.
2. Run that `kubeadm join …` line on **node-02**, as root.
3. Watch node-02 go from gone → NotReady → Ready as the CNI DaemonSet lands.

The check runs on node-02 and passes once it holds a *freshly issued*
`kubelet.conf` (newer than the reset) and the cluster reports it Ready.

<details>
<summary>Hint</summary>

Two machines, two roles — the same split the runbook describes.

The **token expires in 24h**, so don't hunt for the one `init` printed when
this cluster was built; mint a new one. `--print-join-command` hands you the
whole line, both credentials already filled in:

```sh
# on cplane-01
kubeadm token create --print-join-command
# -> kubeadm join <api>:6443 --token <fresh> --discovery-token-ca-cert-hash sha256:<hash>
```

```sh
# on node-02, as root — paste that exact line
kubeadm join <api>:6443 --token <fresh> --discovery-token-ca-cert-hash sha256:<hash>
```

You do **not** re-install a CNI: it's a DaemonSet that already exists
cluster-wide and schedules onto node-02 the moment it registers — which is
why NotReady flips to Ready on its own after a few seconds. Watch from
cplane-01: `kubectl get nodes -w`.

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


```sh
# --- on cplane-01: mint a fresh join command (the init-time token is long expired)
kubeadm token create --print-join-command
# copy the printed 'kubeadm join …' line

# --- on node-02, as root: run exactly that line
kubeadm join 10.0.0.10:6443 --token abcdef.0123456789abcdef \
  --discovery-token-ca-cert-hash sha256:1234…
# [preflight] … [kubelet-start] … 'This node has joined the cluster'

# --- back on cplane-01: watch it come Ready (CNI DaemonSet lands on its own)
kubectl get nodes -w
# node-02   NotReady   <none>   5s    ->   Ready   <none>   25s
```

## Root cause, restated

There was no failure to diagnose here — the point is muscle memory for the
one kubeadm operation you genuinely repeat: adding a node. Everything that
made it work is something the module already taught you to see. The **token**
answered "may this machine join?" and the **ca-cert-hash** answered "is this
the right cluster?" — two credentials, two questions. The node's kubelet then
requested its own client cert via CSR (the TLS bootstrap, `Node,RBAC` scoped),
wrote `/etc/kubernetes/kubelet.conf`, and started. And you touched the CNI
exactly zero times: it's a DaemonSet, so it found the new node by itself —
the same reason "NotReady until a CNI exists" from M4.10 runs in reverse here.

</details>
